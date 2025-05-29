/*
tool application
*/

package main

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cli utility",
	Short: "",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
	
}

func main() {
	rootCmd.Execute()
}
