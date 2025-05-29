package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	// . "github.com/Coosis/go-k8s-cord/internal"
)

var statusCmd = &cobra.Command{
	Use: "status",
	Short: "list status of the system",
	RunE: func(cmd *cobra.Command, args []string) error {
		rasp, err := http.Get("http://localhost:10201/status")
		if err != nil {
			log.Error("Failed to get status:", err)
			return err
		}
		defer rasp.Body.Close()
		if rasp.StatusCode != http.StatusOK {	
			log.Error("Failed to get status, status code:", rasp.StatusCode)
			return fmt.Errorf("failed to get status, status code: %d", rasp.StatusCode)
		}

		txt, err := io.ReadAll(rasp.Body)
		if err != nil {
			log.Error("Failed to read response body:", err)
			return err
		}

		if len(txt) == 0 {
			log.Info("No status information available")
			return nil
		}
		log.Debug("Status response: ", string(txt))

		return nil
	},
}

func init() {
	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stdout)
	rootCmd.AddCommand(statusCmd)
}
