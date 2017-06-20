package unit

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"strconv"

	"github.com/innovate-technologies/Dispatch/dispatchd/config"
	"github.com/innovate-technologies/Dispatch/dispatchd/unit/state"

	etcd "github.com/coreos/etcd/client"
	"github.com/coreos/go-systemd/dbus"
	"golang.org/x/net/context"
)

const unitPath = "/var/run/dispatch/"

var (
	// Config is a pointer need to be set to the main configuration
	Config *config.ConfigurationInfo
	ctx    = context.Background()
	//EtcdAPI is the etcd keys api
	EtcdAPI etcd.KeysAPI
	// DBusConnection is the connection to the system's D-Bus
	DBusConnection DBusConnectionInterface
)

// UnitInterface is the interface to a Unit
type UnitInterface interface {
	Start()
	Stop()
	Restart()
	Create()
	PutOnQueue()
	SaveOnEtcd()
	Destroy()
	LoadAndWatch()
	Watch()
	SetState(state.State)
	SetDesiredState(s state.State)
}

// Unit is a struct containing all info of a specific unit
type Unit struct {
	Name         string `json:"name"`
	Machine      string `json:"machine"`
	Template     string `json:"template,omitempty"` // is set with template name if from Template
	Global       string `json:"global,omitempty"`   // is set with global name if from global
	State        state.State
	DesiredState state.State
	Ports        []int64 `json:"ports"`
	Constraints  map[string]string
	UnitContent  string `json:"unitContent"`
	onEtcd       bool
	onDisk       bool
}

// GetAll returns all units in our zone
func GetAll() ([]Unit, error) {
	setUpEtcd()
	response, err := EtcdAPI.Get(ctx, fmt.Sprintf("/dispatch/units/%s", Config.Zone), &etcd.GetOptions{})
	if err != nil {
		return nil, err
	}

	units := []Unit{}

	for _, unitNode := range response.Node.Nodes {
		pathParts := strings.Split(unitNode.Key, "/")
		units = append(units, NewFromEtcd(pathParts[len(pathParts)-1]))
	}

	return units, nil
}

// New returns a new Unit
func New() Unit {
	if DBusConnection == nil {
		var err error
		DBusConnection, err = dbus.NewSystemdConnection()
		if err != nil {
			panic(err)
		}
	}
	return Unit{}
}

// NewFromEtcd creates a new unit with info from etcd
func NewFromEtcd(name string) Unit {
	setUpEtcd()
	unit := New()
	unit.onEtcd = true
	unit.Name = getKeyFromEtcd(name, "name")
	unit.Machine = getKeyFromEtcd(name, "machine")
	unit.Template = getKeyFromEtcd(name, "template")
	unit.Global = getKeyFromEtcd(name, "global")
	unit.State = state.Dead
	unit.UnitContent = getKeyFromEtcd(name, "unit")
	unit.DesiredState = state.ForString(getKeyFromEtcd(name, "desiredState"))

	unit.Ports = []int64{}
	portsStringArray := strings.Split(getKeyFromEtcd(name, "ports"), ",")
	for _, portString := range portsStringArray {
		port, err := strconv.ParseInt(portString, 10, 64)
		if err == nil {
			unit.Ports = append(unit.Ports, port)
		}
	}

	return unit
}

// Start starts the specific unit
func (unit *Unit) Start() {
	log.Println("Starting unit", unit.Name)
	unit.SetState(state.Starting)
	c := make(chan string)
	_, err := DBusConnection.StartUnit(unit.Name, "fail", c)
	if err != nil {
		log.Println("Error starting unit", unit.Name, err)
		return
	}
	result := <-c
	if result == "done" {
		log.Println("Started unit", unit.Name)
		unit.SetState(state.Active)
	} else {
		log.Println("Failed starting unit", unit.Name)
		unit.SetState(state.Dead)
	}
}

// Stop stops the unit
func (unit *Unit) Stop() {
	log.Println("Stopping unit", unit.Name)
	c := make(chan string)
	_, err := DBusConnection.StopUnit(unit.Name, "fail", c)
	if err != nil {
		log.Println("Error stopping unit", unit.Name, err)
		log.Println("Killing unit", unit.Name)
		DBusConnection.KillUnit(unit.Name, 9) // the big guns!
		unit.SetState(state.Dead)
		return
	}
	result := <-c
	if result == "done" {
		log.Println("Stopped unit", unit.Name)
		unit.SetState(state.Dead)
	}
}

// Restart restarts the unit
func (unit *Unit) Restart() {
	unit.Stop()
	unit.Start()
}

// Create writes the unit to the disk
func (unit *Unit) Create() {
	if unit.Name == "" {
		log.Println("Error starting unit with no name")
		return // can't create file without a name
	}
	thisUnitPath := unitPath + unit.Name

	fileContent := []byte(getKeyFromEtcd(unit.Name, "unit"))

	os.Remove(thisUnitPath) // make sure the old unit is gone

	err := ioutil.WriteFile(thisUnitPath, fileContent, 0644)
	if err != nil {
		panic(err)
	}

	c := make(chan string)
	_, stopErr := DBusConnection.StopUnit(unit.Name, "fail", c) // stop unit to make sure new one is loaded
	if stopErr == nil {
		<-c // wait on completio,
	}

	unit.onDisk = true
	_, dberr := DBusConnection.LinkUnitFiles([]string{thisUnitPath}, true, true)
	fmt.Println(dberr)
	DBusConnection.Reload()
}

