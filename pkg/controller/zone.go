package controller

import (
	"fmt"
	"github.com/oslokommune/okctl/pkg/client"
)

type zoneReconsiler struct {
	commonMetadata *CommonMetadata
	
	client client.DomainService
}

func (z *zoneReconsiler) SetCommonMetadata(metadata *CommonMetadata) {
	z.commonMetadata = metadata
}

// Reconsile knows how to ensure the desired state is achieved
func (z *zoneReconsiler) Reconsile(node *SynchronizationNode) (*ReconsilationResult, error) {
	switch node.State {
	case SynchronizationNodeStatePresent:
		_, err := z.client.CreatePrimaryHostedZoneWithoutUserinput(z.commonMetadata.Ctx, client.CreatePrimaryHostedZoneOpts{
			ID:     z.commonMetadata.Id,
		})
		if err != nil {
			return &ReconsilationResult{Requeue: true}, fmt.Errorf("error creating hosted zone: %w", err)
		}
	case SynchronizationNodeStateAbsent:
		err := z.client.DeletePrimaryHostedZone(z.commonMetadata.Ctx, client.DeletePrimaryHostedZoneOpts{ID: z.commonMetadata.Id})
		if err != nil {
		    return &ReconsilationResult{Requeue: true}, fmt.Errorf("error deleting hosted zone: %w", err)
		}
	}

	return &ReconsilationResult{Requeue: false}, nil
}

func NewZoneReconsiler(client client.DomainService) *zoneReconsiler {
	return &zoneReconsiler{
		client: client,
	}
}

