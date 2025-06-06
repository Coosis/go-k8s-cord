// agent register grpc server-side implementation
package server

import (
	"context"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/google/uuid"

	. "github.com/Coosis/go-k8s-cord/internal/central/model"

	log "github.com/sirupsen/logrus"
	pbc "github.com/Coosis/go-k8s-cord/internal/pb/central/v1"
)

func(s *CentralServer) RegisterAgent(
	ctx context.Context,
	req *pbc.RegisterAgentRequest,
) (*pbc.RegisterAgentResponse, error) {
	id, err := uuid.NewV7()
	if err != nil {
		log.Error("Failed to generate UUID:", err)
		return nil, fmt.Errorf("failed to generate UUID: %w", err)
	}
	idstr := id.String()
	log.Debugf(
		"Registering agent with ID: %s, gRPC Endpoint: %s",
		id,
		req.GetAgentGrpcEndpoint(),
	)
	s.agents[idstr] = &AgentMetadata{
		Name: "TO_BE_SET", // Placeholder, should be set from heartbeat request
		GrpcEndpoint: req.GetAgentGrpcEndpoint(),
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return &pbc.RegisterAgentResponse{
		Success: proto.Bool(true),
		Id: &idstr,
		Timestamp: proto.Int64(time.Now().Unix()),
	}, nil
}
