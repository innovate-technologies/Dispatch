package config

import (
	"encoding/json"
	"os"
)

// ConfigurationInfo contains the config file's content
type ConfigurationInfo struct {
	MachineName string            `json:"machineName"`
	Arch        string            `json:"arch"`
	Tags        map[string]string `json:"tags"`
	BindIP      string            `json:"bindIP"`
	BindPort    int               `json:"bindPort"`
	EtcdAddress string            `json:"etcdAddress"`
	PublicIP    string            `json:"publicIP"`
}

func newConfigurationInfo() ConfigurationInfo {
	config := ConfigurationInfo{}
	config.MachineName, _ = os.Hostname()
	config.BindIP = "127.0.0.1"
	config.BindPort = 7384 // "IT" in ASCII to decimal
	config.EtcdAddress = "http://127.0.0.1:2379"
	return config
}

// GetConfiguration reads the confiruration from config.json and returns it
func GetConfiguration() ConfigurationInfo {
	returnConfig := newConfigurationInfo()
	data, err := os.Open("config.json")
	if err != nil {
		panic(err)
	}
	jsonParser := json.NewDecoder(data)
	err = jsonParser.Decode(&returnConfig)
	if err != nil {
		panic(err)
	}
	return returnConfig
}
