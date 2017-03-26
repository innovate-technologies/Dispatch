package queue

import (
	"fmt"
	"strings"
	"time"

	"../../config"

	"strconv"

	etcd "github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

var (
	ctx     = context.Background()
	etcdAPI etcd.KeysAPI
	// Config is a pointer need to be set to the main configuration
	Config *config.ConfigurationInfo
)

// Run checks for waiting units and assigns them
func Run() {
	setUpEtcd()
	importExisting()
	go watchQueue()
}

// AddUnit adds a unit to the queue
func AddUnit(name string) {
	if etcdAPI == nil {
		setUpEtcd()
	}
	etcdAPI.Set(ctx, fmt.Sprintf("/dispatch/queue/%s/%s", Config.Zone, name), name, &etcd.SetOptions{})
}

func importExisting() {
	response, err := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/queue/%s/", Config.Zone), &etcd.GetOptions{})
	if err != nil {
		return
	}
	for _, node := range response.Node.Nodes {
		go assignUnit(node.Value)
	}
}

func watchQueue() {
	w := etcdAPI.Watcher(fmt.Sprintf("/dispatch/queue/%s/", Config.Zone), &etcd.WatcherOptions{Recursive: true})
	for {
		r, err := w.Next(ctx)
		if err != nil {
			go watchQueue()
			return
		}
		if r.Action == "set" {
			go assignUnit(r.Node.Value)
		}
	}
}

func assignUnit(name string) {
	fmt.Println(name)
	machine := getMachineForConstraints(map[string]string{}) // TO DO implement constraints
	assignUnitToMachine(name, machine)
}

func getMachineForConstraints(contraints map[string]string) string {
	machinesForLoad := map[string]float64{}
	response, err := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/machines/%s/", Config.Zone), &etcd.GetOptions{Recursive: true})
	if err != nil {
		return "" //not important for now
	}
	for _, node := range response.Node.Nodes {
		keyParts := strings.Split(node.Key, "/")
		if keyParts[len(keyParts)-1] == "load" {
			if load, err := strconv.ParseFloat(keyParts[len(keyParts)-1], 64); err == nil {
				// TO DO ADD CONTRAINTS
				machinesForLoad[keyParts[len(keyParts)-2]] = load
			}
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

func assignUnitToMachine(unit, machine string) {
	etcdAPI.Set(ctx, fmt.Sprintf("/dispatch/machines/%s/%s/units/%s", Config.Zone, machine, unit), unit, &etcd.SetOptions{})
	etcdAPI.Delete(ctx, fmt.Sprintf("/dispatch/queue/%s/%s", Config.Zone, unit), &etcd.DeleteOptions{})
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
