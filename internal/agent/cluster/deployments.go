package cluster

import (
	"context"
	"os"

	"k8s.io/client-go/kubernetes"
	"github.com/gogo/protobuf/proto"

	v1 "k8s.io/client-go/applyconfigurations/apps/v1"
	yaml "sigs.k8s.io/yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	log "github.com/sirupsen/logrus"

	pba "github.com/Coosis/go-k8s-cord/internal/pb/agent/v1"
)

func GetDeployments(
	ctx context.Context,
	client *kubernetes.Clientset,
	namespace string,
) ([]*pba.DeploymentMetadata, error) {
	deployements, err := client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	metadataList := []*pba.DeploymentMetadata{}

	for _, item := range deployements.Items {
		metadata := &pba.DeploymentMetadata{
			ApiVersion:        proto.String(item.APIVersion),
			Uid:               proto.String(string(item.UID)),
			Name:              proto.String(item.Name),
			Replicas:          item.Spec.Replicas,
			ReadyReplicas:     proto.Int32(item.Status.ReadyReplicas),
			AvailableReplicas: proto.Int32(item.Status.AvailableReplicas),
			UpdatedReplicas:   proto.Int32(item.Status.UpdatedReplicas),
			CreationTimestamp: proto.Int64(item.CreationTimestamp.Unix()),
		}
		metadataList = append(metadataList, metadata)
	}
	log.Infof("Found %d deployments in namespace %s", len(metadataList), namespace)
	return metadataList, nil
}

func ApplyDeployments(
	ctx context.Context,
	client *kubernetes.Clientset,
	namespace string,
	deployments []string,
) error {
	for _, deployment := range deployments {
		yamlHandle, err := os.ReadFile(deployment)
		if err != nil {
			log.Errorf("Failed to read deployment file %s: %v", deployment, err)
			continue
		}
		dep := &v1.DeploymentApplyConfiguration{}
		if err := yaml.Unmarshal(yamlHandle, dep); err != nil {
			log.Errorf("Failed to unmarshal deployment file %s: %v", deployment, err)
			continue
		}
		if *dep.Name == "" {
			log.Errorf("Deployment file %s does not contain a valid deployment name", deployment)
			continue
		}

		_, err = client.AppsV1().Deployments(namespace).Apply(ctx, dep, metav1.ApplyOptions{
			FieldManager: "go-k8s-cord-agent",
			Force:        true,
		})
		if err != nil {
			log.Errorf("Failed to apply deployment %s: %v", *dep.Name, err)
			continue
		}
	}
	log.Infof("Applied %d deployments in namespace %s", len(deployments), namespace)
	return nil
}

func RemoveDeployments(
	ctx context.Context,
	client *kubernetes.Clientset,
	namespace string,
	deployments []string,
) error {
	for _, deployment := range deployments {
		if err := client.AppsV1().Deployments(namespace).Delete(ctx, deployment, metav1.DeleteOptions{}); err != nil {
			log.Errorf("Failed to remove deployment %s: %v", deployment, err)
			continue
		}
		log.Infof("Removed deployment %s from namespace %s", deployment, namespace)
	}
	return nil
}
