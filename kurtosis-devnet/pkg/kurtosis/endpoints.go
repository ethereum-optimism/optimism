package kurtosis

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/sources/inspect"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/sources/spec"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
)

// ServiceFinder is the main entry point for finding services and their endpoints
type ServiceFinder struct {
	services inspect.ServiceMap

	l1Chain  *spec.ChainSpec
	l2Chains []*spec.ChainSpec
	depsets  map[string]descriptors.DepSet

	triagedServices []*triagedService
}

// ServiceFinderOption configures a ServiceFinder
type ServiceFinderOption func(*ServiceFinder)

// WithL1Chain sets the L1 chain
func WithL1Chain(chain *spec.ChainSpec) ServiceFinderOption {
	return func(f *ServiceFinder) {
		f.l1Chain = chain
	}
}

// WithL2Chains sets the L2 networks
func WithL2Chains(networks []*spec.ChainSpec) ServiceFinderOption {
	return func(f *ServiceFinder) {
		f.l2Chains = networks
	}
}

// WithDepSets sets the dependency sets
func WithDepSets(depsets map[string]descriptors.DepSet) ServiceFinderOption {
	return func(f *ServiceFinder) {
		f.depsets = depsets
	}
}

// NewServiceFinder creates a new ServiceFinder with the given options
func NewServiceFinder(services inspect.ServiceMap, opts ...ServiceFinderOption) *ServiceFinder {
	f := &ServiceFinder{
		services: services,
	}
	for _, opt := range opts {
		opt(f)
	}

	f.triage()
	return f
}

type chainAcceptor func(*spec.ChainSpec) bool

type serviceParser func(string) (int, chainAcceptor, bool)

type triagedService struct {
	tag    string // service tag
	idx    int    // service index (for nodes)
	name   string // service name (for nodes)
	svc    *descriptors.Service
	accept chainAcceptor
}

func acceptAll(c *spec.ChainSpec) bool {
	return true
}

func acceptID(s string) chainAcceptor {
	return func(c *spec.ChainSpec) bool {
		return c.NetworkID == s
	}
}

func acceptIDs(ids ...string) chainAcceptor {
	acceptors := make([]chainAcceptor, 0)
	for _, id := range ids {
		acceptors = append(acceptors, acceptID(id))
	}
	return combineAcceptors(acceptors...)
}

func combineAcceptors(acceptors ...chainAcceptor) chainAcceptor {
	return func(c *spec.ChainSpec) bool {
		for _, acceptor := range acceptors {
			if acceptor(c) {
				return true
			}
		}
		return false
	}
}

// This is now for L1 only. L2 is handled through labels.
func (f *ServiceFinder) triageNode(prefix string) serviceParser {
	return func(serviceName string) (int, chainAcceptor, bool) {
		extractIndex := func(s string) int {
			// Extract numeric index from service name
			parts := strings.Split(s, "-")
			if idx, err := strconv.ParseUint(parts[0], 10, 32); err == nil {
				return int(idx) - 1
			}
			return 0
		}

		if strings.HasPrefix(serviceName, prefix) { // L1
			idx := extractIndex(strings.TrimPrefix(serviceName, prefix))
			return idx, acceptID(f.l1Chain.NetworkID), true
		}

		return 0, nil, false
	}
}

type serviceParserRules map[string]serviceParser

func (spr serviceParserRules) apply(serviceName string, endpoints descriptors.EndpointMap) *triagedService {
	for tag, rule := range spr {
		if idx, accept, ok := rule(serviceName); ok {
			return &triagedService{
				tag:    tag,
				idx:    idx,
				accept: accept,
				svc: &descriptors.Service{
					Name:      serviceName,
					Endpoints: endpoints,
				},
			}
		}
	}
	return nil
}

// TODO: this might need some adjustments as we stabilize labels in optimism-package
const (
	kindLabel                 = "op.kind"
	networkIDLabel            = "op.network.id"
	nodeNameLabel             = "op.network.participant.name"
	nodeIndexLabel            = "op.network.participant.index"
	supervisorSuperchainLabel = "op.network.supervisor.superchain"
)

