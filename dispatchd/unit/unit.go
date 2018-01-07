package unit

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/innovate-technologies/Dispatch/dispatchd/etcdcache"

	"strconv"

	"github.com/spf13/afero"

	"github.com/innovate-technologies/Dispatch/dispatchd/config"
	"github.com/innovate-technologies/Dispatch/dispatchd/etcdclient"
	"github.com/innovate-technologies/Dispatch/dispatchd/unit/state"
	"github.com/innovate-technologies/Dispatch/interfaces"

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
	EtcdAPI interfaces.EtcdAPI = etcdclient.GetEtcdv3()
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
	etcdCache    *etcdcache.EtcdCache
	disableCache bool
	runContext   context.Context
	runCancel    context.CancelFunc
}

// GetAll returns all units in our zone
func GetAll() ([]Unit, error) {
	cache, err := etcdcache.NewForPrefix(fmt.Sprintf("/dispatch/%s/units", Config.Zone))
	if err != nil {
		return nil, err
	}

	units := []Unit{}
	hadUnitNames := map[string]bool{}

	for _, kv := range cache.GetAll() {
		pathParts := strings.Split(string(kv.Key), "/")
		if _, ok := hadUnitNames[pathParts[4]]; !ok {
			units = append(units, NewFromEtcd(pathParts[4]))
			hadUnitNames[pathParts[4]] = true
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
	fillFromEtcd(&unit, name)

	return unit
}

// NewFromEtcdWithCache creates a new unit with info from etcd using a specified cache
func NewFromEtcdWithCache(name string, cache *etcdcache.EtcdCache) Unit {
	if !strings.HasSuffix(name, ".service") {
		name += ".service"
	}
	unit := New()
	unit.etcdCache = cache
	fillFromEtcd(&unit, name)

	return unit
}

func fillFromEtcd(unit *Unit, name string) {
	unit.etcdName = name
	unit.onEtcd = true
	unit.Name = unit.getKeyFromEtcd("name")
	unit.Machine = unit.getKeyFromEtcd("machine")
	unit.Template = unit.getKeyFromEtcd("template")
	unit.Global = unit.getKeyFromEtcd("global")
	unit.UnitContent = unit.getKeyFromEtcd("unit")
	unit.DesiredState = state.ForString(unit.getKeyFromEtcd("desiredState"))
	unit.State = state.ForString(unit.getKeyFromEtcd("state"))

	unit.Ports = []int64{}
	portsStringArray := strings.Split(unit.getKeyFromEtcd("ports"), ",")
	for _, portString := range portsStringArray {
		port, err := strconv.ParseInt(portString, 10, 64)
		if err == nil {
			unit.Ports = append(unit.Ports, port)
		}
	}
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
	<-c
	unit.SetState(state.Active)
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
	unit.runContext, unit.runCancel = context.WithCancel(context.Background())

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
	if unit.Global != "" {
		return
	}
	log.Println("Placing", unit.Name, "on queue")
	EtcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/queue/%s", Config.Zone, unit.Name), unit.Name)
}

func (unit *Unit) normalizeName() {
	if !strings.HasSuffix(unit.Name, ".service") {
		unit.Name += ".service"
	}
}

// SaveOnEtcd saves the unit to etcd
func (unit *Unit) SaveOnEtcd() error {
	var err error
	unit.normalizeName()

	log.Println("Saving", unit.Name, "to etcd")

	err = doIfErrNil(err, unit.setKeyOnEtcd, "name", unit.Name)
	err = doIfErrNil(err, unit.setKeyOnEtcd, "unit", unit.UnitContent)
	err = doIfErrNil(err, unit.setKeyOnEtcd, "template", unit.Template)
	err = doIfErrNil(err, unit.setKeyOnEtcd, "desiredState", unit.DesiredState.String())

	portStrings := []string{}
	for _, port := range unit.Ports {
		portStrings = append(portStrings, strconv.FormatInt(port, 10))
	}
	err = doIfErrNil(err, unit.setKeyOnEtcd, "ports", strings.Join(portStrings, ","))

	if unit.Global != "" {
		unit.setKeyOnEtcd("global", unit.Global)
		_, err = EtcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/globals/%s", Config.Zone, unit.Name), unit.Name)
	}

	if err != nil {
		return err
	}
	unit.onEtcd = true
	return nil
}

// Destroy destroys the given unit
func (unit *Unit) Destroy() {
	log.Println("Destroying unit", unit.Name)
	if unit.runCancel != nil {
		unit.runCancel()
	}

	unit.Stop() // just making sure
	fmt.Println("unit stopped for destroy")

	FS.Remove(unitPath + unit.Name)
	unit.onDisk = false
	DBusConnection.Reload()

	if unit.Name == "" {
		fmt.Println("Destroying unit not on etcd")
		// oopsie
		unit.onEtcd = false //probably not
		return
	}

	EtcdAPI.Delete(ctx, fmt.Sprintf("/dispatch/%s/units/%s", Config.Zone, unit.Name), etcd.WithPrefix())
	EtcdAPI.Delete(ctx, fmt.Sprintf("/dispatch/%s/queue/%s", Config.Zone, unit.Name), etcd.WithPrefix())
	EtcdAPI.Delete(ctx, fmt.Sprintf("/dispatch/%s/machines/%s/units/%s", Config.Zone, unit.Machine, unit.Name), etcd.WithPrefix())
	if unit.Global != "" {
		EtcdAPI.Delete(ctx, fmt.Sprintf("/dispatch/%s/globals/%s", Config.Zone, unit.Name), etcd.WithPrefix())
	}
	unit.etcdCache = etcdcache.New() // clear out all old cache!
	unit.onEtcd = false
	fmt.Println("Destroy done")
}

// LoadAndWatch loads the unit to the system and follows the desired state
func (unit *Unit) LoadAndWatch() {
	unit.Create()
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
		fmt.Println("desiredState: found to be destroy")
		unit.Destroy()
	}
}

// Watch creates and etcd watcher for the desired state of a specific unit
func (unit *Unit) Watch() {
	chans := EtcdAPI.Watch(unit.runContext, fmt.Sprintf("/dispatch/%s/units/%s/desiredState", Config.Zone, unit.Name))
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
	fmt.Println("End watch for ", unit.Name)
}

// WaitOnDestroy waits for the unit to enter a destroyed state
func (unit *Unit) WaitOnDestroy() {
	unit.disableCache = true
	for unit.getKeyFromEtcd("name") != "" {
		time.Sleep(100 * time.Millisecond)
	}
}

// WaitOnState waits for the unit to enter a specific state
func (unit *Unit) WaitOnState(s state.State) {
	stateInfo, err := EtcdAPI.Get(ctx, fmt.Sprintf("/dispatch/%s/units/%s/state", Config.Zone, unit.Name))
	if err != nil || stateInfo.Count == 0 { // hmmm
		return
	}
	if string(stateInfo.Kvs[0].Value) == s.String() {
		return
	}

	chans := EtcdAPI.Watch(context.Background(), fmt.Sprintf("/dispatch/%s/units/%s/state", Config.Zone, unit.Name))
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
	if unit.Global != "" || !unit.isHealthy() {
		return
	}
	unit.State = s
	EtcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/units/%s/state", Config.Zone, unit.Name), s.String())
}

// SetDesiredState sets the desiredstate of the unit and updates etcd
func (unit *Unit) SetDesiredState(s state.State) {
	unit.DesiredState = s
	EtcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/units/%s/desiredState", Config.Zone, unit.Name), s.String())
}

func (unit *Unit) isHealthy() bool {
	unit.disableCache = true
	name := unit.getKeyFromEtcd("name")
	if name == "" {
		return false
	}
	return true
}

func (unit *Unit) getKeyFromEtcd(key string) string {
	if unit.etcdCache != nil {
		if kv, err := unit.etcdCache.Get(key); err == nil {
			return string(kv.Value)
		}
	}
	if unit.etcdName == "" && unit.Name != "" {
		unit.etcdName = unit.Name
	}
	response, err := EtcdAPI.Get(ctx, fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unit.etcdName, key))
	if err != nil || response.Count == 0 {
		return ""
	}
	return string(response.Kvs[0].Value)
}

func (unit *Unit) setKeyOnEtcd(key, content string) error {
	if unit.etcdCache != nil {
		unit.etcdCache.Invalidate(key)
	}
	if unit.etcdName == "" && unit.Name != "" {
		unit.etcdName = unit.Name
	}
	_, err := EtcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unit.etcdName, key), content)
	return err
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

func doIfErrNil(err error, f func(string, string) error, s1, s2 string) error {
	if err != nil {
		return err
	}
	err = f(s1, s2)
	return err
}
