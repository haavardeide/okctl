package reconsiler

import (
	"github.com/oslokommune/okctl/pkg/controller/resourcetree"
)

/*
 * Reconsiler
 */

type ReconsilationResult struct {
	Requeue bool
}

type Reconsiler interface {
	// Reconsile knows how to ensure the desired state is achieved
	Reconsile(*resourcetree.SynchronizationNode) (*ReconsilationResult, error)
	SetCommonMetadata(metadata *resourcetree.CommonMetadata)
}

/*
ReconsilerManager provides a simpler way to organize reconsilers
*/
type ReconsilerManager struct {
	commonMetadata *resourcetree.CommonMetadata
	Reconsilers map[resourcetree.SynchronizationNodeType]Reconsiler
}

// AddReconsiler makes a Reconsiler available in the ReconsilerManager
func (manager *ReconsilerManager) AddReconsiler(key resourcetree.SynchronizationNodeType, Reconsiler Reconsiler) {
	Reconsiler.SetCommonMetadata(manager.commonMetadata)
	
	manager.Reconsilers[key] = Reconsiler
}

// Reconsile chooses the correct reconsiler to use based on a nodes type
func (manager *ReconsilerManager) Reconsile(node *resourcetree.SynchronizationNode)	(*ReconsilationResult, error)  {
	node.RefreshState()
	
	return manager.Reconsilers[node.Type].Reconsile(node)
}

// NewReconsilerManager creates a new ReconsilerManager with a NoopReconsiler already installed
func NewReconsilerManager(metadata *resourcetree.CommonMetadata) *ReconsilerManager {
	return &ReconsilerManager{
		commonMetadata: metadata,
		Reconsilers: map[resourcetree.SynchronizationNodeType]Reconsiler{
			resourcetree.SynchronizationNodeTypeNoop: &NoopReconsiler{},
		},
	}
}