// PutOnQueue places a specific unit on the queue
func (unit *Unit) PutOnQueue() {
	log.Println("Placing", unit.Name, "on queue")
	EtcdAPI.Set(ctx, fmt.Sprintf("/dispatch/queue/%s/%s", Config.Zone, unit.Name), unit.Name, &etcd.SetOptions{})
}

// SaveOnEtcd saves the unit to etcd
func (unit *Unit) SaveOnEtcd() {
	log.Println("Saving", unit.Name, "to etcd")
	setUpEtcd()

	setKeyOnEtcd(unit.Name, "name", unit.Name)
	setKeyOnEtcd(unit.Name, "unit", unit.UnitContent)
	setKeyOnEtcd(unit.Name, "template", unit.Template)
	setKeyOnEtcd(unit.Name, "desiredState", unit.DesiredState.String())

	portStrings := []string{}
	for _, port := range unit.Ports {
		portStrings = append(portStrings, strconv.FormatInt(port, 10))
	}
	setKeyOnEtcd(unit.Name, "ports", strings.Join(portStrings, ","))

	if unit.Global != "" {
		setKeyOnEtcd(unit.Name, "global", unit.Global)
		EtcdAPI.Set(ctx, fmt.Sprintf("/dispatch/globals/%s/%s", Config.Zone, unit.Name), unit.Name, &etcd.SetOptions{})
	}

	unit.onEtcd = true
}

// Destroy destroys the given unit
func (unit *Unit) Destroy() {
	log.Println("Destroying unit", unit.Name)

	unit.Stop() // just making sure

	os.Remove(unitPath + unit.Name)
	unit.onDisk = false
	DBusConnection.Reload()

	EtcdAPI.Delete(ctx, fmt.Sprintf("/dispatch/units/%s/%s", Config.Zone, unit.Name), &etcd.DeleteOptions{Recursive: true})
	EtcdAPI.Delete(ctx, fmt.Sprintf("/dispatch/machines/%s/%s/units/%s", Config.Zone, Config.MachineName, unit.Name), &etcd.DeleteOptions{Recursive: true})
	if unit.Global != "" {
		EtcdAPI.Delete(ctx, fmt.Sprintf("/dispatch/globals/%s/%s", Config.Zone, unit.Name), &etcd.DeleteOptions{})
	}
}

// LoadAndWatch loads the unit to the system and follows the desired state
func (unit *Unit) LoadAndWatch() {
	if !unit.onDisk {
		unit.Create()
	}
	unit.becomeDesiredState()
	go unit.Watch()
}

func (unit *Unit) becomeDesiredState() {
	fmt.Println("desiredState:", unit.DesiredState)
	if unit.DesiredState == state.Active {
		unit.Start()
	} else if unit.DesiredState == state.Dead {
		unit.Stop()
	} else if unit.DesiredState == state.Destroy {
		unit.Destroy()
	}
}

// Watch creates and etcd watcher for the desired state of a specific unit
func (unit *Unit) Watch() {
	w := EtcdAPI.Watcher(fmt.Sprintf("/dispatch/units/%s/%s/desiredState", Config.Zone, unit.Name), &etcd.WatcherOptions{})
	for {
		r, err := w.Next(ctx)
		if err != nil {
			go unit.Watch()
			return
		}
		if r.Action == "set" {
			unit.DesiredState = state.ForString(r.Node.Value)
			unit.becomeDesiredState()
		}
	}
}

// SetState sets the state of the unit and updates etcd
func (unit *Unit) SetState(s state.State) {
	if unit.Global != "" {
		return
	}
	unit.State = s
	EtcdAPI.Set(ctx, fmt.Sprintf("/dispatch/units/%s/%s/state", Config.Zone, unit.Name), s.String(), &etcd.SetOptions{})
}

// SetDesiredState sets the desiredstate of the unit and updates etcd
func (unit *Unit) SetDesiredState(s state.State) {
	unit.DesiredState = s
	EtcdAPI.Set(ctx, fmt.Sprintf("/dispatch/units/%s/%s/desiredState", Config.Zone, unit.Name), s.String(), &etcd.SetOptions{})
}

func setUpEtcd() {
	if EtcdAPI != nil {
		return
	}
	c, err := etcd.New(etcd.Config{
		Endpoints:               []string{Config.EtcdAddress},
		Transport:               etcd.DefaultTransport,
		HeaderTimeoutPerRequest: 10 * time.Second,
	})
	if err != nil {
		panic(err)
	}
	EtcdAPI = etcd.NewKeysAPI(c)
}

func getKeyFromEtcd(unit, key string) string {
	response, err := EtcdAPI.Get(ctx, fmt.Sprintf("/dispatch/units/%s/%s/%s", Config.Zone, unit, key), &etcd.GetOptions{})
	if err != nil {
		return ""
	}
	return response.Node.Value
}

func setKeyOnEtcd(unit, key, content string) {
	EtcdAPI.Set(ctx, fmt.Sprintf("/dispatch/units/%s/%s/%s", Config.Zone, unit, key), content, &etcd.SetOptions{})
}
