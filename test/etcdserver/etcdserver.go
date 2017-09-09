package etcdserver

import (
	"log"
	"os"
	"time"

	"github.com/coreos/etcd/embed"
)

var e *embed.Etcd
var failCount int

// Start starts an embedded etcd server for testing
func Start() {
	done := make(chan bool)
	go startGo(done)
	<-done
}

func startGo(done chan bool) {
	cfg := embed.NewConfig()
	cfg.Dir = "default.etcd"
	var err error
	e, err = embed.StartEtcd(cfg)
	if err != nil {
		failCount++
		if failCount < 15 {
			time.Sleep(600 * time.Millisecond)
			Start()
		} else {
			log.Fatal(err)
		}
		return
	}
	failCount = 0
	select {
	case <-e.Server.ReadyNotify():
		log.Printf("Server is ready!")
		done <- true
	case <-time.After(60 * time.Second):
		e.Server.Stop() // trigger a shutdown
		log.Printf("Server took too long to start!")
	}
	log.Fatal(<-e.Err())
}

// Stop stops the started embedded etcd server
func Stop() {
	e.Server.HardStop()
	e.Close()
	os.RemoveAll("default.etcd")
}
