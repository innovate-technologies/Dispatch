package cmd

import (
	"fmt"
	"net"
	"net/http"
	"os"

	resty "gopkg.in/resty.v0"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	zone    string
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "Dispatch",
	Short: "Low level distributed init system",
	Long:  `Dispatch is a distributed server system on top of systemd and etcd`,
}

var r *resty.Client

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	unixSocket := "/var/run/dispatch.socket"
	transport := http.Transport{
		Dial: func(_, _ string) (net.Conn, error) {
			return net.Dial("unix", unixSocket)
		},
	}
	r = resty.New().SetTransport(&transport).SetHostURL("http://dispatch/").SetScheme("http")

	// Here you will define your flags and configuration settings.
	// Cobra supports Persistent Flags, which, if defined here,
	// will be global for your application.

}

// initConfig reads in config file and ENV variables if set.
func initConfig() {

	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName(".Dispatch") // name of config file (without extension)
	viper.AddConfigPath("$HOME")     // adding home directory as first search path
	viper.AutomaticEnv()             // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

// Helping things

// StandardResponse is the server's response for most of the non get requests
type StandardResponse struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}
