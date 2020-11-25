package controller

/*
 * Reconsiler
 */

type ReconsilationResult struct {
	Requeue bool
}

type Reconsiler interface {
	// Reconsile knows how to ensure the desired state is achieved
	Reconsile(*SynchronizationNode) (*ReconsilationResult, error)
	SetCommonMetadata(metadata *CommonMetadata)
}

/*
reconsilerManager provides a simpler way to organize reconsilers
*/
type reconsilerManager struct {
	commonMetadata *CommonMetadata
	reconsilers map[SynchronizationNodeType]Reconsiler
}

// AddReconsiler makes a reconsiler available in the reconsilerManager
func (manager *reconsilerManager) AddReconsiler(key SynchronizationNodeType, reconsiler Reconsiler) {
	reconsiler.SetCommonMetadata(manager.commonMetadata)
	
	manager.reconsilers[key] = reconsiler
}

// Reconsile chooses the correct reconsiler to use based on a nodes type
func (manager *reconsilerManager) Reconsile(node *SynchronizationNode)	(*ReconsilationResult, error)  {
	node.refreshState()
	
	return manager.reconsilers[node.Type].Reconsile(node)
}

// NewReconsilerManager creates a new reconsilerManager with a NoopReconsiler already installed
func NewReconsilerManager(metadata *CommonMetadata) *reconsilerManager {
	return &reconsilerManager{
		commonMetadata: metadata,
		reconsilers: map[SynchronizationNodeType]Reconsiler{
			SynchronizationNodeTypeNoop: &NoopReconsiler{},
		},
	}
}
