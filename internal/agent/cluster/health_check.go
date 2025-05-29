package cluster

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

func Healthz(
	ctx context.Context,
	clientset kubernetes.Clientset,
) (string, error) {
	log.Info("Checking API-server health...")
	resp, err := clientset.Discovery().
		RESTClient().
		Get().
		AbsPath("/healthz").
		Param("verbose", "").
		Do(ctx).
		Raw()
	if err != nil {
		return "", fmt.Errorf("failed to get healthz: %w", err)
	}
	return string(resp), nil
}

func Livez(
	ctx context.Context,
	clientset kubernetes.Clientset,
) (string, error) {
	log.Info("Checking API-server livez...")
	resp, err := clientset.Discovery().
		RESTClient().
		Get().
		AbsPath("/livez").
		Param("verbose", "").
		Do(ctx).
		Raw()
	if err != nil {
		return "", fmt.Errorf("failed to get livez: %w", err)
	}
	return string(resp), nil
}

func Readyz(
	ctx context.Context,
	clientset kubernetes.Clientset,
) (string, error) {
	log.Info("Checking API-server readyz...")
	resp, err := clientset.Discovery().
		RESTClient().
		Get().
		AbsPath("/readyz").
		Param("verbose", "").
		Do(ctx).
		Raw()
	if err != nil {
		return "", fmt.Errorf("failed to get readyz: %w", err)
	}
	return string(resp), nil
}
