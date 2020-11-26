package controller

import (
	"fmt"
	"github.com/mishudark/errors"
	"github.com/oslokommune/okctl/pkg/apis/okctl.io/v1alpha1"
	"github.com/oslokommune/okctl/pkg/config"
	"github.com/oslokommune/okctl/pkg/controller/reconsiler"
	"github.com/oslokommune/okctl/pkg/controller/resourcetree"
	"github.com/oslokommune/okctl/pkg/git"
	"github.com/spf13/afero"
	"path"
)

type existingServices struct {
	hasALBIngressController bool
	hasCluster bool
	hasExternalDNS bool
	hasExternalSecrets bool
	hasGithubSetup bool
	hasIdentityManager bool
	hasPrimaryHostedZone bool
	hasVPC bool
}

// NewCreateCurrentStateGraphOpts creates an initialized existingServices struct
func NewCreateCurrentStateGraphOpts(fs *afero.Afero, outputDir string) (*existingServices, error) {
	var err error

	// TODO: Create tester for .okctl.yml only information. Github state gets stored there
	
	tester := func(target string) (exists bool) {
		baseDir := path.Join(outputDir, target)
		
		exists, err = fs.DirExists(baseDir)
		
		return exists
	}

	return &existingServices{
		hasPrimaryHostedZone:    tester(config.DefaultDomainBaseDir),
		hasVPC:                  tester(config.DefaultVpcBaseDir),
		hasCluster:              tester(config.DefaultClusterBaseDir),
		hasExternalSecrets:      tester(config.DefaultExternalSecretsBaseDir),
		hasALBIngressController: tester(config.DefaultAlbIngressControllerBaseDir),
		hasExternalDNS:          tester(config.DefaultExternalDNSBaseDir),
		hasIdentityManager: 	 tester(config.DefaultIdentityPoolBaseDir),
	}, err
}

// CreateCurrentStateGraph knows how to generate a ResourceNode tree based on the current state
func CreateCurrentStateGraph(opts *existingServices) (root *resourcetree.ResourceNode) {
	root = createNode(nil, resourcetree.ResourceNodeTypeGroup, true)
	
	var (
		vpcNode,
		clusterNode *resourcetree.ResourceNode
	)
	
	createNode(root, resourcetree.ResourceNodeTypeZone, opts.hasPrimaryHostedZone)
	createNode(root, resourcetree.ResourceNodeTypeGithub, false)
	vpcNode = createNode(root, resourcetree.ResourceNodeTypeVPC, opts.hasVPC)

	clusterNode = createNode(vpcNode, resourcetree.ResourceNodeTypeCluster, opts.hasCluster)

	createNode(clusterNode, resourcetree.ResourceNodeTypeExternalSecrets, opts.hasExternalSecrets)
	createNode(clusterNode, resourcetree.ResourceNodeTypeALBIngress, opts.hasALBIngressController)
	createNode(clusterNode, resourcetree.ResourceNodeTypeExternalDNS, opts.hasExternalDNS)

	return root
}

// CreateDesiredStateGraph knows how to create a ResourceNode tree based on a cluster declaration
func CreateDesiredStateGraph(cluster *v1alpha1.Cluster) (root *resourcetree.ResourceNode) {
	root = createNode(nil, resourcetree.ResourceNodeTypeGroup, true)

	var (
		vpcNode,
		clusterNode *resourcetree.ResourceNode
	)
	
	if len(cluster.DNSZones) > 0 {
		for range cluster.DNSZones { // TODO: not gonna work. More than one will generate multiple primaries
			createNode(root, resourcetree.ResourceNodeTypeZone, true)
		}
	}

	createNode(root, resourcetree.ResourceNodeTypeGithub, true)
	vpcNode = createNode(root, resourcetree.ResourceNodeTypeVPC, true)

	clusterNode = createNode(vpcNode, resourcetree.ResourceNodeTypeCluster, true)

	createNode(clusterNode, resourcetree.ResourceNodeTypeExternalSecrets, cluster.Integrations.ExternalSecrets)
	createNode(clusterNode, resourcetree.ResourceNodeTypeALBIngress, cluster.Integrations.ALBIngressController)
	createNode(clusterNode, resourcetree.ResourceNodeTypeExternalDNS, cluster.Integrations.ExternalDNS) // TODO: Needs to be dependent on primary hosted zone

	return root
}

// ApplyDesiredStateMetadata applies metadata from a cluster definition to the nodes
func ApplyDesiredStateMetadata(graph *resourcetree.ResourceNode, cluster *v1alpha1.Cluster, repoDir string) error {
	// TODO: Fetch cluster first and fetch hosted zone from cluster to ensure primary hosted zone is fetched after
	// moving primary hosted zone
	primaryHostedZoneNode := graph.GetNode(&resourcetree.ResourceNode{ Type: resourcetree.ResourceNodeTypeZone})
	if primaryHostedZoneNode == nil {
		return errors.New("expected primary hosted zone node was not found")
	}

	primaryHostedZoneNode.Metadata = reconsiler.HostedZoneMetadata{Domain: cluster.DNSZones[0].ParentDomain}

	vpcNode := graph.GetNode(&resourcetree.ResourceNode{ Type: resourcetree.ResourceNodeTypeVPC})
	if vpcNode == nil {
		return errors.New("expected vpc node was not found")
	}
	
	vpcNode.Metadata = reconsiler.VPCMetadata{
		Cidr:             cluster.VPC.CIDR,
		HighAvailability: cluster.VPC.HighAvailability,
	}
	
	githubNode := graph.GetNode(&resourcetree.ResourceNode{ Type: resourcetree.ResourceNodeTypeGithub})
	if githubNode == nil {
		return errors.New("expected github node was not found")
	}

	repo, err := git.GithubRepoFullName(cluster.Github.Organisation, repoDir)
	if err != nil {
			  return fmt.Errorf("error fetching full git repo name: %w", err)
			  }

	githubNode.Metadata = reconsiler.GithubMetadata{
		Organization: cluster.Github.Organisation,
		Repository:   repo,
	}

	argocdNode := graph.GetNode(&resourcetree.ResourceNode{ Type: resourcetree.ResourceNodeTypeArgoCD})
	if argocdNode != nil {
		argocdNode.Metadata = reconsiler.ArgocdMetadata{Organization: cluster.Github.Organisation }
	}

	return nil
}

func createNode(parent *resourcetree.ResourceNode, nodeType resourcetree.ResourceNodeType, present bool) (child *resourcetree.ResourceNode) {
	child = &resourcetree.ResourceNode{
		Type:           nodeType,
		Children:       make([]*resourcetree.ResourceNode, 0),
	}
	
	if present {
		child.State = resourcetree.ResourceNodeStatePresent
	} else {
		child.State = resourcetree.ResourceNodeStateAbsent
	}
	
	if parent != nil {
		parent.Children = append(parent.Children, child)
	}

	return child
}
