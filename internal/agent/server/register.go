package server

import (
	"context"
	"fmt"

	pbc "github.com/Coosis/go-k8s-cord/internal/pb/central/v1"
)

func (s *AgentServer) Register(ctx context.Context) error {
	centralClient := pbc.NewCentralServiceClient(s.centralConn)
	resp, err := centralClient.RegisterAgent(ctx, &pbc.RegisterAgentRequest{
		AgentGrpcEndpoint: &s.Config.Addr,
	})
	if err != nil {
		return fmt.Errorf("failed to register agent: %w", err)
	}
	if !resp.GetSuccess() {
		return fmt.Errorf("register failed, remote: %s", resp.GetMessage())
	}
	s.Config.Registered = true
	s.Config.UUID = resp.GetId()
	if err := s.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config after registration: %w", err)
	}
	return nil
}
