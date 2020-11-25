
package controller

import (
	"fmt"
	"github.com/miekg/dns"
	"github.com/mishudark/errors"
	"github.com/oslokommune/okctl/pkg/api"
	"github.com/oslokommune/okctl/pkg/client"
)

type IdentityManagerResourceState struct {
	HostedZoneID string
	Domain string
}

type identityManagerReconsiler struct {
	commonMetadata *CommonMetadata

	client client.IdentityManagerService
}

func (z *identityManagerReconsiler) SetCommonMetadata(metadata *CommonMetadata) {
	z.commonMetadata = metadata
}

/*
Reconsile knows how to ensure the desired state is achieved
Requires:
- Hosted Zone
- Nameservers setup
 */
func (z *identityManagerReconsiler) Reconsile(node *SynchronizationNode) (*ReconsilationResult, error) {
	resourceState, ok := node.ResourceState.(IdentityManagerResourceState)
	if !ok {
		return nil, errors.New("unable to cast identity manager resourceState")
	}

	switch node.State {
	case SynchronizationNodeStatePresent:
		authDomain := fmt.Sprintf("auth.%s", resourceState.Domain)
		authFQDN := dns.Fqdn(authDomain)
		
		_, err := z.client.CreateIdentityPool(z.commonMetadata.Ctx, api.CreateIdentityPoolOpts{
			ID:           z.commonMetadata.Id,
			AuthDomain:   authDomain,
			AuthFQDN:     authFQDN,
			HostedZoneID: resourceState.HostedZoneID,
		})
		if err != nil {
			return &ReconsilationResult{Requeue: true}, fmt.Errorf("error creating identity manager resource: %w", err)
		}
	case SynchronizationNodeStateAbsent:
		return nil, errors.New("deleting identity manager resource is not implemented")
	}

	return &ReconsilationResult{Requeue: false}, nil
}

func NewIdentityManagerReconsiler(client client.IdentityManagerService) *identityManagerReconsiler {
	return &identityManagerReconsiler{
		client: client,
	}
}
