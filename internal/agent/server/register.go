// Register
package server

import (
	"context"
	"fmt"

	. "github.com/Coosis/go-k8s-cord/internal/agent/model"

	pbc "github.com/Coosis/go-k8s-cord/internal/pb/central/v1"
)

func (s *AgentServer) Register(ctx context.Context) (string, error) {
	cfg := GetAgentConfig()
	centralClient := pbc.NewCentralServiceClient(s.centralConn)
	resp, err := centralClient.RegisterAgent(ctx, &pbc.RegisterAgentRequest{
		AgentGrpcEndpoint: &cfg.Addr,
	})
	if err != nil {
		return "", fmt.Errorf("failed to register agent: %w", err)
	}
	if !resp.GetSuccess() {
		return "", fmt.Errorf("register failed, remote: %s", resp.GetMessage())
	}
	return resp.GetId(), nil
}
