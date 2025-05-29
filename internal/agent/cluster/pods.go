package cluster

import (
	"context"

	"k8s.io/client-go/kubernetes"

	"github.com/gogo/protobuf/proto"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pba "github.com/Coosis/go-k8s-cord/internal/pb/agent/v1"
)

func GetPods(
	ctx context.Context,
	client *kubernetes.Clientset,
	namespace string,
) ([]*pba.PodMetadata, error) {
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	// maybe add the following fields to the PodMetadata struct?
	// p.Labels
	// p.Annotations
	// p.OwnerReferences
	// p.Status.Conditions
	metadataList := []*pba.PodMetadata{}
	for _, p := range pods.Items {
		containers := []*pba.ContainerSnapshot{}
		for _, c := range p.Spec.Containers {
			containers = append(containers, &pba.ContainerSnapshot{
				Name: proto.String(c.Name),
				Image: proto.String(c.Image),
			})
		}

		if p.DeletionTimestamp == nil {
			p.DeletionTimestamp = &metav1.Time{
				Time: metav1.Now().Time,
			}
		}

		if p.DeletionGracePeriodSeconds == nil {
			p.DeletionGracePeriodSeconds = proto.Int64(0)
		}

		metadata := &pba.PodMetadata {
			Name: proto.String(p.Name),
			Namespace: proto.String(p.Namespace),
			Uid: proto.String((string)(p.UID)),
			ApiVersion: proto.String(p.APIVersion),
			CreationTimestamp: proto.Int64(p.CreationTimestamp.Unix()),
			DeletionTimestamp: proto.Int64(p.DeletionTimestamp.Unix()),
			DeletionGracePeriodSeconds: p.DeletionGracePeriodSeconds,
			NodeName: proto.String(p.Spec.NodeName),
			Containers: containers,
			Phase: proto.String((string)(p.Status.Phase)),
			PodIp: proto.String(p.Status.PodIP),
		}
		metadataList = append(metadataList, metadata)
	}
	log.Infof("Found %d pods in namespace %s", len(metadataList), namespace)
	return metadataList, nil
}
