package etcdclient

import (
	"log"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/pkg/transport"
	"github.com/innovate-technologies/Dispatch/dispatchd/config"
)

var instance *etcd.Client

// GetEtcdv3 gives the etcdv3 client
func GetEtcdv3() *etcd.Client {
	if instance == nil {
		var config = config.GetConfiguration()
		var etcdConfig = etcd.Config{
			Endpoints: []string{config.EtcdAddress},
		}

		if config.EtcdAuth.Username != "" {
			etcdConfig.Username = config.EtcdAuth.Username
			etcdConfig.Password = config.EtcdAuth.Password
		}

		if config.EtcdTLS.CACert != "" {
			tlsInfo := transport.TLSInfo{
				TrustedCAFile: config.EtcdTLS.CACert,
			}
			tlsConfig, err := tlsInfo.ClientConfig()
			if err != nil {
				log.Fatal(err)
			}
			etcdConfig.TLS = tlsConfig
		}

		c, err := etcd.New(etcdConfig)
		if err != nil {
			panic(err)
		}

		instance = c
	}
	return instance
}
