package unit

import (
	"fmt"
	"io/ioutil"
	"time"

	"../config"
	"./state"
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
	Name         string
	Machine      string
	State        state.State
	DesiredState state.State
	Ports        []int
	Constraints  map[string]string
	UnitContent  string
	onEtcd       bool
	onDisk       bool
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
	return Unit{onEtcd: false}
}

// NewFromEtcd creates a new unit with info from etcd
func NewFromEtcd(name string) Unit {
	setUpEtcd()
	unit := New()
	unit.onEtcd = true
	unit.Name = name
	unit.Machine = getKeyFromEtcd(name, "machine")
	unit.State = state.Dead
	unit.UnitContent = getKeyFromEtcd(name, "unit")
	unit.DesiredState = state.ForString(getKeyFromEtcd(name, "desiredState"))
	return unit
}

// Start starts the specific unit
func (unit *Unit) Start() {
	fmt.Println("TO DO start", unit.Name)
}

// Stop stops the unit
func (unit *Unit) Stop() {
	fmt.Println("TO DO")
}

// Restart restarts the unit
func (unit *Unit) Restart() {
	fmt.Println("TO DO")
}

// Create writes the unit to the disk
func (unit *Unit) Create() {
	thisUnitPath := unitPath + unit.Name

	fileContent := []byte(unit.getKeyFromEtcd("unit"))
	err := ioutil.WriteFile(thisUnitPath, fileContent, 0644)
	if err != nil {
		panic(err)
	}
	unit.onDisk = true
	dbusConnection.LinkUnitFiles([]string{thisUnitPath}, true, true)
}

// Destroy destroys the given unit
func (unit *Unit) Destroy() {
	fmt.Println("TO DO")
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
		fmt.Println(r)
	}
}

func (unit *Unit) getKeyFromEtcd(key string) string {
	response, err := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/units/%s/%s/%s", Config.Zone, unit.Name, key), &etcd.GetOptions{})
	if err != nil {
		return ""
	}
	return response.Node.Value
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
