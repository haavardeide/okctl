package controller

import (
	"fmt"
	"github.com/oslokommune/okctl/pkg/controller/reconsiler"
	"github.com/oslokommune/okctl/pkg/controller/resourcetree"
)

// Synchronize knows how to discover differences between desired and actual state and rectify them
func Synchronize(reconsilerManager *reconsiler.ReconsilerManager, desiredGraph *resourcetree.SynchronizationNode, currentGraph *resourcetree.SynchronizationNode) error {
	diffGraph := *desiredGraph

	//desiredGraph.Apply(currentGraph)
	diffGraph.ApplyFunction(ApplyCurrentState, currentGraph)
	//currentGraph.Apply(desiredGraph)
	
	return handleNode(reconsilerManager, &diffGraph)
}

// handleNode knows how to run Reconsile() on every node of a graph
func handleNode(reconsilerManager *reconsiler.ReconsilerManager, currentNode *resourcetree.SynchronizationNode) error {
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

func ApplyCurrentState(receiver *resourcetree.SynchronizationNode, target *resourcetree.SynchronizationNode) {
	if receiver.State == target.State {
		receiver.State = resourcetree.SynchronizationNodeStateNoop
	}
}

