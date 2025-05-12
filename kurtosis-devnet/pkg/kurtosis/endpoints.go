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

const (
	l1Placeholder = ""
)

// ServiceFinder is the main entry point for finding services and their endpoints
type ServiceFinder struct {
	services        inspect.ServiceMap
	nodeServices    []string
	l2ServicePrefix string

	l2Chains []*spec.ChainSpec
	depsets  map[string]descriptors.DepSet

	triagedServices []*triagedService
}

// ServiceFinderOption configures a ServiceFinder
type ServiceFinderOption func(*ServiceFinder)

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
		services:        services,
		nodeServices:    []string{"cl", "el"},
		l2ServicePrefix: "op-",
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
	svc    *descriptors.Service
	accept chainAcceptor
}

func acceptAll(c *spec.ChainSpec) bool {
	return true
}

func acceptNameOrID(s string) chainAcceptor {
	return func(c *spec.ChainSpec) bool {
		return c.Name == s || c.NetworkID == s
	}
}

func acceptNamesOrIDs(names ...string) chainAcceptor {
	acceptors := make([]chainAcceptor, 0)
	for _, name := range names {
		acceptors = append(acceptors, acceptNameOrID(name))
	}
	return combineAcceptors(acceptors...)
}

func acceptL1() chainAcceptor {
	return acceptNameOrID(l1Placeholder)
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
			return idx, acceptL1(), true
		}

		l2Prefix := f.l2ServicePrefix + prefix
		if strings.HasPrefix(serviceName, l2Prefix) {
			serviceName = strings.TrimPrefix(serviceName, l2Prefix)
			// first we have the chain ID
			parts := strings.Split(serviceName, "-")
			chainID := parts[0]
			idx := extractIndex(parts[1])
			return idx, acceptNameOrID(chainID), true
		}

		return 0, nil, false
	}
}

func (f *ServiceFinder) triageExclusiveL2Service(prefix string) serviceParser {
	idx := -1
	return func(serviceName string) (int, chainAcceptor, bool) {
		if strings.HasPrefix(serviceName, prefix) {
			suffix := strings.TrimPrefix(serviceName, prefix)
			return idx, acceptNameOrID(suffix), true
		}
		return idx, nil, false
	}
}

func (f *ServiceFinder) triageMultiL2Service(prefix string) serviceParser {
	idx := -1
	return func(serviceName string) (int, chainAcceptor, bool) {
		if strings.HasPrefix(serviceName, prefix) {
			suffix := strings.TrimPrefix(serviceName, prefix)
			parts := strings.Split(suffix, "-")
			// parts[0] is the service ID
			return idx, acceptNamesOrIDs(parts[1:]...), true
		}
		return idx, nil, false
	}
}

func (f *ServiceFinder) triageSuperchainService(prefix string) serviceParser {
	idx := -1
	return func(serviceName string) (int, chainAcceptor, bool) {
		if strings.HasPrefix(serviceName, prefix) {
			suffix := strings.TrimPrefix(serviceName, prefix)
			parts := strings.Split(suffix, "-")
			// parts[0] is the service ID
			ds, ok := f.depsets[parts[1]]
			if !ok {
				return idx, nil, false
			}
			var depSet depset.StaticConfigDependencySet
			if err := json.Unmarshal(ds, &depSet); err != nil {
				return idx, nil, false
			}

			chains := make([]string, 0)
			for _, chain := range depSet.Chains() {
				chains = append(chains, chain.String())
			}
			return idx, acceptNamesOrIDs(chains...), true
		}
		return idx, nil, false
	}
}

func (f *ServiceFinder) triageUniversalL2Service(name string) serviceParser {
	idx := -1
	return func(serviceName string) (int, chainAcceptor, bool) {
		if serviceName == name {
			return idx, acceptAll, true
		}
		return idx, nil, false
	}
}

func (f *ServiceFinder) triage() {
	rules := map[string]serviceParser{
		"el":         f.triageNode("el-"),
		"cl":         f.triageNode("cl-"),
		"batcher":    f.triageExclusiveL2Service("op-batcher-"),
		"proposer":   f.triageExclusiveL2Service("op-proposer-"),
		"proxyd":     f.triageExclusiveL2Service("proxyd-"),
		"supervisor": f.triageSuperchainService("op-supervisor-"),
		"challenger": f.triageMultiL2Service("op-challenger-"),
		"faucet":     f.triageUniversalL2Service("op-faucet"),
	}

	triagedServices := []*triagedService{}
	for serviceName, ports := range f.services {
		endpoints := make(descriptors.EndpointMap)
		for portName, portInfo := range ports {
			endpoints[portName] = portInfo
		}
		svc := &descriptors.Service{
			Name:      serviceName,
			Endpoints: endpoints,
		}
		for tag, rule := range rules {
			if idx, accept, ok := rule(serviceName); ok {
				triagedServices = append(triagedServices, &triagedService{
					tag:    tag,
					idx:    idx,
					accept: accept,
					svc:    svc,
				})
			}
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
			nodes[svc.idx] = node
		} else {
			services[svc.tag] = append(services[svc.tag], svc.svc)
		}
	}

	return nodes, services
}

// FindL1Services finds L1 nodes.
func (f *ServiceFinder) FindL1Services() ([]descriptors.Node, descriptors.RedundantServiceMap) {
	chain := &spec.ChainSpec{
		Name:      l1Placeholder,
		NetworkID: l1Placeholder,
	}
	return f.findChainServices(chain)
}

// FindL2Services finds L2 nodes and services for a specific network
func (f *ServiceFinder) FindL2Services(s *spec.ChainSpec) ([]descriptors.Node, descriptors.RedundantServiceMap) {
	return f.findChainServices(s)
}
