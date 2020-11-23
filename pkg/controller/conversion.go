package controller

import (
	"fmt"
	"github.com/oslokommune/okctl/pkg/api"
	"github.com/oslokommune/okctl/pkg/apis/okctl.io/v1alpha1"
	"github.com/oslokommune/okctl/pkg/client/core/store/filesystem"
	"github.com/oslokommune/okctl/pkg/client/store"
	"github.com/oslokommune/okctl/pkg/config"
	"github.com/spf13/afero"
	"path"
	"strings"
)

type existingServices struct {
	hasPrimaryHostedZone bool
	hasVPC bool
	hasCluster bool
	hasExternalSecrets bool
	hasALBIngressController bool
	hasExternalDNS bool
}

func NewCreateCurrentStateGraphOpts(fs *afero.Afero, outputDir string) (*existingServices, error) {
	var err error
	
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
	vpcNode = createNode(root, SynchronizationNodeTypeVPC, opts.hasVPC)
	clusterNode = createNode(vpcNode, SynchronizationNodeTypeCluster, opts.hasCluster)
	createNode(clusterNode, SynchronizationNodeTypeExternalSecrets, opts.hasExternalSecrets)
	createNode(clusterNode, SynchronizationNodeTypeALBIngress, opts.hasALBIngressController)
	createNode(clusterNode, SynchronizationNodeTypeExternalDNS, opts.hasExternalDNS)

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

	vpcNode = createNode(root, SynchronizationNodeTypeVPC, true)
	clusterNode = createNode(vpcNode, SynchronizationNodeTypeCluster, true)

	createNode(clusterNode, SynchronizationNodeTypeExternalSecrets, cluster.Integrations.ExternalSecrets)
	createNode(clusterNode, SynchronizationNodeTypeALBIngress, cluster.Integrations.ALBIngressController)
	createNode(clusterNode, SynchronizationNodeTypeExternalDNS, cluster.Integrations.ExternalDNS)

	return root
}

func ApplyDesiredStateMetadata(graph *SynchronizationNode, cluster *v1alpha1.Cluster) {
	// TODO: Do something about primary hosted zone
	vpcNode := graph.GetNode(&SynchronizationNode{ Type: SynchronizationNodeTypeVPC })
	vpcNode.Metadata = VPCMetadata{
		Cidr:             cluster.VPC.CIDR,
		HighAvailability: cluster.VPC.HighAvailability,
	}

	// TODO: Do something about cluster? maybe not
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

func getHostedZoneMetadata(fs *afero.Afero, cluster v1alpha1.Cluster) (*filesystem.HostedZone, error) {
	hostedZone := filesystem.HostedZone{}
	
	domain := fmt.Sprintf("%s.oslo.systems", strings.Join([]string{cluster.Metadata.Name, cluster.Metadata.Environment}, "-"))
	baseDir := path.Join(cluster.Github.OutputPath, cluster.Metadata.Environment, domain, config.DefaultDomainBaseDir)
	
	_, err := store.NewFileSystem(baseDir, fs).
		GetStruct(config.DefaultDomainOutputsFile, hostedZone, store.FromJSON()).
		Do()
	if err != nil {
		return nil, fmt.Errorf("error reading from hosted zone state file: %w", err)
	}
	
	return &hostedZone, nil
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
