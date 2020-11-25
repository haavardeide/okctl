package controller

import (
	"context"
	"fmt"
	"github.com/oslokommune/okctl/pkg/api"
)

// SynchronizationNodeType defines what type of resource a SynchronizationNode represents
type SynchronizationNodeType int
const (
	// SynchronizationNodeTypeNoop represents a node that has no actions associated with it. For now, only the root node
	SynchronizationNodeTypeNoop SynchronizationNodeType = iota // TODO: NodeTypeGroup
	// SynchronizationNodeTypeZone represents a HostedZone resource
	SynchronizationNodeTypeZone
	// SynchronizationNodeTypeVPC represents a VPC resource
	SynchronizationNodeTypeVPC
	// SynchronizationNodeTypeCluster represents a EKS cluster resource
	SynchronizationNodeTypeCluster
	// SynchronizationNodeTypeExternalSecrets represents an External Secrets resource
	SynchronizationNodeTypeExternalSecrets
	// SynchronizationNodeTypeALBIngress represents an ALB Ingress resource
	SynchronizationNodeTypeALBIngress
	// SynchronizationNodeTypeExternalDNS represents an External DNS resource
	SynchronizationNodeTypeExternalDNS
	// SynchronizationNodeTypeGithub represents a Github setup
	SynchronizationNodeTypeGithub
	// SynchronizationNodeTypeIdentityManager represents a Identity Manager resource
	SynchronizationNodeTypeIdentityManager
	// SynchronizationNodeTypeArgoCD represents an ArgoCD resource
	SynchronizationNodeTypeArgoCD
)

// SynchronizationNodeState defines what state the resource is in, used to infer what action to take
type SynchronizationNodeState int
const (
	// SynchronizationNodeStateNoop represents a state where no action is needed. E.g.: if the desired state of the
	// resource conforms with the actual state
	SynchronizationNodeStateNoop SynchronizationNodeState = iota
	// SynchronizationNodeStatePresent represents the state where the resource exists
	SynchronizationNodeStatePresent
	// SynchronizationNodeStateAbsent represents the state where the resource does not exist
	SynchronizationNodeStateAbsent
)

// CommonMetadata represents metadata required by most if not all operations on services
type CommonMetadata struct {
	Ctx context.Context
	Id api.ID
}

// StateRefreshFn is a function that attempts to retrieve state potentially can only be retrieved at runtime. E.g.:
// state that can only exist after an external resource has been created
type StateRefreshFn func(node *SynchronizationNode)

// SynchronizationNode represents a component of the cluster and its dependencies
type SynchronizationNode struct {
	Type SynchronizationNodeType
	State SynchronizationNodeState

	// Contains metadata regarding the resource supplied by the desired state definition
	Metadata             interface{}

	StateRefresher 		 StateRefreshFn
	// ResourceState contains data that needs to be retrieved runtime. In other words, data that possibly can only exist
	// after an external resource has been created
	ResourceState 		 interface{}

	Children []*SynchronizationNode
}

func (receiver *SynchronizationNode) refreshState() {
	if receiver.StateRefresher == nil {
		return
	}

	receiver.StateRefresher(receiver)
}

func (receiver *SynchronizationNode) SetStateRefresher(nodeType SynchronizationNodeType, refresher StateRefreshFn) {
	targetNode := receiver.GetNode(&SynchronizationNode{Type: nodeType})

	if targetNode == nil {
		return
	}

	targetNode.StateRefresher = refresher
}

// Equals knows how to compare two SynchronizationNodes
func (receiver *SynchronizationNode) Equals(node *SynchronizationNode) bool {
	if node == nil {
		return false
	}
	
	return node.Type == receiver.Type // TODO: should allow for multiple instances of same typed nodes
}

// GetNode returns an identical node as node from the receiver's tree
func (receiver *SynchronizationNode) GetNode(node *SynchronizationNode) *SynchronizationNode {
	if receiver.Equals(node) {
		return receiver
	}
	
	for _, child := range receiver.Children {
		result := child.GetNode(node)

		if result != nil {
			return result
		}
	}
	
	return nil
}

func (receiver *SynchronizationNode) Apply(state *SynchronizationNode) {
	for _, child := range receiver.Children {
		child.Apply(state)
	}
	
	relevantNode := state.GetNode(receiver)
	if relevantNode == nil {
		receiver.State = SynchronizationNodeStatePresent
	} else {
		receiver.State = SynchronizationNodeStateNoop
	}
}

type ApplyFn func(receiver *SynchronizationNode, target *SynchronizationNode)

func (receiver *SynchronizationNode) ApplyFunction(fn ApplyFn, targetGraph *SynchronizationNode) {
	for _, child := range receiver.Children {
		child.ApplyFunction(fn, targetGraph)
	}
	
	targetNode := targetGraph.GetNode(receiver)
	fn(receiver, targetNode)
}

func ApplyCurrentState(receiver *SynchronizationNode, target *SynchronizationNode) {
	if receiver.State == target.State {
		receiver.State = SynchronizationNodeStateNoop
	}
}

// Synchronize knows how to discover differences between desired and actual state and rectify them
func Synchronize(reconsilerManager *reconsilerManager, desiredGraph *SynchronizationNode, currentGraph *SynchronizationNode) error {
	diffGraph := *desiredGraph

	//desiredGraph.Apply(currentGraph)
	diffGraph.ApplyFunction(ApplyCurrentState, currentGraph)
	//currentGraph.Apply(desiredGraph)
	
	return handleNode(reconsilerManager, &diffGraph)
}

// handleNode knows how to run Reconsile() on every node of a graph
func handleNode(reconsilerManager *reconsilerManager, currentNode *SynchronizationNode) error {
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
