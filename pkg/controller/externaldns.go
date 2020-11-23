package controller

import (
	"errors"
	"fmt"
	"github.com/oslokommune/okctl/pkg/api"
	"github.com/oslokommune/okctl/pkg/client"
)

type ExternalDNSResourceState struct {
	HostedZoneID string
	Domain string
}

type externalDNSReconsiler struct {
	commonMetadata *CommonMetadata
	
	client client.ExternalDNSService
}

func (z *externalDNSReconsiler) SetCommonMetadata(metadata *CommonMetadata) {
	z.commonMetadata = metadata
}

// Reconsile knows how to ensure the desired state is achieved
func (z *externalDNSReconsiler) Reconsile(node *SynchronizationNode) (*ReconsilationResult, error) {
	resourceState, ok := node.ResourceState.(ExternalDNSResourceState)
	if !ok {
		return nil, errors.New("error casting External DNS resourceState")
	}

	switch node.State {
	case SynchronizationNodeStatePresent:
		_, err := z.client.CreateExternalDNS(z.commonMetadata.Ctx, client.CreateExternalDNSOpts{
			ID:           api.ID{},
			HostedZoneID: resourceState.HostedZoneID,
			Domain:       resourceState.Domain,
		})
		if err != nil {
			return &ReconsilationResult{Requeue: true}, fmt.Errorf("error creating external DNS: %w", err)
		}
	case SynchronizationNodeStateAbsent:
		err := z.client.DeleteExternalDNS(z.commonMetadata.Ctx, z.commonMetadata.Id)
		if err != nil {
			return &ReconsilationResult{Requeue: true}, fmt.Errorf("error deleting external DNS: %w", err)
		}
	}


	return &ReconsilationResult{Requeue: false}, nil
}

func NewExternalDNSReconsiler(client client.ExternalDNSService) *externalDNSReconsiler {
	return &externalDNSReconsiler{
		client: client,
	}
}
