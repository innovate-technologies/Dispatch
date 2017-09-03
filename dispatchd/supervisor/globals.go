package supervisor

import (
	"fmt"
	"strings"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"golang.org/x/net/context"
)

func watchGlobals() {
	chans := etcdAPI.Watch(context.Background(), fmt.Sprintf("/dispatch/%s/globals", Config.Zone), etcd.WithPrefix())
	for resp := range chans {
		for _, ev := range resp.Events {
			if ev.IsCreate() {
				assignToAllMachines(string(ev.Kv.Value))
			}
			if ev.Type == mvccpb.DELETE {
				removeFromAllMachines(string(ev.Kv.Value))
			}
		}
	}

}

func assignToAllMachines(unit string) {
	keys, _ := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/%s/machines/", Config.Zone), etcd.WithPrefix())
	machinesHad := map[string]bool{}
	for _, kv := range keys.Kvs {
		machine := strings.Split(string(kv.Key), "/")[4]
		if _, ok := machinesHad[machine]; !ok {
			etcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/machines/%s/units/%s", Config.Zone, machine, unit), unit)
			machinesHad[machine] = true
		}
	}
}

func removeFromAllMachines(unit string) {
	keys, _ := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/%s/machines/", Config.Zone), etcd.WithPrefix())
	machinesHad := map[string]bool{}
	for _, kv := range keys.Kvs {
		machine := strings.Split(string(kv.Key), "/")[4]
		if _, ok := machinesHad[machine]; !ok {
			etcdAPI.Delete(ctx, fmt.Sprintf("/dispatch/%s/machines/%s/units/%s", Config.Zone, machine, unit))
			machinesHad[machine] = true
		}
	}
}
