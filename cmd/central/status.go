package main

import (
	"bufio"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	. "github.com/Coosis/go-k8s-cord/internal/central/model"
)

var statusCmd = &cobra.Command{
	Use: "status",
	Short: "list statuses of the agent",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := GetCentralConfig()
		addr := fmt.Sprintf("http://localhost%s/api/v1/status", cfg.HTTPSPort)
		rasp, err := http.Get(addr)
		if err != nil {
			return err
		}
		defer rasp.Body.Close()
		if rasp.StatusCode != http.StatusOK {	
			return fmt.Errorf("failed to get status, status code: %d", rasp.StatusCode)
		}

		scanner := bufio.NewScanner(rasp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println(line)
		}
		// txt, err := io.ReadAll(rasp.Body)
		// if err != nil {
		// 	log.Error("Failed to read response body: ", err)
		// 	return err
		// }
		//
		// if len(txt) == 0 {
		// 	log.Info("No status information available")
		// 	return nil
		// }


		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
