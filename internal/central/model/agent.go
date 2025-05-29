package model

import "google.golang.org/grpc"

type AgentMetadata struct {
	Name string
	GrpcEndpoint string

	AgentConn *grpc.ClientConn
}
