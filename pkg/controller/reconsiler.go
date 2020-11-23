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
func (receiver *reconsilerManager) AddReconsiler(key SynchronizationNodeType, r Reconsiler) {
	r.SetCommonMetadata(receiver.commonMetadata)
	
	receiver.reconsilers[key] = r
}

// Reconsile chooses the correct reconsiler to use based on a nodes type
func (receiver *reconsilerManager) Reconsile(node *SynchronizationNode)	(*ReconsilationResult, error)  {
	node.refreshState()
	
	return receiver.reconsilers[node.Type].Reconsile(node)
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

// NoopReconsiler handles reconsiliation for meta nodes (e.g. the root node)
type NoopReconsiler struct {}

func (receiver *NoopReconsiler) SetCommonMetadata(_ *CommonMetadata) {}
func (receiver *NoopReconsiler) Reconsile(_ *SynchronizationNode) (*ReconsilationResult, error) {
	return &ReconsilationResult{Requeue: false}, nil
}
