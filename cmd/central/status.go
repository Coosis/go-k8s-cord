package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"

	. "github.com/Coosis/go-k8s-cord/internal/central/model"
	. "github.com/Coosis/go-k8s-cord/internal/central/server"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use: "status",
	Short: "list statuses of the agent",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := CentralServerConfigSetup()
		if err != nil {
			return fmt.Errorf("failed to setup central server config: %w", err)
		}
		config := NewCentralConfig()
		addr := fmt.Sprintf("http://localhost%s/stat", config.HTTPSPort)
		rasp, err := http.Get(addr)
		if err != nil {
			log.Error("Failed to get status: ", err)
			return err
		}
		defer rasp.Body.Close()
		if rasp.StatusCode != http.StatusOK {	
			log.Error("Failed to get status, status code: ", rasp.StatusCode)
			return fmt.Errorf("failed to get status, status code: %d", rasp.StatusCode)
		}

		log.Info("Status response: ")
		scanner := bufio.NewScanner(rasp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			log.Info(line)
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
	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stdout)
	rootCmd.AddCommand(statusCmd)
}
