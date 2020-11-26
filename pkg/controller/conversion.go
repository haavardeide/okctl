package controller

import (
	"fmt"
	"github.com/mishudark/errors"
	"github.com/oslokommune/okctl/pkg/api"
	"github.com/oslokommune/okctl/pkg/apis/okctl.io/v1alpha1"
	"github.com/oslokommune/okctl/pkg/client/store"
	"github.com/oslokommune/okctl/pkg/config"
	"github.com/oslokommune/okctl/pkg/config/state"
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

// CreateCurrentStateGraph knows how to generate a graph based on the current state
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
	//createNode(clusterNode, resourcetree.ResourceNodeTypeIdentityManager, opts.hasIdentityManager)

	return root
}

// CreateDesiredStateGraph knows how to create a graph based on a cluster declaration
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
	createNode(clusterNode, resourcetree.ResourceNodeTypeExternalDNS, cluster.Integrations.ExternalDNS) // TODO: Needs to be dependend on primary hosted zone
	//createNode(clusterNode, ResourceNodeTypeIdentityManager, cluster.Integrations.Cognito) // TODO: ArgoCD is dependent on cognito, but i dont think cognito is dependent on anything other than hosted zone

	return root
}

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
	if githubNode != nil { // TODO: A Github is required, so this should probably break something if not existing
		repo, err := git.GithubRepoFullName(cluster.Github.Organisation, repoDir)
		if err != nil {
		    return fmt.Errorf("error fetching full git repo name: %w", err)
		}

		githubNode.Metadata = reconsiler.GithubMetadata{
			Organization: cluster.Github.Organisation,
			Repository:   repo,
		}
	}
	
	argocdNode := graph.GetNode(&resourcetree.ResourceNode{ Type: resourcetree.ResourceNodeTypeArgoCD})
	if argocdNode != nil {
		argocdNode.Metadata = reconsiler.ArgocdMetadata{Organization: cluster.Github.Organisation }
	}

	// TODO: Do something about cluster? maybe not
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

func getVpcState(fs *afero.Afero, outputDir string) api.Vpc {
	vpc := api.Vpc{}
	
	baseDir := path.Join(outputDir, "vpc")

	_, err := store.NewFileSystem(baseDir, fs).
		GetStruct(config.DefaultVpcOutputs, &vpc, store.FromJSON()).
		Do()
	if err != nil {
		panic(fmt.Errorf("error reading from vpc state file: %w", err))
	}
	
	return vpc
}

type StringFetcher func() string
func CreateClusterStateRefresher(fs *afero.Afero, outputDir string, cidrFn StringFetcher) resourcetree.StateRefreshFn {
	return func(node *resourcetree.ResourceNode) {
		vpc := getVpcState(fs, outputDir)
		
		vpc.Cidr = cidrFn()

		node.ResourceState = reconsiler.ClusterResourceState{VPC: vpc}
	}
}

func CreateALBIngressControllerRefresher(fs *afero.Afero, outputDir string) resourcetree.StateRefreshFn {
	return func(node *resourcetree.ResourceNode) {
		vpc := getVpcState(fs, outputDir)
		
		node.ResourceState = reconsiler.AlbIngressControllerResourceState{VpcID: vpc.VpcID}
	}
}

func CreateExternalDNSStateRefresher(domainFetcher StringFetcher, hostedZoneIDFetcher StringFetcher) resourcetree.StateRefreshFn {
	return func(node *resourcetree.ResourceNode) {
		node.ResourceState = reconsiler.ExternalDNSResourceState{
			HostedZoneID: hostedZoneIDFetcher(),
			Domain:       domainFetcher(),
		}
	}
}

func CreateIdentityManagerRefresher(domainFetcher StringFetcher, hostedZoneIDFetcher StringFetcher) resourcetree.StateRefreshFn {
	return func(node *resourcetree.ResourceNode) {
		node.ResourceState = reconsiler.IdentityManagerResourceState{
			HostedZoneID: hostedZoneIDFetcher(),
			Domain:       domainFetcher(),
		}
	}
}

func CreateGithubStateRefresher(ghGetter reconsiler.GithubGetter, ghSetter reconsiler.GithubSetter) resourcetree.StateRefreshFn {
	return func(node *resourcetree.ResourceNode) {
		node.ResourceState = reconsiler.GithubResourceState{
			Getter: ghGetter,
			Saver: ghSetter,
		}
	}
}

type HostedZoneFetcher func() *state.HostedZone
func CreateArgocdStateRefresher(hostedZoneFetcher HostedZoneFetcher) resourcetree.StateRefreshFn {
	return func(node *resourcetree.ResourceNode) {
		node.ResourceState = reconsiler.ArgocdResourceState{
			HostedZone: hostedZoneFetcher(),
			Repository: nil,
			UserPoolID: "",
			AuthDomain: "",
		}
	}
}
