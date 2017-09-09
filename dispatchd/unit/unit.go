package unit

import (
	"fmt"
	"log"
	"strings"

	"strconv"

	"github.com/spf13/afero"

	"github.com/innovate-technologies/Dispatch/dispatchd/config"
	"github.com/innovate-technologies/Dispatch/dispatchd/etcdclient"
	"github.com/innovate-technologies/Dispatch/dispatchd/unit/state"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/coreos/go-systemd/dbus"
	"golang.org/x/net/context"
)

const unitPath = "/var/run/dispatch/"

var (
	// Config is a pointer need to be set to the main configuration
	Config *config.ConfigurationInfo
	ctx    = context.Background()
	//EtcdAPI is the etcd keys api
	etcdAPI = etcdclient.GetEtcdv3()
	// DBusConnection is the connection to the system's D-Bus
	DBusConnection DBusConnectionInterface
	// FS is the file system to be used
	FS = afero.NewOsFs()
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
	etcdName     string
}

// GetAll returns all units in our zone
func GetAll() ([]Unit, error) {
	response, err := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/%s/units", Config.Zone), etcd.WithPrefix())
	if err != nil {
		return nil, err
	}

	units := []Unit{}
	hadUnitNames := map[string]bool{}

	for _, kv := range response.Kvs {
		pathParts := strings.Split(string(kv.Key), "/")
		if _, ok := hadUnitNames[pathParts[4]]; !ok {
			units = append(units, NewFromEtcd(pathParts[4]))
		}

	}

	return units, nil
}

// New returns a new Unit
func New() Unit {
	setUpDBus()
	return Unit{}
}

// NewFromEtcd creates a new unit with info from etcd
func NewFromEtcd(name string) Unit {
	if !strings.HasSuffix(name, ".service") {
		name += ".service"
	}

	unit := New()
	unit.etcdName = name
	unit.onEtcd = true
	unit.Name = getKeyFromEtcd(name, "name")
	unit.Machine = getKeyFromEtcd(name, "machine")
	unit.Template = getKeyFromEtcd(name, "template")
	unit.Global = getKeyFromEtcd(name, "global")
	unit.UnitContent = getKeyFromEtcd(name, "unit")
	unit.DesiredState = state.ForString(getKeyFromEtcd(name, "desiredState"))
	unit.State = state.ForString(getKeyFromEtcd(name, "state"))

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
		if unit.etcdName != "" {
			// Faulty unit
			unit.Name = unit.etcdName
			unit.Destroy()
		}
		return // can't create file without a name
	}
	thisUnitPath := unitPath + unit.Name

	fileContent := []byte(unit.UnitContent)

	FS.MkdirAll(unitPath, 0755)

	FS.Remove(thisUnitPath) // make sure the old unit is gone
	file, err := FS.Create(thisUnitPath)
	if err != nil {
		panic(err)
	}
	_, err = file.Write(fileContent)
	if err != nil {
		panic(err)
	}

	c := make(chan string)
	_, stopErr := DBusConnection.StopUnit(unit.Name, "fail", c) // stop unit to make sure new one is loaded
	if stopErr == nil {
		<-c // wait on completion
	}

	unit.onDisk = true
	_, dberr := DBusConnection.LinkUnitFiles([]string{thisUnitPath}, true, true)
	fmt.Println(dberr)
	DBusConnection.Reload()
}

// PutOnQueue places a specific unit on the queue
func (unit *Unit) PutOnQueue() {
	log.Println("Placing", unit.Name, "on queue")
	if unit.Global != "" {
		return
	}
	etcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/queue/%s", Config.Zone, unit.Name), unit.Name)
}

func (unit *Unit) normalizeName() {
	if !strings.HasSuffix(unit.Name, ".service") {
		unit.Name += ".service"
	}
}

// SaveOnEtcd saves the unit to etcd
func (unit *Unit) SaveOnEtcd() {
	unit.normalizeName()

	log.Println("Saving", unit.Name, "to etcd")

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
		etcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/globals/%s", Config.Zone, unit.Name), unit.Name)
	}

	unit.onEtcd = true
}

