package controller

import (
	"errors"
	"fmt"
	"github.com/oslokommune/okctl/pkg/client"
)

type AlbIngressControllerResourceState struct {
	VpcID string
}

type albIngressReconsiler struct {
	commonMetadata *CommonMetadata
	client client.ALBIngressControllerService
}

// SetCommonMetadata stores common metadata for later use
func (z *albIngressReconsiler) SetCommonMetadata(metadata *CommonMetadata) {
	z.commonMetadata = metadata
}

// Reconsile knows how to ensure the desired state is achieved
func (z *albIngressReconsiler) Reconsile(node *SynchronizationNode) (*ReconsilationResult, error) {
	state, ok := node.ResourceState.(AlbIngressControllerResourceState)
	if !ok {
	    return nil, errors.New("error casting ALB Ingress Controller state")
	}

	switch node.State {
	case SynchronizationNodeStatePresent:
		_, err := z.client.CreateALBIngressController(z.commonMetadata.Ctx, client.CreateALBIngressControllerOpts{
			ID:    z.commonMetadata.Id,
			VPCID: state.VpcID,
		})
		if err != nil {
			return &ReconsilationResult{Requeue: true}, fmt.Errorf("error creating ALB Ingress controller: %w", err)
		}
	case SynchronizationNodeStateAbsent:
		err := z.client.DeleteALBIngressController(z.commonMetadata.Ctx, z.commonMetadata.Id)
		if err != nil {
			return &ReconsilationResult{Requeue: true}, fmt.Errorf("error deleting ALB Ingress controller: %w", err)
		}
	}

	return &ReconsilationResult{Requeue: false}, nil
}

func NewALBIngressReconsiler(client client.ALBIngressControllerService) *albIngressReconsiler {
	return &albIngressReconsiler{
		client: client,
	}
}
