package main

import (
	"fmt"

	"./pkg/config"
	"./pkg/machine"
)

var configuration config.ConfigurationInfo

func main() {
	fmt.Println("Dispatch")
	fmt.Println("Copyright 2017 Innovate Technologies")
	fmt.Println("====================================")
	configuration = config.GetConfiguration()
	fmt.Println(configuration.MachineName)

	machine.Config = &configuration
	machine.RegisterMachine()
}
