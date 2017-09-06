package machine

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/innovate-technologies/Dispatch/dispatchd/config"
	"github.com/innovate-technologies/Dispatch/dispatchd/etcdclient"
	"github.com/innovate-technologies/Dispatch/dispatchd/unit"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"golang.org/x/net/context"
)

var (
	ctx             = context.Background()
	etcdAPI         = etcdclient.GetEtcdv3()
	machineLocation string
	// Config is a pointer need to be set to the main configuration
	Config     *config.ConfigurationInfo
	units      map[string]unit.Unit
	aliveLease *etcd.LeaseGrantResponse
)

// RegisterMachine adds the machine to the cluster
func RegisterMachine() {
	unit.KillAllOldUnits() // Starting clean

	unit.Config = Config           // pass through the config
	units = map[string]unit.Unit{} // initialize map

	machineLocation = fmt.Sprintf("/dispatch/%s/machines/%s", Config.Zone, Config.MachineName)

	etcdAPI.Put(ctx, machineLocation+"/arch", Config.Arch)
	etcdAPI.Put(ctx, machineLocation+"/ip", Config.PublicIP)

	var err error
	aliveLease, err = etcdAPI.Lease.Grant(ctx, 10)
	if err != nil {
		panic(err)
	}

	etcdAPI.Put(ctx, machineLocation+"/alive", "1", etcd.WithLease(aliveLease.ID))

	go renewAlive()
	go updateLoad()
	go startUnits()
	go checkUnits()
}

func renewAlive() {
	for {
		etcdAPI.Lease.KeepAliveOnce(ctx, aliveLease.ID)
		time.Sleep(1 * time.Second)
	}
}

func updateLoad() {
	for {
		out, err := exec.Command("uptime").Output()
		if err == nil {
			uptimeString := fmt.Sprintf("%s", out)
			var textAfterLoadAverage string
			if strings.Index(uptimeString, "load averages") >= 0 {
				textAfterLoadAverage = strings.Split(uptimeString, "load averages: ")[1]
			} else {
				textAfterLoadAverage = strings.Split(uptimeString, "load average: ")[1]
			}
			load := strings.Split(textAfterLoadAverage, ",")[0] //to do: divide #CPU
			etcdAPI.Put(ctx, machineLocation+"/load", load)
		}
		time.Sleep(1 * time.Second)
	}
}

func setTags(tags map[string]string) {

}

func startUnits() {
	result, err := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/%s/machines/%s/units", Config.Zone, Config.MachineName))
	if err == nil {
		for _, kv := range result.Kvs {
			unitName := string(kv.Value)
			u := unit.NewFromEtcd(unitName)
			go u.LoadAndWatch()
			units[unitName] = u
		}
	}
	go watchUnits()
}

func watchUnits() {
	chans := etcdAPI.Watch(context.Background(), fmt.Sprintf("/dispatch/%s/queue", Config.Zone), etcd.WithPrefix())
	for resp := range chans {
		for _, ev := range resp.Events {
			if ev.IsCreate() {
				unitName := string(ev.Kv.Value)
				u := unit.NewFromEtcd(unitName)
				go u.LoadAndWatch()
				units[unitName] = u
			}
			if ev.Type == mvccpb.DELETE {
				unitName := string(ev.PrevKv.Value)
				if unit, exists := units[unitName]; exists {
					unit.Destroy()
					delete(units, unitName)
				}
			}
		}
	}
}

func checkUnits() {
	for {
		time.Sleep(10 * time.Second)
		result, err := etcdAPI.Get(ctx, machineLocation+"/units", etcd.WithPrefix())
		if err == nil {
			for _, kv := range result.Kvs {
				unitName := string(kv.Value)
				if _, ok := units[unitName]; !ok {
					u := unit.NewFromEtcd(unitName)
					go u.LoadAndWatch()
					units[unitName] = u
				}
			}
		}
	}
}
