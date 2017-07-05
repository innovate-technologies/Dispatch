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

// catTemplateCmd represents the catTemplate command
var catTemplateCmd = &cobra.Command{
	Use:   "cat-template",
	Short: "Gets the content of a template",
	Long:  `cat-template gets the content of a template`,
	Run:   catTemplate,
}

func init() {
	RootCmd.AddCommand(catTemplateCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// catTemplateCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// catTemplateCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func catTemplate(cmd *cobra.Command, args []string) {

	if len(args) < 1 {
		fmt.Println("Not enough arguments")
		return
	}

	template := TemplateContent{}

	_, err := r.R().SetResult(&template).Get("/template/" + args[0])
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(template.UnitContent)
}