func (f *ServiceFinder) getNetworkIDs(svc *inspect.Service) []string {
	var network_ids []string
	id, ok := svc.Labels[networkIDLabel]
	if !ok {
		// network IDs might be specified through a superchain
		superchain, ok := svc.Labels[supervisorSuperchainLabel]
		if !ok {
			return nil
		}
		ds, ok := f.depsets[superchain]
		if !ok {
			return nil
		}
		var depSet depset.StaticConfigDependencySet
		err := json.Unmarshal(ds, &depSet)
		if err != nil {
			return nil
		}
		for _, chain := range depSet.Chains() {
			network_ids = append(network_ids, chain.String())
		}
	} else {
		network_ids = strings.Split(id, "-")
	}

	return network_ids
}

func (f *ServiceFinder) triageByLabels(svc *inspect.Service, name string, endpoints descriptors.EndpointMap) *triagedService {
	tag, ok := svc.Labels[kindLabel]
	if !ok {
		return nil
	}
	network_ids := f.getNetworkIDs(svc)
	idx := -1
	if val, ok := svc.Labels[nodeIndexLabel]; ok {
		i, err := strconv.Atoi(val)
		if err != nil {
			return nil
		}
		idx = i
	}

	accept := acceptIDs(network_ids...)
	if len(network_ids) == 0 { // TODO: this is only for faucet right now, we can remove this once we have a proper label for all services
		accept = acceptAll
	}
	return &triagedService{
		tag:    tag,
		idx:    idx,
		name:   svc.Labels[nodeNameLabel],
		accept: accept,
		svc: &descriptors.Service{
			Name:      name,
			Endpoints: endpoints,
		},
	}
}

func (f *ServiceFinder) triage() {
	rules := serviceParserRules{
		"el": f.triageNode("el-"),
		"cl": f.triageNode("cl-"),
	}

	triagedServices := []*triagedService{}
	for serviceName, svc := range f.services {
		endpoints := make(descriptors.EndpointMap)
		for portName, portInfo := range svc.Ports {
			endpoints[portName] = portInfo
		}

		// Ultimately we'll rely only on labels, and most of the code in this file will disappear as a result.
		//
		// For now though the L1 services are still not tagged properly so we rely on the name resolution as a fallback
		triaged := f.triageByLabels(svc, serviceName, endpoints)
		if triaged == nil {
			triaged = rules.apply(serviceName, endpoints)
		}

		if triaged != nil {
			triagedServices = append(triagedServices, triaged)
		}
	}

	f.triagedServices = triagedServices
}

func (f *ServiceFinder) findChainServices(chain *spec.ChainSpec) ([]descriptors.Node, descriptors.RedundantServiceMap) {
	var nodes []descriptors.Node
	services := make(descriptors.RedundantServiceMap)

	var selected []*triagedService
	for _, svc := range f.triagedServices {
		if svc.accept(chain) {
			if svc.idx >= len(nodes) {
				// just resize the slice, that'll create "0" items for the new indices.
				// We don't expect more than a few nodes per chain, so this is fine.
				nodes = make([]descriptors.Node, svc.idx+1)
			}
			if svc.idx < 0 { // not a node service
				// create a dummy entry for the service
				services[svc.tag] = nil
			}
			selected = append(selected, svc)
		}
	}

	// Now our slice is the right size, and our map has the right keys, we can just fill in the data
	for _, svc := range selected {
		if svc.idx >= 0 {
			node := nodes[svc.idx]
			if node.Services == nil {
				node.Services = make(descriptors.ServiceMap)
			}
			node.Services[svc.tag] = svc.svc
			node.Name = svc.name
			nodes[svc.idx] = node
		} else {
			services[svc.tag] = append(services[svc.tag], svc.svc)
		}
	}

	return nodes, services
}

// FindL1Services finds L1 nodes.
func (f *ServiceFinder) FindL1Services() ([]descriptors.Node, descriptors.RedundantServiceMap) {
	return f.findChainServices(f.l1Chain)
}

// FindL2Services finds L2 nodes and services for a specific network
func (f *ServiceFinder) FindL2Services(s *spec.ChainSpec) ([]descriptors.Node, descriptors.RedundantServiceMap) {
	return f.findChainServices(s)
}
