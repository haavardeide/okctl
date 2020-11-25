package controller

import (
	"fmt"
	"github.com/miekg/dns"
	"github.com/mishudark/errors"
	"github.com/oslokommune/okctl/pkg/client"
)

type HostedZoneMetadata struct {
	Domain string
}

type zoneReconsiler struct {
	commonMetadata *CommonMetadata
	
	client client.DomainService
}

func (z *zoneReconsiler) SetCommonMetadata(metadata *CommonMetadata) {
	z.commonMetadata = metadata
}

// Reconsile knows how to ensure the desired state is achieved
func (z *zoneReconsiler) Reconsile(node *SynchronizationNode) (*ReconsilationResult, error) {
	metadata, ok := node.Metadata.(HostedZoneMetadata)
	if !ok {
		return nil, errors.New("error casting HostedZone metadata")
	}
	
	switch node.State {
	case SynchronizationNodeStatePresent:
		fqdn := dns.Fqdn(metadata.Domain)

		_, err := z.client.CreatePrimaryHostedZoneWithoutUserinput(z.commonMetadata.Ctx, client.CreatePrimaryHostedZoneOpts{
			ID:     z.commonMetadata.Id,
			Domain: metadata.Domain,
			FQDN: fqdn,
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

