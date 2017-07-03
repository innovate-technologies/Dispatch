package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// delete-templateCmd represents the delete-template command
var deleteTemplateCmd = &cobra.Command{
	Use:   "delete-template",
	Short: "Delete a template",
	Long:  `delete-template deletes a template from the cluster`,
	Run:   deleteTemplate,
}

func init() {
	RootCmd.AddCommand(deleteTemplateCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// delete-templateCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// delete-templateCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}

func deleteTemplate(cmd *cobra.Command, args []string) {

	if len(args) < 1 {
		fmt.Println("Not enough arguments")
		return
	}
	response := StandardResponse{}

	_, postErr := r.R().SetHeader("Content-Type", "application/json").SetResult(&response).Delete("/template/" + args[0])
	if postErr != nil {
		fmt.Println(postErr)
		return
	}
	if response.Status == "error" {
		fmt.Println(response.Error)
		return
	}
	fmt.Println("Template has been deleted")
}
