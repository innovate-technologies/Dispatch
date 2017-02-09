package machine

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"../config"

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

	machineLocation = fmt.Sprintf("/dispatch/machine/%s", Config.MachineName)

	etcdAPI.Set(ctx, machineLocation+"/arch", Config.Arch, &etcd.SetOptions{})
	etcdAPI.Set(ctx, machineLocation+"/ip", Config.PublicIP, &etcd.SetOptions{})

	etcdAPI.Set(ctx, machineLocation+"/alive", "1", &etcd.SetOptions{TTL: 10 * time.Second})

	go renewAlive()
	go updateLoad()
	time.Sleep(1000 * time.Second) // to be removed
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
