/*
used for starting agent
*/

package main

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "agent",
	Short: "",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
	
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		panic(err)
	}
}
