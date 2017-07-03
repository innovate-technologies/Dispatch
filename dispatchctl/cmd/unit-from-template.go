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
	"strings"

	"github.com/spf13/cobra"
)

type templateUnitParams struct {
	Name string            `json:"name"`
	Vars map[string]string `json:"vars"`
}

type vars []string

func (v *vars) Set(value string) error {
	*v = append(*v, value)
	return nil
}

func (v *vars) Type() string {
	return "[]string"
}

func (v *vars) String() string {
	return strings.Join(*v, " ")
}

// unitFromTemplateCmd represents the unitFromTemplate command
var unitFromTemplateCmd = &cobra.Command{
	Use:   "unit-from-template",
	Short: "Creates a new unti from a template",
	Long: `Creates a new unti from a template
	unit-from-template [template] [unit name]`,
	Run: unitFromTemplate,
}

var unitVars = vars{}

func init() {
	RootCmd.AddCommand(unitFromTemplateCmd)

	unitFromTemplateCmd.Flags().VarP(&unitVars, "toggle", "t", "Help message for toggle")
}

func unitFromTemplate(cmd *cobra.Command, args []string) {
	if len(args) < 2 {
		fmt.Println("Not enough arguments")
		return
	}

	response := StandardResponse{}
	params := templateUnitParams{
		Name: args[1],
	}

	params.Vars = map[string]string{}
	for _, variable := range unitVars {
		if strings.Contains(variable, "=") {
			parts := strings.Split(variable, "=")
			params.Vars[parts[0]] = strings.Join(parts[1:], "=")
		}
	}

	_, postErr := r.R().SetHeader("Content-Type", "application/json").SetBody(params).SetResult(&response).Post("/unit/from-template/" + args[0])
	if postErr != nil {
		fmt.Println(postErr)
		return
	}
	if response.Status == "error" {
		fmt.Println(response.Error)
		return
	}
	fmt.Println("Unit has been created on the cluster")
}
