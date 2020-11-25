package controller

// noopMetadata contains data known at initialization. Usually information from the desired state
type noopMetadata struct {}

// noopResourceState contains data that potentially can only be known at runtime. E.g.: state only known after an
// external resource has been created
type noopResourceState struct {}

// NoopReconsiler handles reconsiliation for dummy nodes (e.g. the root node) and acts as a template for other
// reconsilers
type NoopReconsiler struct {}

// SetCommonMetadata knows how to store common metadata on the reconsiler. This should do nothing if common metadata is
// not needed
func (receiver *NoopReconsiler) SetCommonMetadata(_ *CommonMetadata) {}

// Reconsile knows how to create, update and delete the relevant resource
func (receiver *NoopReconsiler) Reconsile(node *SynchronizationNode) (*ReconsilationResult, error) {
	//metadata, ok := node.Metadata.(noopMetadata)
	//if !ok {
	//	return nil, errors.New("could not cast Noop metadata")
	//}
	//
	//state, ok := node.ResourceState.(noopResourceState)
	//if !ok {
	//	return nil, errors.New("could not cast Noop resource state")
	//}

	switch node.State {
	case SynchronizationNodeStatePresent:
		// Create a resource
	case SynchronizationNodeStateAbsent:
		// Delete a resource
	}

	return &ReconsilationResult{Requeue: false}, nil
}
