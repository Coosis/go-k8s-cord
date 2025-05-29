/*
used for starting central control application
*/

package main

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cli for central control",
	Short: "",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
	
}

func main() {
	rootCmd.Execute()
}
