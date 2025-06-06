package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	. "github.com/Coosis/go-k8s-cord/internal/agent/model"

	pba "github.com/Coosis/go-k8s-cord/internal/pb/agent/v1"
)

var podsCmd = &cobra.Command{
	Use: "pods",
	Short: "list pods of the agent",
	RunE: func(cmd *cobra.Command, args []string) error {
		AgentServerConfigSetup()
		config := NewAgentConfig()
		if config.Addr == ADDR_PLACEHOLDER {
			return fmt.Errorf(AGENT_CONFIG_NOT_SET)
		}
		addr := fmt.Sprintf("http://%s%s/pods/list", config.Addr, config.HTTPSPort)
		rasp, err := http.Get(addr)
		if err != nil {
			return err
		}
		defer rasp.Body.Close()
		if rasp.StatusCode != http.StatusOK {	
			return fmt.Errorf("failed to get status, status code: %d", rasp.StatusCode)
		}

		var pods []*pba.PodMetadata
		if err := json.NewDecoder(rasp.Body).Decode(&pods); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}

		if len(pods) == 0 {
			return nil
		}

		for _, pod := range pods {
			fmt.Printf(
				"Pod Name: %s, Namespace: %s, Status: %s\n",
				*pod.Name,
				*pod.Namespace,
				*pod.Uid,
			)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(podsCmd)
}
