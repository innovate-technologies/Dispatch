package command

import (
	"bufio"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/innovate-technologies/Dispatch/dispatchdconfig"

	etcd "github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

var (
	ctx     = context.Background()
	etcdAPI etcd.KeysAPI
	// Config is a pointer need to be set to the main configuration
	Config    *config.ConfigurationInfo
	etcdMutex sync.Mutex
)

// Run starts watching for commands to execute
func Run() {
	setUpEtcd()
	go watchForNewCommands()
}

func watchForNewCommands() {
	w := etcdAPI.Watcher(fmt.Sprintf("/dispatch/commands/%s/", Config.Zone), &etcd.WatcherOptions{Recursive: true})
	for {
		r, err := w.Next(ctx)
		if err != nil {
			go watchForNewCommands()
			return
		}
		if pathParts := strings.Split(r.Node.Key, "/"); r.Action == "set" && pathParts[len(pathParts)-1] == "command" {
			go executeAndReturnResults(strings.Join(pathParts[:len(pathParts)-1], "/"), r.Node.Value)
		}
	}
}

func executeAndReturnResults(key, command string) {
	fmt.Println(key) // TO DO execute and push info to etcd
	fmt.Println(command)

	etcdAPI.Set(ctx, fmt.Sprintf("%s/machines/%s/output", key, Config.MachineName), "", &etcd.SetOptions{})

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
	etcdAPI.Set(ctx, fmt.Sprintf("%s/machines/%s/result", key, Config.MachineName), err.Error(), &etcd.SetOptions{})
}

func readSdtToEtcd(key string, std *bufio.Reader) {
	outputPath := fmt.Sprintf("%s/machines/%s/output", key, Config.MachineName)
	for {
		line, _, err := std.ReadLine()
		if err != nil {
			return // end of stream
		}
		etcdMutex.Lock()
		response, _ := etcdAPI.Get(ctx, outputPath, &etcd.GetOptions{})
		output := response.Node.Value + string(line[:]) + "\n"
		etcdAPI.Set(ctx, outputPath, output, &etcd.SetOptions{})
		etcdMutex.Unlock()
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
