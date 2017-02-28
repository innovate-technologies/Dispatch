package machine

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"../config"
	"../unit"

	etcd "github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

var (
	ctx             = context.Background()
	etcdAPI         etcd.KeysAPI
	machineLocation string
	// Config is a pointer need to be set to the main configuration
	Config *config.ConfigurationInfo
)

// RegisterMachine adds the machine to the cluster
func RegisterMachine() {
	setUpEtcd()

	unit.Config = Config // pass throug the config

	machineLocation = fmt.Sprintf("/dispatch/machines/%s/%s", Config.Zone, Config.MachineName)

	etcdAPI.Set(ctx, machineLocation+"/arch", Config.Arch, &etcd.SetOptions{})
	etcdAPI.Set(ctx, machineLocation+"/ip", Config.PublicIP, &etcd.SetOptions{})

	etcdAPI.Set(ctx, machineLocation+"/alive", "1", &etcd.SetOptions{TTL: 10 * time.Second})

	go renewAlive()
	go updateLoad()
	go startUnits()
}

func renewAlive() {
	for {
		etcdAPI.Set(ctx, machineLocation+"/alive", "", &etcd.SetOptions{TTL: 10 * time.Second, Refresh: true})
		time.Sleep(1 * time.Second)
	}
}

func updateLoad() {
	for {
		out, err := exec.Command("uptime").Output()
		if err == nil {
			uptimeString := fmt.Sprintf("%s", out)
			load := strings.Split((strings.Split(uptimeString, "load average: ")[1]), ",")[0]
			etcdAPI.Set(ctx, machineLocation+"/load", load, &etcd.SetOptions{})
		}
		time.Sleep(1 * time.Second)
	}
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

func setTags(tags map[string]string) {

}

func startUnits() {
	result, err := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/machines/%s/%s/units", Config.Zone, Config.MachineName), &etcd.GetOptions{Recursive: true})
	if err == nil {
		for _, node := range result.Node.Nodes {
			u := unit.NewFromEtcd(node.Value)
			u.Start()
		}
	}
	go watchUnits()
}

func watchUnits() {
	w := etcdAPI.Watcher(machineLocation+"/units", &etcd.WatcherOptions{Recursive: true})
	for {
		r, err := w.Next(ctx)
		if err != nil {
			go watchUnits()
			return
		}
		fmt.Println(r)
	}
}
