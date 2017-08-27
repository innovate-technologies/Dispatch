package queue

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/innovate-technologies/Dispatch/dispatchd/config"
	"github.com/innovate-technologies/Dispatch/dispatchd/unit"
	"github.com/innovate-technologies/Dispatch/dispatchd/unit/template"

	"strconv"

	etcd "github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

var (
	ctx     = context.Background()
	etcdAPI etcd.KeysAPI
	// Config is a pointer need to be set to the main configuration
	Config     *config.ConfigurationInfo
	queueMutex = &sync.Mutex{}
)

// Run checks for waiting units and assigns them
func Run() {
	setUpEtcd()
	importExisting()
	go watchQueue()
	go checkQueue() // make sure to not forget the unsatisfiable
}

// AddUnit adds a unit to the queue
func AddUnit(name string) {
	if etcdAPI == nil {
		setUpEtcd()
	}
	etcdAPI.Set(ctx, fmt.Sprintf("/dispatch/%s/queue/%s", Config.Zone, name), name, &etcd.SetOptions{})
	etcdAPI.Set(ctx, fmt.Sprintf("/dispatch/%s/units/%s/machine", Config.Zone, name), "", &etcd.SetOptions{})
}

func checkQueue() {
	for {
		time.Sleep(5 * time.Second)
		importExisting()
	}
}

func importExisting() {
	queueMutex.Lock()
	response, err := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/%s/queue/", Config.Zone), &etcd.GetOptions{})
	if err != nil {
		return
	}
	for _, node := range response.Node.Nodes {
		go assignUnit(node.Value)
	}
	queueMutex.Unlock()
}

func watchQueue() {
	w := etcdAPI.Watcher(fmt.Sprintf("/dispatch/%s/queue/", Config.Zone), &etcd.WatcherOptions{Recursive: true})
	for {
		r, err := w.Next(ctx)
		if err != nil {
			go watchQueue()
			return
		}
		queueMutex.Lock()
		if r.Action == "set" {
			go assignUnit(r.Node.Value)
		}
		queueMutex.Unlock()
	}
}

func assignUnit(name string) {
	fmt.Println(name)
	newUnit := unit.NewFromEtcd(name)
	machine := getMachineForUnitConstraints(newUnit)
	if machine != "" {
		assignUnitToMachine(name, machine)
	}
}

func getMachineForUnitConstraints(u unit.Unit) string {
	// contraints := u.Constraints
	ports := u.Ports
	var unitTemplate template.Template
	if u.Template != "" {
		unitTemplate = template.NewFromEtcd(u.Template)
	}

	// TO DO implement constraints
	machinesForLoad := map[string]float64{}
	response, err := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/%s/machines/", Config.Zone), &etcd.GetOptions{Recursive: true})
	if err != nil {
		fmt.Println(err)
		return "" //not important for now
	}
	for _, machine := range response.Node.Nodes {
		machineNameParts := strings.Split(machine.Key, "/")

		machineName := machineNameParts[len(machineNameParts)-1]
		var load float64
		unitNames := []string{}

		for _, key := range machine.Nodes {
			keyParts := strings.Split(key.Key, "/")
			if keyParts[len(keyParts)-1] == "load" {
				load, _ = strconv.ParseFloat(key.Value, 64)
			}
			if keyParts[len(keyParts)-1] == "units" {
				for _, unit := range key.Nodes {
					unitNames = append(unitNames, unit.Value)
				}
			}
		}

		goCount := 0
		unitChan := make(chan unit.Unit)
		units := []unit.Unit{}
		for _, unitName := range unitNames {
			go getUnit(unitName, unitChan)
			goCount++
		}
		for goCount > 0 {
			unit := <-unitChan
			units = append(units, unit)
			goCount--
		}
		allPortsAvailable := true
		var numSameTemplate int64
	L:
		for _, unit := range units {
			// check template
			if unit.Template == u.Template {
				numSameTemplate++
			}

			// check ports
			for _, unitPort := range unit.Ports {
				for _, port := range ports {
					if unitPort == port {
						allPortsAvailable = false
						break L
					}
				}
			}
		}
		if allPortsAvailable && (numSameTemplate < unitTemplate.MaxPerMachine || unitTemplate.MaxPerMachine == 0) { // check ports and template constraints
			machinesForLoad[machineName] = load
		}
	}
	isCompared := false
	lowestLoad := 0.0
	lowestLoadMachine := ""
	for machine, load := range machinesForLoad {
		if load < lowestLoad || !isCompared {
			lowestLoad = load
			lowestLoadMachine = machine
			isCompared = true
		}
	}
	return lowestLoadMachine
}

func getUnit(name string, out chan unit.Unit) {
	out <- unit.NewFromEtcd(name)
}

func assignUnitToMachine(unit, machine string) {
	etcdAPI.Set(ctx, fmt.Sprintf("/dispatch/%s/machines/%s/units/%s", Config.Zone, machine, unit), unit, &etcd.SetOptions{})
	etcdAPI.Set(ctx, fmt.Sprintf("/dispatch/%s/units/%s/machine", Config.Zone, unit), machine, &etcd.SetOptions{})
	etcdAPI.Delete(ctx, fmt.Sprintf("/dispatch/%s/queue/%s", Config.Zone, unit), &etcd.DeleteOptions{})
}

func setUpEtcd() {
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
