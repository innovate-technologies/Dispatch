package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// destroyCmd represents the destroy command
var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Deletes a unit from the cluster",
	Long:  `Destroy sends a command to the cluster to delete the file and stop the running instance`,
	Run:   destroyUnit,
}

func init() {
	RootCmd.AddCommand(destroyCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// destroyCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// destroyCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}

func destroyUnit(cmd *cobra.Command, args []string) {
	if len(args) < 1 {
		fmt.Println("Not enough arguments")
		return
	}
	response := StandardResponse{}

	_, postErr := r.R().SetHeader("Content-Type", "application/json").SetResult(&response).Delete("/unit/" + args[0])
	if postErr != nil {
		fmt.Println(postErr)
		return
	}
	if response.Status == "error" {
		fmt.Println(response.Error)
		return
	}
	fmt.Println("Unit has been destroyed")
}
