package main

import (
	"fmt"

	"./pkg/apiserver"
	"./pkg/config"
	"./pkg/machine"
	"./pkg/supervisor"
)

var configuration config.ConfigurationInfo

func main() {
	fmt.Println("Dispatch")
	fmt.Println("Copyright 2017 Innovate Technologies")
	fmt.Println("====================================")
	configuration = config.GetConfiguration()
	fmt.Println(configuration.MachineName)

	machine.Config = &configuration
	supervisor.Config = &configuration

	machine.RegisterMachine()
	supervisor.Run()
	apiserver.Run()
}
