package etcdserver

import (
	"log"
	"os"
	"time"

	"github.com/coreos/etcd/embed"
)

var e *embed.Etcd

// Start starts an embedded etcd server for testing
func Start() {
	cfg := embed.NewConfig()
	cfg.Dir = "default.etcd"
	var err error
	e, err = embed.StartEtcd(cfg)
	if err != nil {
		log.Fatal(err)
	}
	select {
	case <-e.Server.ReadyNotify():
		log.Printf("Server is ready!")
	case <-time.After(60 * time.Second):
		e.Server.Stop() // trigger a shutdown
		log.Printf("Server took too long to start!")
	}
	//log.Fatal(<-e.Err())
}

// Stop stops the started embedded etcd server
func Stop() {
	e.Server.Stop()
	e.Close()
	os.RemoveAll("default.etcd")
}
