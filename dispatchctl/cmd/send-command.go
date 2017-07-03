package cmd

import (
	"fmt"

	"strings"

	"github.com/spf13/cobra"
)

type commandInfo struct {
	Command string `json:"command" form:"command" query:"command"`
}

// send-commandCmd represents the send-command command
var sendCommandCmd = &cobra.Command{
	Use:   "send-command",
	Short: "Sends a command to the cluster",
	Long:  `send-command sends a command to the cluster to be executed on all members in the shell. `,
	Run:   sendCommand,
}

func init() {
	RootCmd.AddCommand(sendCommandCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// send-commandCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// send-commandCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}

func sendCommand(cmd *cobra.Command, args []string) {
	command := commandInfo{}
	response := StandardResponse{}
	command.Command = strings.Join(args, " ")
	_, postErr := r.R().SetHeader("Content-Type", "application/json").SetBody(command).SetResult(&response).Post("/command")
	if postErr != nil {
		fmt.Println(postErr)
	}
	fmt.Println("send-command called")
}