// Destroy destroys the given unit
func (unit *Unit) Destroy() {
	log.Println("Destroying unit", unit.Name)

	unit.Stop() // just making sure

	FS.Remove(unitPath + unit.Name)
	unit.onDisk = false
	DBusConnection.Reload()

	etcdAPI.Delete(ctx, fmt.Sprintf("/dispatch/%s/units/%s", Config.Zone, unit.Name), etcd.WithPrefix())
	etcdAPI.Delete(ctx, fmt.Sprintf("/dispatch/%s/queue/%s", Config.Zone, unit.Name), etcd.WithPrefix())
	etcdAPI.Delete(ctx, fmt.Sprintf("/dispatch/%s/machines/%s/units/%s", Config.Zone, Config.MachineName, unit.Name), etcd.WithPrefix())
	if unit.Global != "" {
		etcdAPI.Delete(ctx, fmt.Sprintf("/dispatch/%s/globals/%s", Config.Zone, unit.Name), etcd.WithPrefix())
	}
	unit.onEtcd = false
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
	chans := etcdAPI.Watch(context.Background(), fmt.Sprintf("/dispatch/%s/units/%s/desiredState", Config.Zone, unit.Name), etcd.WithPrefix())
	for resp := range chans {
		for _, ev := range resp.Events {
			if ev.IsModify() || ev.IsCreate() {
				unit.DesiredState = state.ForString(string(ev.Kv.Value))
				unit.becomeDesiredState()
			}
			if ev.Type == mvccpb.DELETE {
				break
			}
		}
	}
}

// WaitOnDestroy waits for the unit to enter a destroyed state
func (unit *Unit) WaitOnDestroy() {
	response, err := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/%s/units/%s", Config.Zone, unit.Name), etcd.WithPrefix())
	if err != nil || response.Count == 0 { // Destroyed already
		return
	}

	chans := etcdAPI.Watch(context.Background(), fmt.Sprintf("/dispatch/%s/units/%s/name", Config.Zone, unit.Name), etcd.WithPrefix())
	for resp := range chans {
		for _, ev := range resp.Events {
			if ev.Type == mvccpb.DELETE {
				break
			}
		}
	}
}

// WaitOnState waits for the unit to enter a specific state
func (unit *Unit) WaitOnState(s state.State) {
	stateInfo, err := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/%s/units/%s/state", Config.Zone, unit.Name))
	if err != nil || stateInfo.Count == 0 { // hmmm
		return
	}
	if string(stateInfo.Kvs[0].Value) == s.String() {
		return
	}

	chans := etcdAPI.Watch(context.Background(), fmt.Sprintf("/dispatch/%s/units/%s/state", Config.Zone, unit.Name), etcd.WithPrefix())
	for resp := range chans {
		for _, ev := range resp.Events {
			if string(ev.Kv.Value) == s.String() {
				break
			}
		}
	}
}

// SetState sets the state of the unit and updates etcd
func (unit *Unit) SetState(s state.State) {
	if unit.Global != "" {
		return
	}
	unit.State = s
	etcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/units/%s/state", Config.Zone, unit.Name), s.String())
}

// SetDesiredState sets the desiredstate of the unit and updates etcd
func (unit *Unit) SetDesiredState(s state.State) {
	unit.DesiredState = s
	etcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/units/%s/desiredState", Config.Zone, unit.Name), s.String())
}

func getKeyFromEtcd(unit, key string) string {
	response, err := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unit, key))
	if err != nil || response.Count == 0 {
		return ""
	}
	return string(response.Kvs[0].Value)
}

func setKeyOnEtcd(unit, key, content string) {
	etcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unit, key), content)
}

func setUpDBus() {
	if DBusConnection == nil {
		var err error
		DBusConnection, err = dbus.NewSystemdConnection()
		if err != nil {
			panic(err)
		}
	}
}

// KillAllOldUnits makes sure all old Dispatch spawned unit files on the system are deleted
func KillAllOldUnits() {
	setUpDBus()

	FS.MkdirAll(unitPath, 0755) // maybe we have a first run
	files, err := afero.Afero{Fs: FS}.ReadDir(unitPath)
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		log.Println("Stopping unit", file.Name())
		DBusConnection.KillUnit(file.Name(), 9) // do we care at this point?
		FS.Remove(unitPath + file.Name())
	}

	DBusConnection.Reload()
}
