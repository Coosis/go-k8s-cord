package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"

	. "github.com/Coosis/go-k8s-cord/internal/agent/model"
)

var statusCmd = &cobra.Command{
	Use: "status",
	Short: "list status of the agent",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := AgentServerConfigSetup()
		if err != nil {
			return fmt.Errorf("failed to setup agent server config: %w", err)
		}

		config := NewAgentConfig()
		if config.Addr == ADDR_PLACEHOLDER {
			return fmt.Errorf(AGENT_CONFIG_NOT_SET)
		}
		addr := fmt.Sprintf("http://%s%s/livez", config.Addr, config.HTTPSPort)
		rasp, err := http.Get(addr)
		if err != nil {
			return err
		}
		defer rasp.Body.Close()
		if rasp.StatusCode != http.StatusOK {	
			return fmt.Errorf("failed to get status, status code: %d", rasp.StatusCode)
		}

		txt, err := io.ReadAll(rasp.Body)
		if err != nil {
			return err
		}

		if len(txt) == 0 {
			return nil
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
