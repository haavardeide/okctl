package controller

import (
	"fmt"
	"github.com/oslokommune/okctl/pkg/controller/reconsiler"
	"github.com/oslokommune/okctl/pkg/controller/resourcetree"
)

// Synchronize knows how to discover differences between desired and actual state and rectify them
func Synchronize(reconsilerManager *reconsiler.ReconsilerManager, desiredTree *resourcetree.ResourceNode, currentTree *resourcetree.ResourceNode) error {
	diffGraph := *desiredTree

	diffGraph.ApplyFunction(applyCurrentState, currentTree)
	
	return handleNode(reconsilerManager, &diffGraph)
}

// handleNode knows how to run Reconsile() on every node of a ResourceNode tree
func handleNode(reconsilerManager *reconsiler.ReconsilerManager, currentNode *resourcetree.ResourceNode) error {
	_, err := reconsilerManager.Reconsile(currentNode)
	if err != nil {
	    return fmt.Errorf("error reconsiling node: %w", err)
	}

	for _, node := range currentNode.Children {
		err = handleNode(reconsilerManager, node)
		if err != nil {
		    return fmt.Errorf("error handling node: %w", err)
		}
	}
	
	return nil
}

// applyCurrentState knows how to apply the current state on a desired state to produce a diff that knows which
// resources to create, and which resources is already existing
func applyCurrentState(receiver *resourcetree.ResourceNode, target *resourcetree.ResourceNode) {
	if receiver.State == target.State {
		receiver.State = resourcetree.ResourceNodeStateNoop
	}
}

