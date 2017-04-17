package main

import (
	"fmt"

	"github.com/innovate-technologies/Dispatch/dispatchdapiserver"
	"github.com/innovate-technologies/Dispatch/dispatchdcommand"
	"github.com/innovate-technologies/Dispatch/dispatchdconfig"
	"github.com/innovate-technologies/Dispatch/dispatchdmachine"
	"github.com/innovate-technologies/Dispatch/dispatchdsupervisor"
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
