package server

import (
	"context"
	. "github.com/Coosis/go-k8s-cord/internal/agent/cluster"
	pba "github.com/Coosis/go-k8s-cord/internal/pb/agent/v1"
)

func(s *AgentServer) ListPods(ctx context.Context, req *pba.ListPodsRequest) (*pba.ListPodsResponse, error) {
	pods, err := GetPods(ctx, s.k8sClientSet, "default")
	if err != nil {
		return nil, err
	}

	return &pba.ListPodsResponse{
		Pods: pods,
	}, nil
}
