package supervisor

import (
	"fmt"
	"strings"
	"time"

	"github.com/coreos/etcd/mvcc/mvccpb"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/innovate-technologies/Dispatch/dispatchd/config"
	"github.com/innovate-technologies/Dispatch/dispatchd/etcdclient"
	"github.com/innovate-technologies/Dispatch/dispatchd/supervisor/queue"

	"golang.org/x/net/context"
)

var (
	ctx     = context.Background()
	etcdAPI = etcdclient.GetEtcdv3()
	// Config is a pointer need to be set to the main configuration
	Config *config.ConfigurationInfo
	// IsSupervisor indicates if this machine is the supervisor
	IsSupervisor = false
	aliveLease   *etcd.LeaseGrantResponse
)

// Run checks for a supervisor and becomes supervisor when needed
func Run() {
	if !isSupervisorAlive() {
		election()
	}
	go watchToBecomeSupervisor()
}

func isSupervisorAlive() bool {
	key, err := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/%s/supervisor/alive", Config.Zone))
	if err != nil || len(key.Kvs) == 0 {
		return false
	}
	return true
}

func election() {
	voteForSupervisor()
	winner := getWinningVote()
	if winner == Config.MachineName {
		fmt.Println("Becoming supervisor")
		becomeSupervisor()
	}
}

func voteForSupervisor() {
	fmt.Println("Voting for new supervisor")
	lease, err := etcdAPI.Lease.Grant(ctx, 10)
	if err != nil {
		panic(err)
	}

	etcdAPI.Txn(ctx).
		If(etcd.Compare(etcd.CreateRevision(fmt.Sprintf("/dispatch/%s/vote", Config.Zone)), "=", 0)).
		Then(etcd.OpPut(fmt.Sprintf("/dispatch/%s/vote", Config.Zone), Config.MachineName, etcd.WithLease(lease.ID))).
		Commit()
}

func getWinningVote() string {
	key, err := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/%s/vote", Config.Zone))
	if err != nil {
		panic(err)
	}
	return string(key.Kvs[0].Value)
}

func becomeSupervisor() {
	IsSupervisor = true
	aliveLease, _ = etcdAPI.Lease.Grant(ctx, 10)
	etcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/supervisor/alive", Config.Zone), "1", etcd.WithLease(aliveLease.ID))
	etcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/supervisor/machine", Config.Zone), Config.MachineName, etcd.WithLease(aliveLease.ID))
	go letPeasantsKnow()
	go watchMachines()
	go watchGlobals()
	queue.Config = Config
	go queue.Run()
}

// letPeasantsKnow makes sure everybody knows you're not dead
func letPeasantsKnow() {
	for {
		etcdAPI.Lease.KeepAliveOnce(ctx, aliveLease.ID)
		time.Sleep(1 * time.Second)
	}
}

func watchToBecomeSupervisor() {
	if IsSupervisor {
		return // why would you watch yourself?
	}
	chans := etcdAPI.Watch(context.Background(), fmt.Sprintf("/dispatch/%s/supervisor/alive", Config.Zone))
L:
	for resp := range chans {
		for _, ev := range resp.Events {
			if ev.Type == mvccpb.DELETE {
				election()
				break L
			}
		}
	}

	fmt.Println("DEBUG end alive watch")

}

func watchMachines() {
	go checkForDeadMachines() // clean out the dead on arrival ones
	chans := etcdAPI.Watch(context.Background(), fmt.Sprintf("/dispatch/%s/machines/", Config.Zone), etcd.WithPrefix())
	for resp := range chans {
		for _, ev := range resp.Events {
			if keyComponents := strings.Split(string(ev.Kv.Key), "/"); keyComponents[len(keyComponents)-1] == "alive" {
				machineKey := strings.Join(keyComponents[:5], "/")

				// if died
				if ev.Type == mvccpb.DELETE {
					fmt.Println(keyComponents[len(keyComponents)-2], "died")
					foundDeadMachine(machineKey)
				}

				// if new
				if ev.IsCreate() {

					fmt.Println(machineKey, "is alive")
					foundNewMachine(machineKey)
				}
			}
		}
	}
}

func checkForDeadMachines() {
	response, err := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/%s/machines/", Config.Zone), etcd.WithPrefix())
	if err != nil {
		return //not important for now
	}

	serversHad := map[string]bool{}
	for _, kv := range response.Kvs {
		keyParts := strings.Split(string(kv.Key), "/")
		machine := keyParts[4]

		if _, ok := serversHad[machine]; !ok {
			key := strings.Join(keyParts[:5], "/")
			res, err := etcdAPI.Get(ctx, fmt.Sprintf("%s/alive", key))
			if err != nil || res.Count == 0 {
				fmt.Println(key, "dead at arrival")
				foundDeadMachine(key)
			}
			serversHad[machine] = true
		}
	}
}

func foundDeadMachine(key string) {
	result, err := etcdAPI.Get(ctx, key+"/units", etcd.WithPrefix())
	if err == nil {
		for _, kv := range result.Kvs {
			queue.AddUnit(string(kv.Value))
		}
	}
	etcdAPI.Delete(ctx, key, etcd.WithPrefix())
}

func foundNewMachine(key string) {
	// set globals
	result, err := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/%s/globals/", Config.Zone), etcd.WithPrefix())
	if err == nil {
		for _, kv := range result.Kvs {
			etcdAPI.Put(ctx, key+"/units/"+string(kv.Value), string(kv.Value))
		}
	}
}
