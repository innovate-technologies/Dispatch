package main

import (
	"fmt"

	"dispatchd/apiserver"
	"dispatchd/command"
	"dispatchd/config"
	"dispatchd/machine"
	"dispatchd/supervisor"
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
	command.Config = &configuration

	machine.RegisterMachine()
	supervisor.Run()
	command.Run()

	apiserver.Run()

}
