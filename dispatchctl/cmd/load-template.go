package cmd

import (
	"fmt"
	"io/ioutil"

	"github.com/spf13/cobra"
)

// load-templateCmd represents the load-template command
var loadTemplateCmd = &cobra.Command{
	Use:   "load-template",
	Short: "Loads a template file",
	Long:  `load-template loads a template file to the cluster that can later be used for unit creation`,
	Run:   loadTemplate,
}

// TemplateParams contains all the parameters needed to create a template
type TemplateParams struct {
	Name          string            `json:"name"`
	Ports         []int64           `json:"ports"`
	Constraints   map[string]string `json:"constraints"`
	UnitContent   string            `json:"unitContent"`
	MaxPerMachine int64             `json:"maxPerMachine"`
}

func init() {
	RootCmd.AddCommand(loadTemplateCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// load-templateCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// load-templateCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}

func loadTemplate(cmd *cobra.Command, args []string) {
	if len(args) < 2 {
		fmt.Println("Not enough arguments")
		return
	}

	fileBytes, err := ioutil.ReadFile(args[1])
	if err != nil {
		fmt.Println("Can't open file")
		return
	}

	params := TemplateParams{}
	params.Name = args[0]
	params.UnitContent = string(fileBytes)

	response := StandardResponse{}

	res, postErr := r.R().SetHeader("Content-Type", "application/json").SetBody(params).SetResult(&response).Post("/template")

	fmt.Println(string(res.Body()))

	if postErr != nil {
		fmt.Println(postErr)
		return
	}
	if response.Status == "error" {
		fmt.Println(response.Error)
		return
	}
	fmt.Println("Template has been loaded to the cluster")
}
