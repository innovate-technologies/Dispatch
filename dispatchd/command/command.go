package command

import (
	"bufio"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/innovate-technologies/Dispatch/dispatchd/config"
	"github.com/innovate-technologies/Dispatch/dispatchd/etcdclient"

	etcd "github.com/coreos/etcd/clientv3"
	"golang.org/x/net/context"
)

var (
	ctx     = context.Background()
	etcdAPI = etcdclient.GetEtcdv3()
	// Config is a pointer need to be set to the main configuration
	Config    *config.ConfigurationInfo
	etcdMutex = &sync.Mutex{}
)

// Run starts watching for commands to execute
func Run() {
	go watchForNewCommands()
}

// SendCommand places a command to be ran on etcd
func SendCommand(command string) string {
	lease, _ := etcdAPI.Lease.Grant(ctx, 60*60*60*24) //24h
	etcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/commands/%d/command", Config.Zone, lease.ID), command, etcd.WithLease(lease.ID))
	return fmt.Sprintf("%d", lease.ID)
}

func watchForNewCommands() {
	chans := etcdAPI.Watch(ctx, fmt.Sprintf("/dispatch/%s/commands", Config.Zone), etcd.WithPrefix())
	for resp := range chans {
		for _, ev := range resp.Events {
			if pathParts := strings.Split(string(ev.Kv.Key), "/"); ev.IsCreate() && pathParts[len(pathParts)-1] == "command" {
				go executeAndReturnResults(strings.Join(pathParts[:len(pathParts)-1], "/"), string(ev.Kv.Value))
			}
		}
	}
}

func executeAndReturnResults(key, command string) {
	fmt.Println(key) // TO DO execute and push info to etcd
	fmt.Println(command)

	etcdAPI.Put(ctx, fmt.Sprintf("%s/machines/%s/output", key, Config.MachineName), "")

	commandParts := strings.Split(command, " ")
	commandoProcess := exec.Command(commandParts[0], commandParts[1:]...)
	stdoutPipe, _ := commandoProcess.StdoutPipe()
	stderrPipe, _ := commandoProcess.StderrPipe()
	stdoutReader := bufio.NewReader(stdoutPipe)
	stderrReader := bufio.NewReader(stderrPipe)

	go readSdtToEtcd(key, stdoutReader)
	go readSdtToEtcd(key, stderrReader)

	err := commandoProcess.Run() // blocking this thread
	if err == nil {
		err = errors.New("ok")
	}
	fmt.Println("done")
	etcdAPI.Put(ctx, fmt.Sprintf("%s/machines/%s/result", key, Config.MachineName), err.Error())
}

func readSdtToEtcd(key string, std *bufio.Reader) {
	outputPath := fmt.Sprintf("%s/machines/%s/output", key, Config.MachineName)
	for {
		line, _, err := std.ReadLine()
		if err != nil {
			return // end of stream
		}
		etcdMutex.Lock()

		var output string
		if response, _ := etcdAPI.Get(ctx, outputPath); response.Count == 0 {
			output = string(line[:]) + "\n"
		} else {
			output = string(response.Kvs[0].Value) + string(line[:]) + "\n"
		}

		lease, _ := etcdAPI.Lease.Grant(ctx, 60*60*60*24) //24h
		etcdAPI.Put(ctx, outputPath, output, etcd.WithLease(lease.ID))
		etcdMutex.Unlock()
	}
}
