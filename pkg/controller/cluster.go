package controller

import (
	"errors"
	"fmt"
	"github.com/oslokommune/okctl/pkg/api"
	"github.com/oslokommune/okctl/pkg/client"
)

// ClusterResourceState contains runtime data needed in Reconsile()
type ClusterResourceState struct {
	VPC api.Vpc
}

type clusterReconsiler struct {
	commonMetadata *CommonMetadata

	client client.ClusterService
}

// SetCommonMetadata saves common metadata for use in Reconsile()
func (z *clusterReconsiler) SetCommonMetadata(metadata *CommonMetadata) {
	z.commonMetadata = metadata
}

// Reconsile knows how to ensure the desired state is achieved
func (z *clusterReconsiler) Reconsile(node *SynchronizationNode) (*ReconsilationResult, error) {
	resourceState, ok := node.ResourceState.(ClusterResourceState)
	if !ok {
		return nil, errors.New("error casting cluster resourceState")
	}

	switch node.State {
	case SynchronizationNodeStatePresent:
		_, err := z.client.CreateCluster(z.commonMetadata.Ctx, api.ClusterCreateOpts{
			ID:                z.commonMetadata.Id,
			Cidr:              resourceState.VPC.Cidr,
			VpcID:             resourceState.VPC.VpcID,
			VpcPrivateSubnets: resourceState.VPC.PrivateSubnets,
			VpcPublicSubnets:  resourceState.VPC.PublicSubnets,
		})
		if err != nil {
			return &ReconsilationResult{Requeue: true}, fmt.Errorf("error creating cluster: %w", err)
		}
	case SynchronizationNodeStateAbsent:
		err := z.client.DeleteCluster(z.commonMetadata.Ctx, api.ClusterDeleteOpts{ID: z.commonMetadata.Id})
		if err != nil {
			return &ReconsilationResult{Requeue: true}, fmt.Errorf("error deleting cluster: %w", err)
		}
	}

	return &ReconsilationResult{Requeue: false}, nil
}

// NewClusterReconsiler creates a new reconsiler for the cluster resource
func NewClusterReconsiler(client client.ClusterService) *clusterReconsiler {
	return &clusterReconsiler{
		client: client,
	}
}

