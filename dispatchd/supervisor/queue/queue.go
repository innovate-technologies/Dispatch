package queue

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/innovate-technologies/Dispatch/dispatchd/etcdcache"

	"github.com/innovate-technologies/Dispatch/dispatchd/config"
	"github.com/innovate-technologies/Dispatch/dispatchd/etcdclient"
	"github.com/innovate-technologies/Dispatch/dispatchd/unit"
	"github.com/innovate-technologies/Dispatch/dispatchd/unit/template"

	"strconv"

	etcd "github.com/coreos/etcd/clientv3"
	"golang.org/x/net/context"
)

var (
	ctx     = context.Background()
	etcdAPI = etcdclient.GetEtcdv3()
	// Config is a pointer need to be set to the main configuration
	Config      *config.ConfigurationInfo
	queueMutex  = &sync.Mutex{}
	assignMutex = &sync.Mutex{}
)

// Run checks for waiting units and assigns them
func Run() {
	importExisting()
	go watchQueue()
	go checkQueue() // make sure to not forget the unsatisfiable
}

// AddUnit adds a unit to the queue
func AddUnit(name string) {
	etcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/queue/%s", Config.Zone, name), name)
	etcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/units/%s/machine", Config.Zone, name), "")
}

func checkQueue() {
	for {
		time.Sleep(5 * time.Second)
		importExisting()
	}
}

func importExisting() {
	queueMutex.Lock()
	response, err := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/%s/queue/", Config.Zone), etcd.WithPrefix())
	if err != nil {
		return
	}
	for _, kv := range response.Kvs {
		go assignUnit(string(kv.Value))
	}
	queueMutex.Unlock()
}

func watchQueue() {
	chans := etcdAPI.Watch(context.Background(), fmt.Sprintf("/dispatch/%s/queue", Config.Zone), etcd.WithPrefix())
	for resp := range chans {
		for _, ev := range resp.Events {
			if ev.IsCreate() {
				queueMutex.Lock()
				go assignUnit(string(ev.Kv.Value))
				queueMutex.Unlock()
			}
		}
	}
}

func assignUnit(name string) {
	assignMutex.Lock()
	fmt.Println(name)
	newUnit := unit.NewFromEtcd(name)
	machine := getMachineForUnitConstraints(newUnit)
	if machine != "" {
		assignUnitToMachine(name, machine)
	}
	assignMutex.Unlock()
}

type machineInfoContent struct {
	Load  float64
	Units []string
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
	response, err := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/%s/machines/", Config.Zone), etcd.WithPrefix())
	if err != nil {
		fmt.Println(err)
		return "" //not important for now
	}

	machineInfo := map[string]machineInfoContent{}

	for _, key := range response.Kvs {
		keyParts := strings.Split(string(key.Key), "/")
		machineName := keyParts[4]
		if keyParts[5] == "load" {
			if _, ok := machineInfo[machineName]; !ok {
				machineInfo[machineName] = machineInfoContent{}
			}
			info := machineInfo[machineName]
			info.Load, _ = strconv.ParseFloat(string(key.Value), 64)
			machineInfo[machineName] = info
		}

		if keyParts[5] == "units" {
			if _, ok := machineInfo[machineName]; !ok {
				machineInfo[machineName] = machineInfoContent{}
			}
			info := machineInfo[machineName]
			info.Units = append(info.Units, string(key.Value))
			machineInfo[machineName] = info
		}
	}

	unitCache, _ := etcdcache.NewForPrefix(fmt.Sprintf("/dispatch/%s/units/", Config.Zone))

	for machine := range machineInfo {
		goCount := 0
		unitChan := make(chan unit.Unit)
		units := []unit.Unit{}
		for _, unitName := range machineInfo[machine].Units {
			go getUnit(unitName, unitCache, unitChan)
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
			machinesForLoad[machine] = machineInfo[machine].Load
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

func getUnit(name string, cache *etcdcache.EtcdCache, out chan unit.Unit) {
	out <- unit.NewFromEtcdWithCache(name, cache)
}

func assignUnitToMachine(unit, machine string) {
	etcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/machines/%s/units/%s", Config.Zone, machine, unit), unit)
	etcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/units/%s/machine", Config.Zone, unit), machine)
	etcdAPI.Delete(ctx, fmt.Sprintf("/dispatch/%s/queue/%s", Config.Zone, unit))
}
