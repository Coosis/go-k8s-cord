package server

import (
	"context"
	"fmt"
	"time"

	pbc "github.com/Coosis/go-k8s-cord/internal/pb/central/v1"
	"github.com/gogo/protobuf/proto"
)

func (s *AgentServer) Heartbeat(ctx context.Context) error {
	if s.Config.UUID == "" {
		return fmt.Errorf("agent UUID is not set, please register the agent first")
	}
	grpcEndpoint := s.Config.GRPCEndpoint()
	if grpcEndpoint == "" {
		return fmt.Errorf("GRPC endpoint is not set in the config file, please edit the config file and set the endpoint")
	}
	centralClient := pbc.NewCentralServiceClient(s.centralConn)
	resp, err := centralClient.SendHeartbeat(ctx, &pbc.HeartbeatRequest{
		AgentId: proto.String(s.Config.UUID),
		AgentName: proto.String(s.Config.AgentName),
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
