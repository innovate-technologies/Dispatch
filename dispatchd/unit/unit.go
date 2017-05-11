package unit

import (
	"fmt"
	"io/ioutil"
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
	Config         *config.ConfigurationInfo
	ctx            = context.Background()
	etcdAPI        etcd.KeysAPI
	dbusConnection *dbus.Conn
)

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
	response, err := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/units/%s", Config.Zone), &etcd.GetOptions{})
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
	if dbusConnection == nil {
		var err error
		dbusConnection, err = dbus.NewSystemdConnection()
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
	unit.Name = name
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
	unit.SetState(state.Starting)
	c := make(chan string)
	dbusConnection.StartUnit(unit.Name, "fail", c)
	result := <-c
	if result == "done" {
		unit.SetState(state.Active)
	} else {
		unit.SetState(state.Dead)
	}
}

// Stop stops the unit
func (unit *Unit) Stop() {
	c := make(chan string)
	dbusConnection.StopUnit(unit.Name, "fail", c)
	result := <-c
	if result == "done" {
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
	thisUnitPath := unitPath + unit.Name

	fileContent := []byte(unit.getKeyFromEtcd("unit"))
	err := ioutil.WriteFile(thisUnitPath, fileContent, 0644)
	if err != nil {
		panic(err)
	}
	c := make(chan string)
	dbusConnection.StopUnit(unit.Name, "fail", c) // stop unit to make sure new one is loaded
	<-c
	unit.onDisk = true
	dbusConnection.LinkUnitFiles([]string{thisUnitPath}, true, true)
	dbusConnection.Reload()
}

// PutOnQueue places a specific unit on the queue
func (unit *Unit) PutOnQueue() {
	etcdAPI.Set(ctx, fmt.Sprintf("/dispatch/queue/%s/%s/", Config.Zone, unit.Name), unit.Name, &etcd.SetOptions{})
}

// SaveOnEtcd saves the unit to etcd
func (unit *Unit) SaveOnEtcd() {
	setUpEtcd()

	setKeyOnEtcd(unit.Name, "name", unit.Name)
	setKeyOnEtcd(unit.Name, "unit", unit.UnitContent)
	setKeyOnEtcd(unit.Name, "template", unit.Template)
	setKeyOnEtcd(unit.Name, "global", unit.Global)
	setKeyOnEtcd(unit.Name, "desiredState", unit.DesiredState.String())

	portStrings := []string{}
	for port := range unit.Ports {
		portStrings = append(portStrings, strconv.Itoa(port))
	}
	setKeyOnEtcd(unit.Name, "ports", strings.Join(portStrings, ","))

	if unit.Global != "" {
		etcdAPI.Set(ctx, fmt.Sprintf("/dispatch/globals/%s/%s", Config.Zone, unit.Name), unit.Name, &etcd.SetOptions{})
	}

	unit.onEtcd = true
}

// Destroy destroys the given unit
func (unit *Unit) Destroy() {
	unit.Stop() // just making sure
	os.Remove(unitPath + unit.Name)
	unit.onDisk = false
	dbusConnection.Reload()
	etcdAPI.Delete(ctx, fmt.Sprintf("/dispatch/units/%s/%s", Config.Zone, unit.Name), &etcd.DeleteOptions{Recursive: true})
	if unit.Global != "" {
		etcdAPI.Delete(ctx, fmt.Sprintf("/dispatch/globals/%s/%s", Config.Zone, unit.Name), &etcd.DeleteOptions{})
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
	fmt.Println("desiredstate:", unit.DesiredState)
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
	w := etcdAPI.Watcher(fmt.Sprintf("/dispatch/units/%s/%s/desiredState", Config.Zone, unit.Name), &etcd.WatcherOptions{})
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

func (unit *Unit) getKeyFromEtcd(key string) string {
	response, err := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/units/%s/%s/%s", Config.Zone, unit.Name, key), &etcd.GetOptions{})
	if err != nil {
		return ""
	}
	return response.Node.Value
}

func (unit *Unit) SetState(s state.State) {
	if unit.Global != "" {
		return
	}
	unit.State = s
	etcdAPI.Set(ctx, fmt.Sprintf("/dispatch/units/%s/%s/state", Config.Zone, unit.Name), s.String(), &etcd.SetOptions{})
}

func setUpEtcd() {
	if etcdAPI != nil {
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
	etcdAPI = etcd.NewKeysAPI(c)
}

func getKeyFromEtcd(unit, key string) string {
	response, err := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/units/%s/%s/%s", Config.Zone, unit, key), &etcd.GetOptions{})
	if err != nil {
		return ""
	}
	return response.Node.Value
}

func setKeyOnEtcd(unit, key, content string) {
	etcdAPI.Set(ctx, fmt.Sprintf("/dispatch/templates/%s/%s/%s", Config.Zone, unit, key), content, &etcd.SetOptions{})
}
