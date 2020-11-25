package controller

import (
	"fmt"
	"github.com/mishudark/errors"
	"github.com/oslokommune/okctl/pkg/api"
	"github.com/oslokommune/okctl/pkg/apis/okctl.io/v1alpha1"
	"github.com/oslokommune/okctl/pkg/client/store"
	"github.com/oslokommune/okctl/pkg/config"
	"github.com/oslokommune/okctl/pkg/config/state"
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
func CreateCurrentStateGraph(opts *existingServices) (root *SynchronizationNode) {
	root = createNode(nil, SynchronizationNodeTypeNoop, true)
	
	var (
		vpcNode,
		clusterNode *SynchronizationNode
	)
	
	createNode(root, SynchronizationNodeTypeZone, opts.hasPrimaryHostedZone)
	createNode(root, SynchronizationNodeTypeGithub, false)
	vpcNode = createNode(root, SynchronizationNodeTypeVPC, opts.hasVPC)

	clusterNode = createNode(vpcNode, SynchronizationNodeTypeCluster, opts.hasCluster)

	createNode(clusterNode, SynchronizationNodeTypeExternalSecrets, opts.hasExternalSecrets)
	createNode(clusterNode, SynchronizationNodeTypeALBIngress, opts.hasALBIngressController)
	createNode(clusterNode, SynchronizationNodeTypeExternalDNS, opts.hasExternalDNS)
	//createNode(clusterNode, SynchronizationNodeTypeIdentityManager, opts.hasIdentityManager)

	return root
}

// CreateDesiredStateGraph knows how to create a graph based on a cluster declaration
func CreateDesiredStateGraph(cluster *v1alpha1.Cluster) (root *SynchronizationNode) {
	root = createNode(nil, SynchronizationNodeTypeNoop, true)

	var (
		vpcNode,
		clusterNode *SynchronizationNode
	)
	
	if len(cluster.DNSZones) > 0 {
		for range cluster.DNSZones { // TODO: not gonna work. More than one will generate multiple primaries
			createNode(root, SynchronizationNodeTypeZone, true)
		}
	}

	createNode(root, SynchronizationNodeTypeGithub, true)
	vpcNode = createNode(root, SynchronizationNodeTypeVPC, true)

	clusterNode = createNode(vpcNode, SynchronizationNodeTypeCluster, true)

	createNode(clusterNode, SynchronizationNodeTypeExternalSecrets, cluster.Integrations.ExternalSecrets)
	createNode(clusterNode, SynchronizationNodeTypeALBIngress, cluster.Integrations.ALBIngressController)
	createNode(clusterNode, SynchronizationNodeTypeExternalDNS, cluster.Integrations.ExternalDNS) // TODO: Needs to be dependend on primary hosted zone
	//createNode(clusterNode, SynchronizationNodeTypeIdentityManager, cluster.Integrations.Cognito) // TODO: ArgoCD is dependent on cognito, but i dont think cognito is dependent on anything other than hosted zone

	return root
}

func ApplyDesiredStateMetadata(graph *SynchronizationNode, cluster *v1alpha1.Cluster, repoDir string) error {
	// TODO: Fetch cluster first and fetch hosted zone from cluster to ensure primary hosted zone is fetched after
	// moving primary hosted zone
	primaryHostedZoneNode := graph.GetNode(&SynchronizationNode{ Type: SynchronizationNodeTypeZone })
	if primaryHostedZoneNode == nil {
		return errors.New("expected primary hosted zone node was not found")
	}

	primaryHostedZoneNode.Metadata = HostedZoneMetadata{Domain: cluster.DNSZones[0].ParentDomain}

	vpcNode := graph.GetNode(&SynchronizationNode{ Type: SynchronizationNodeTypeVPC })
	if vpcNode == nil {
		return errors.New("expected vpc node was not found")
	}
	
	vpcNode.Metadata = VPCMetadata{
		Cidr:             cluster.VPC.CIDR,
		HighAvailability: cluster.VPC.HighAvailability,
	}
	
	githubNode := graph.GetNode(&SynchronizationNode{ Type: SynchronizationNodeTypeGithub })
	if githubNode != nil { // TODO: A Github is required, so this should probably break something if not existing
		repo, err := git.GithubRepoFullName(cluster.Github.Organisation, repoDir)
		if err != nil {
		    return fmt.Errorf("error fetching full git repo name: %w", err)
		}

		githubNode.Metadata = GithubMetadata{
			Organization: cluster.Github.Organisation,
			Repository:   repo,
		}
	}
	
	argocdNode := graph.GetNode(&SynchronizationNode{ Type: SynchronizationNodeTypeArgoCD })
	if argocdNode != nil {
		argocdNode.Metadata = ArgocdMetadata{Organization: cluster.Github.Organisation }
	}

	// TODO: Do something about cluster? maybe not
	return nil
}

func createNode(parent *SynchronizationNode, nodeType SynchronizationNodeType, present bool) (child *SynchronizationNode) {
	child = &SynchronizationNode{
		Type:           nodeType,
		Children:       make([]*SynchronizationNode, 0),
	}
	
	if present {
		child.State = SynchronizationNodeStatePresent
	} else {
		child.State = SynchronizationNodeStateAbsent
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
func CreateClusterStateRefresher(fs *afero.Afero, outputDir string, cidrFn StringFetcher) StateRefreshFn {
	return func(node *SynchronizationNode) {
		vpc := getVpcState(fs, outputDir)
		
		vpc.Cidr = cidrFn()

		node.ResourceState = ClusterResourceState{VPC: vpc}
	}
}

func CreateALBIngressControllerRefresher(fs *afero.Afero, outputDir string) StateRefreshFn {
	return func(node *SynchronizationNode) {
		vpc := getVpcState(fs, outputDir)
		
		node.ResourceState = AlbIngressControllerResourceState{VpcID: vpc.VpcID}
	}
}

func CreateExternalDNSStateRefresher(domainFetcher StringFetcher, hostedZoneIDFetcher StringFetcher) StateRefreshFn {
	return func(node *SynchronizationNode) {
		node.ResourceState = ExternalDNSResourceState{
			HostedZoneID: hostedZoneIDFetcher(),
			Domain:       domainFetcher(),
		}
	}
}

func CreateIdentityManagerRefresher(domainFetcher StringFetcher, hostedZoneIDFetcher StringFetcher) StateRefreshFn {
	return func(node *SynchronizationNode) {
		node.ResourceState = IdentityManagerResourceState{
			HostedZoneID: hostedZoneIDFetcher(),
			Domain:       domainFetcher(),
		}
	}
}

func CreateGithubStateRefresher(ghGetter GithubGetter, ghSetter GithubSetter) StateRefreshFn {
	return func(node *SynchronizationNode) {
		node.ResourceState = GithubResourceState{
			Getter: ghGetter,
			Saver: ghSetter,
		}
	}
}

type HostedZoneFetcher func() *state.HostedZone
func CreateArgocdStateRefresher(hostedZoneFetcher HostedZoneFetcher) StateRefreshFn {
	return func(node *SynchronizationNode) {
		node.ResourceState = ArgocdResourceState{
			HostedZone: hostedZoneFetcher(),
			Repository: nil,
			UserPoolID: "",
			AuthDomain: "",
		}
	}
}
