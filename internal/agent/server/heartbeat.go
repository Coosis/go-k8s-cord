package server

import (
	"context"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	. "github.com/Coosis/go-k8s-cord/internal/agent/model"

	pbc "github.com/Coosis/go-k8s-cord/internal/pb/central/v1"
)

func (s *AgentServer) Heartbeat(ctx context.Context) error {
	cfg := GetAgentConfig()
	if cfg.UUID == "" {
		return fmt.Errorf("agent UUID is not set, please register the agent first")
	}
	grpcEndpoint := cfg.GRPCEndpoint()
	if grpcEndpoint == "" {
		return fmt.Errorf("GRPC endpoint is not set in the config file, please edit the config file and set the endpoint")
	}
	centralClient := pbc.NewCentralServiceClient(s.centralConn)
	resp, err := centralClient.SendHeartbeat(ctx, &pbc.HeartbeatRequest{
		AgentId: proto.String(cfg.UUID),
		AgentName: proto.String(cfg.AgentName),
		Timestamp: proto.Int64(time.Now().Unix()),
		AgentGrpcEndpoint: proto.String(grpcEndpoint),
	})
	if err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}
	if !resp.GetSuccess() {
		return fmt.Errorf("heartbeat failed, remote: %s", resp.GetMessage())
	}
	return nil
}
