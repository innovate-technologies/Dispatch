package cmd

import (
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// TemplateContent are all the properties for a unit
type TemplateContent struct {
	Name          string            `json:"name"`
	Ports         []int64           `json:"ports"`
	Constraints   map[string]string `json:"constraints"`
	UnitContent   string            `json:"unitContent"`
	MaxPerMachine int64             `json:"maxPerMachine"`
}

var listTemplatesCmd = &cobra.Command{
	Use:   "list-templates",
	Short: "Gets all templates",
	Long:  `Gets a list of all templates running in a zone`,
	Run:   runListTemplates,
}

func init() {
	RootCmd.AddCommand(listTemplatesCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// list-unitsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// list-unitsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}

func runListTemplates(cmd *cobra.Command, args []string) {
	templates := []TemplateContent{}

	_, err := r.R().SetResult(&templates).Get("/templates")
	if err != nil {
		fmt.Println(err)
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 3, 0, 1, ' ', 0)
	fmt.Fprintln(w, "NAME\tPORTS\tMAX PER MACHINE")
	for _, template := range templates {
		ports := ""
		for _, port := range template.Ports {
			ports += strconv.FormatInt(port, 10) + " "
		}

		fmt.Fprintln(w, template.Name+"\t"+ports+"\t"+strconv.FormatInt(template.MaxPerMachine, 10))
	}

	w.Flush()
}
