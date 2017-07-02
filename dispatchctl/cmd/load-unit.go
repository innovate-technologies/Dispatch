package cmd

import (
	"fmt"
	"strings"

	"io/ioutil"

	"github.com/spf13/cobra"
)

// UnitParams are all the parameters for a unit
type UnitParams struct {
	Name         string  `json:"name"`
	DesiredState string  `json:"desiredState"`
	Ports        []int64 `json:"ports"`
	Constraints  map[string]string
	UnitContent  string `json:"unitContent"`
}

// loadunitCmd represents the load-unit command
var loadunitCmd = &cobra.Command{
	Use:   "load-unit",
	Short: "Loads unit file",
	Long:  `Loads a unit file onto the dispatch cluster`,
	Run:   runLoadUnit,
}

func init() {
	RootCmd.AddCommand(loadunitCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// load-unitCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// load-unitCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}

func runLoadUnit(cmd *cobra.Command, args []string) {
	if len(args) < 1 {
		fmt.Println("Not enough arguments")
		return
	}

	fileBytes, err := ioutil.ReadFile(args[0])
	if err != nil {
		fmt.Println("Can't open file")
		return
	}
	pathParts := strings.Split(args[0], "/")

	params := UnitParams{}
	params.Name = pathParts[len(pathParts)-1]
	params.UnitContent = string(fileBytes)

	response := StandardResponse{}

	_, postErr := r.R().SetHeader("Content-Type", "application/json").SetBody(params).SetResult(&response).Post("/unit")
	if postErr != nil {
		fmt.Println(postErr)
		return
	}
	if response.Status == "error" {
		fmt.Println(response.Error)
		return
	}
	fmt.Println("Unit has been loaded to the cluster")
}
