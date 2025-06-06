package server

import (
	"context"
	"path/filepath"

	"github.com/gogo/protobuf/proto"

	. "github.com/Coosis/go-k8s-cord/internal/agent/cluster"
	. "github.com/Coosis/go-k8s-cord/internal/agent/model"
	. "github.com/Coosis/go-k8s-cord/internal/agent/deployment"
	pba "github.com/Coosis/go-k8s-cord/internal/pb/agent/v1"
)

func(s *AgentServer) ListDeployments(
	ctx context.Context,
	req *pba.ListDeploymentsRequest,
) (*pba.ListDeploymentsResponse, error) {
	deployments, err := GetDeployments(ctx, s.k8sClientSet, "default")
	if err != nil {
		return nil, err
	}

	return &pba.ListDeploymentsResponse{
		Deployments: deployments,
	}, nil
}

func(s *AgentServer) ApplyDeployments(
	ctx context.Context,
	req *pba.ApplyDeploymentsRequest,
) (*pba.ApplyDeploymentsResponse, error) {
	cfg := GetAgentConfig()
	paths := make([]string, 0, len(req.DeploymentName))
	for _, deployment := range req.DeploymentName {
		path := filepath.Join(cfg.DeploymentDir, deployment)
		paths = append(paths, path)
	}
	err := ApplyDeployments(ctx, s.k8sClientSet, "default", paths)
	if err != nil {
		return nil, err
	}

	return &pba.ApplyDeploymentsResponse{
		Success: proto.Bool(true),
	}, nil
}

func(s *AgentServer) RemoveDeployments(
	ctx context.Context,
	req *pba.RemoveDeploymentsRequest,
) (*pba.RemoveDeploymentsResponse, error) {
	paths := make([]string, 0, len(req.DeploymentName))
	for _, deployment := range req.DeploymentName {
		paths = append(paths, deployment)
	}
	err := RemoveDeployments(ctx, s.k8sClientSet, "default", paths)
	if err != nil {
		return nil, err
	}

	return &pba.RemoveDeploymentsResponse{
		Success: proto.Bool(true),
	}, nil
}

func(s *AgentServer) GetDeploymentsHash(
	ctx context.Context,
	req *pba.GetDeploymentsHashRequest,
) (*pba.GetDeploymentsHashResponse, error) {
	cfg := GetAgentConfig()
	hash, err := DeploymentHash(cfg.DeploymentDir)
	if err != nil {
		return nil, err
	}

	return &pba.GetDeploymentsHashResponse{
		Hash: proto.String(hash),
	}, nil
}
