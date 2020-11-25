
package controller

import (
	"fmt"
	"github.com/mishudark/errors"
	"github.com/oslokommune/okctl/pkg/client"
	"github.com/oslokommune/okctl/pkg/config/state"
)

type ArgocdMetadata struct {
	Organization string
}

type ArgocdResourceState struct {
	HostedZone *state.HostedZone
	Repository *client.GithubRepository
	
	UserPoolID string
	AuthDomain string
}

type argocdReconsiler struct {
	commonMetadata *CommonMetadata

	client client.ArgoCDService
}

func (z *argocdReconsiler) SetCommonMetadata(metadata *CommonMetadata) {
	z.commonMetadata = metadata
}

/*
Reconsile knows how to ensure the desired state is achieved.
Dependent on:
- Github repo setup
- Cognito user pool
- Primary hosted Zone
 */
func (z *argocdReconsiler) Reconsile(node *SynchronizationNode) (*ReconsilationResult, error) {
	resourceState, ok := node.ResourceState.(ArgocdResourceState)
	if !ok {
		return nil, errors.New("error casting argocd resource resourceState")
	}
	
	//repository := client.GithubRepository{
	//	ID:           z.commonMetadata.Id,
	//	Organisation: "",
	//	Repository:   z.commonMetadata.Id.Repository,
	//	FullName:     "",
	//	GitURL:       "",
	//	DeployKey:    nil,
	//}

	switch node.State {
	case SynchronizationNodeStatePresent:
		_, err := z.client.CreateArgoCD(z.commonMetadata.Ctx, client.CreateArgoCDOpts{
			ID:                 z.commonMetadata.Id,
			Domain:             resourceState.HostedZone.Domain,
			FQDN:               resourceState.HostedZone.FQDN,
			HostedZoneID:       resourceState.HostedZone.ID,
			GithubOrganisation: resourceState.Repository.Organisation,
			UserPoolID:         resourceState.UserPoolID,
			AuthDomain:         resourceState.AuthDomain,
			Repository:         resourceState.Repository,
		})
		if err != nil {
			return &ReconsilationResult{Requeue: true}, fmt.Errorf("error creating argocd: %w", err)
		}
	case SynchronizationNodeStateAbsent:
		return nil, errors.New("deletion of the argocd resource is not implemented")
	}

	return &ReconsilationResult{Requeue: false}, nil
}

func NewArgocdReconsiler(client client.ArgoCDService) *argocdReconsiler {
	return &argocdReconsiler{
		client: client,
	}
}
