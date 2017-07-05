// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// catCmd represents the cat command
var catCmd = &cobra.Command{
	Use:   "cat",
	Short: "Get the content of a unit",
	Long:  `Cat gets the content of a unit on the cluster`,
	Run:   catUnit,
}

func init() {
	RootCmd.AddCommand(catCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// catCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// catCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func catUnit(cmd *cobra.Command, args []string) {

	if len(args) < 1 {
		fmt.Println("Not enough arguments")
		return
	}

	unit := UnitContent{}

	_, err := r.R().SetResult(&unit).Get("/unit/" + args[0])
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(unit.UnitContent)
}
