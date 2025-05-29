package server

import (
	"context"
	"fmt"
	"time"

	"github.com/Coosis/go-k8s-cord/internal/central/model"
	pbc "github.com/Coosis/go-k8s-cord/internal/pb/central/v1"
	"github.com/gogo/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Server's implementation for receiving heartbeats from agents.
func(s *CentralServer) SendHeartbeat(
	ctx context.Context,
	req *pbc.HeartbeatRequest,
) (*pbc.HeartbeatResponse, error) {
	id := req.GetAgentId()
	idstr := fmt.Sprintf("%s", id)
	log.Debug("Heartbeat received from agent: ", id)
	// Register the agent in etcd
	now := time.Now().Unix()
	nowstr := fmt.Sprintf("%d", now)
	log.Debugf("Registering heartbeat for agent %s at %s", idstr, nowstr)
	_, err := s.etcd.Put(ctx, idstr, nowstr)
	if err != nil {
		return nil, fmt.Errorf("failed to register heartbeat in etcd: %w", err)
	}

	if _, exists := s.agents[idstr]; !exists {
		// If the agent is not registered, create a new entry
		log.Debugf("Agent %s not found, creating new entry", idstr)
		cred := credentials.NewTLS(s.tlsConfig)
		// Create a gRPC connection to the agent
		conn, err := grpc.NewClient(req.GetAgentGrpcEndpoint(), grpc.WithTransportCredentials(cred))
		if err != nil {
			return nil, fmt.Errorf("failed to create gRPC connection to agent: %w", err)
		}
		s.agents[idstr] = &model.AgentMetadata{
			Name: req.GetAgentName(), // Set the agent name from the request
			GrpcEndpoint: req.GetAgentGrpcEndpoint(),
			AgentConn: conn, // Store the gRPC connection
		}
	} else {
		log.Debugf("Agent %s already registered, updating metadata", idstr)
		// Update the agent metadata if it already exists
		s.agents[idstr].Name = req.GetAgentName()
		if s.agents[idstr].GrpcEndpoint != req.GetAgentGrpcEndpoint() {
			// If the gRPC endpoint has changed, we need to create a new connection
			log.Debugf("Agent %s gRPC endpoint changed, updating connection", idstr)
			s.agents[idstr].AgentConn.Close() // Close the old connection
			cred := credentials.NewTLS(s.tlsConfig)
			// Create a gRPC connection to the agent
			conn, err := grpc.NewClient(req.GetAgentGrpcEndpoint(), grpc.WithTransportCredentials(cred))
			if err != nil {
				return nil, fmt.Errorf("failed to create gRPC connection to agent: %w", err)
			}
			s.agents[idstr].AgentConn = conn
		}
		if s.agents[idstr].AgentConn == nil {
			// If the connection is nil, create a new one
			log.Debugf("Agent %s connection is nil, creating a new one", idstr)
			cred := credentials.NewTLS(s.tlsConfig)
			// Create a gRPC connection to the agent
			conn, err := grpc.NewClient(req.GetAgentGrpcEndpoint(), grpc.WithTransportCredentials(cred))
			if err != nil {
				return nil, fmt.Errorf("failed to create gRPC connection to agent: %w", err)
			}
			s.agents[idstr].AgentConn = conn
		}
		s.agents[idstr].GrpcEndpoint = req.GetAgentGrpcEndpoint()
	}

	return &pbc.HeartbeatResponse{
		Success: proto.Bool(true),
		Id: proto.String(id),
		Timestamp: proto.Int64(time.Now().Unix()),
	}, nil
}

