package controller

import (
	"fmt"
	"github.com/oslokommune/okctl/pkg/client"
)

type externalSecretsReconsiler struct {
	commonMetadata *CommonMetadata
	
	client client.ExternalSecretsService
}

// SetCommonMetadata saves common metadata for use in Reconsile()
func (z *externalSecretsReconsiler) SetCommonMetadata(metadata *CommonMetadata) {
	z.commonMetadata = metadata
}

// Reconsile knows how to ensure the desired state is achieved
func (z *externalSecretsReconsiler) Reconsile(node *SynchronizationNode) (*ReconsilationResult, error) {
	switch node.State {
	case SynchronizationNodeStatePresent:
		_, err := z.client.CreateExternalSecrets(z.commonMetadata.Ctx, client.CreateExternalSecretsOpts{ID: z.commonMetadata.Id})
		
		if err != nil {
			return &ReconsilationResult{Requeue: true}, fmt.Errorf("error creating external secrets: %w", err)
		}
	case SynchronizationNodeStateAbsent:
		err := z.client.DeleteExternalSecrets(z.commonMetadata.Ctx, z.commonMetadata.Id)

		if err != nil {
			return &ReconsilationResult{Requeue: true}, fmt.Errorf("error deleting external secrets: %w", err)
		}
	}

	return &ReconsilationResult{Requeue: false}, nil
}

// NewExternalSecretsReconsiler creates a new reconsiler for the ExternalSecrets resource
func NewExternalSecretsReconsiler(client client.ExternalSecretsService) *externalSecretsReconsiler {
	return &externalSecretsReconsiler{
		client: client,
	}
}

