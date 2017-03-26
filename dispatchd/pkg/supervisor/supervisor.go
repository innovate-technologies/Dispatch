package supervisor

import (
	"fmt"
	"strings"
	"time"

	"../config"
	"./queue"

	etcd "github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

var (
	ctx     = context.Background()
	etcdAPI etcd.KeysAPI
	// Config is a pointer need to be set to the main configuration
	Config *config.ConfigurationInfo
	// IsSupervisor indicates if this machine is the supervisor
	IsSupervisor = false
)

// Run checks for a supervisor and becomes supervisor when needed
func Run() {
	setUpEtcd()
	checkSupervisorAlive()
	go watchToBecomeSupervisor()
}

func checkSupervisorAlive() {
	_, err := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/supervisor/%s/alive", Config.Zone), &etcd.GetOptions{})
	if err != nil {
		election()
	}
}

func election() {
	voteKey := voteForSupervisor()
	winner := getWinningVote()
	if winner == voteKey {
		fmt.Println("Becoming supervisor")
		becomeSupervisor()
	}
}

func voteForSupervisor() string {
	fmt.Println("Voting for new supervisor")
	res, err := etcdAPI.CreateInOrder(ctx, fmt.Sprintf("/dispatch/vote/%s/", Config.Zone), Config.MachineName, &etcd.CreateInOrderOptions{TTL: 10 * time.Second})
	if err != nil {
		panic(err)
	}
	return res.Node.Key

}

func getWinningVote() string {
	res, err := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/vote/%s/", Config.Zone), &etcd.GetOptions{Recursive: true, Sort: true})
	if err != nil {
		panic(err)
	}
	return res.Node.Nodes[0].Key
}

func becomeSupervisor() {
	IsSupervisor = true
	etcdAPI.Set(ctx, fmt.Sprintf("/dispatch/supervisor/%s/alive", Config.Zone), "1", &etcd.SetOptions{TTL: 10 * time.Second})
	etcdAPI.Set(ctx, fmt.Sprintf("/dispatch/supervisor/%s/machine", Config.Zone), Config.MachineName, &etcd.SetOptions{})
	go letPeasantsKnow()
	go watchMachines()
	queue.Config = Config
	go queue.Run()
}

// letPeasantsKnow makes sure everybody knows you're not dead
func letPeasantsKnow() {
	for {
		etcdAPI.Set(ctx, fmt.Sprintf("/dispatch/supervisor/%s/alive", Config.Zone), "", &etcd.SetOptions{TTL: 10 * time.Second, Refresh: true})
		time.Sleep(1 * time.Second)
	}
}

func watchToBecomeSupervisor() {
	if IsSupervisor {
		return // why would you watch yourself?
	}
	w := etcdAPI.Watcher(fmt.Sprintf("/dispatch/supervisor/%s/alive", Config.Zone), &etcd.WatcherOptions{})
	for {
		r, err := w.Next(ctx)
		if err != nil {
			go watchToBecomeSupervisor()
			return
		}
		if r.Action == "expire" {
			election()
		}
	}
}

func watchMachines() {
	go checkForDeadMachines() // clean out the dead on arrival ones
	w := etcdAPI.Watcher(fmt.Sprintf("/dispatch/machines/%s/", Config.Zone), &etcd.WatcherOptions{Recursive: true})
	for {
		r, err := w.Next(ctx)
		if err != nil {
			fmt.Println("Oops.... this has yet to be designed. Consider me dead please")
			return
		}
		keyComponents := strings.Split(r.Node.Key, "/")
		if r.Action == "expire" && keyComponents[len(keyComponents)-1] == "alive" {
			fmt.Println(keyComponents[len(keyComponents)-2], "died")
			foundDeadMachine(strings.Join(keyComponents[:len(keyComponents)-1], "/"))
		}
	}
}

func checkForDeadMachines() {
	response, err := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/machines/%s/", Config.Zone), &etcd.GetOptions{})
	if err != nil {
		return //not important for now
	}
	for _, node := range response.Node.Nodes {
		_, err := etcdAPI.Get(ctx, fmt.Sprintf("%s/alive", node.Key), &etcd.GetOptions{})
		if err != nil {
			fmt.Println(node.Key, "dead at arrival")
			foundDeadMachine(node.Key)
		}
	}
}

func foundDeadMachine(key string) {
	result, err := etcdAPI.Get(ctx, key+"/units", &etcd.GetOptions{Recursive: true})
	if err == nil {
		for _, node := range result.Node.Nodes {
			go queue.AddUnit(node.Value)
		}
	}
	etcdAPI.Delete(ctx, key, &etcd.DeleteOptions{Recursive: true, Dir: true})
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
