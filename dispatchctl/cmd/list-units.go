package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/innovate-technologies/Dispatch/dispatchd/unit/state"
	"github.com/spf13/cobra"
)

// UnitContent are all the properties for a unit
type UnitContent struct {
	Name         string  `json:"name"`
	DesiredState int     `json:"desiredState"`
	Ports        []int64 `json:"ports"`
	Constraints  map[string]string
	UnitContent  string `json:"unitContent"`
	State        int    `json:"state"`
	Machine      string `json:"machine"`
}

// list-unitsCmd represents the list-units command
var listUnitsCmd = &cobra.Command{
	Use:   "list-units",
	Short: "Gets all units",
	Long:  `Gets a list of all units running in a zone`,
	Run:   runListUnits,
}

func init() {
	RootCmd.AddCommand(listUnitsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// list-unitsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// list-unitsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}

func runListUnits(cmd *cobra.Command, args []string) {
	units := []UnitContent{}

	_, err := r.R().SetResult(&units).Get("/units")
	if err != nil {
		fmt.Println(err)
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 3, 0, 1, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATE\tDESIRED STATE\t MACHINE")
	for _, unit := range units {
		if unit.Machine == "" {
			unit.Machine = "<none>"
		}
		fmt.Fprintln(w, unit.Name+"\t"+state.ForInt(unit.State).String()+"\t"+state.ForInt(unit.DesiredState).String()+"\t"+unit.Machine)
	}

	w.Flush()
}
