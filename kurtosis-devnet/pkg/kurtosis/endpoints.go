package kurtosis

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/sources/inspect"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/sources/spec"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
)

type ChainSpec struct {
	spec.ChainSpec
	DepSets map[string]descriptors.DepSet
}

// ServiceFinder is the main entry point for finding services and their endpoints
type ServiceFinder struct {
	services        inspect.ServiceMap
	nodeServices    []string
	l2ServicePrefix string
	l2Networks      []ChainSpec
	globalServices  []string
}

// ServiceFinderOption configures a ServiceFinder
type ServiceFinderOption func(*ServiceFinder)

// WithNodeServices sets the node service identifiers
func WithNodeServices(services []string) ServiceFinderOption {
	return func(f *ServiceFinder) {
		f.nodeServices = services
	}
}

// WithL2ServicePrefix sets the prefix used to identify L2 services
func WithL2ServicePrefix(prefix string) ServiceFinderOption {
	return func(f *ServiceFinder) {
		f.l2ServicePrefix = prefix
	}
}

// WithL2Networks sets the L2 networks
func WithL2Networks(networks []ChainSpec) ServiceFinderOption {
	return func(f *ServiceFinder) {
		f.l2Networks = networks
	}
}

// WithGlobalServices sets the global services
func WithGlobalServices(services []string) ServiceFinderOption {
	return func(f *ServiceFinder) {
		f.globalServices = services
	}
}

// NewServiceFinder creates a new ServiceFinder with the given options
func NewServiceFinder(services inspect.ServiceMap, opts ...ServiceFinderOption) *ServiceFinder {
	f := &ServiceFinder{
		services:        services,
		nodeServices:    []string{"cl", "el"},
		l2ServicePrefix: "op-",
		globalServices:  []string{"op-faucet"},
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// FindL1Services finds L1 nodes.
func (f *ServiceFinder) FindL1Services() ([]descriptors.Node, descriptors.ServiceMap) {
	return f.findRPCEndpoints(func(serviceName string) (string, int, bool) {
		// Find node services and global services
		allServices := append(f.nodeServices, f.globalServices...)
		for _, service := range allServices {
			if strings.HasPrefix(serviceName, service) {
				// strip the L2 prefix if it's there.
				name := strings.TrimPrefix(serviceName, f.l2ServicePrefix)
				tag, idx := f.serviceTag(name)
				return tag, idx, true
			}
		}
		return "", 0, false
	})
}

// FindL2Services finds L2 nodes and services for a specific network
func (f *ServiceFinder) FindL2Services(s ChainSpec) ([]descriptors.Node, descriptors.ServiceMap) {
	network := s.Name
	networkID := s.NetworkID
	return f.findRPCEndpoints(func(serviceName string) (string, int, bool) {
		possibleSuffixes := []string{"-" + network, "-" + networkID}
		for _, suffix := range possibleSuffixes {
			if strings.HasSuffix(serviceName, suffix) {
				name := strings.TrimSuffix(serviceName, suffix)
				tag, idx := f.serviceTag(strings.TrimPrefix(name, f.l2ServicePrefix))
				return tag, idx, true
			}
		}

		// skip over the other L2 services
		for _, l2Network := range f.l2Networks {
			if strings.HasSuffix(serviceName, "-"+l2Network.Name) || strings.HasSuffix(serviceName, "-"+l2Network.NetworkID) {
				return "", 0, false
			}
		}

		// supervisor is special: itcovers multiple networks, so we need to
		// identify the depset this chain belongs to
		if strings.HasPrefix(serviceName, "op-supervisor") {
			for dsName, ds := range s.DepSets {
				suffix := "-" + dsName
				if !strings.HasSuffix(serviceName, suffix) {
					// not the right depset for this supervisor, skip it
					continue
				}
				var depSet depset.StaticConfigDependencySet
				if err := json.Unmarshal(ds, &depSet); err != nil {
					return "", 0, false
				}
				var chainID eth.ChainID
				if err := chainID.UnmarshalText([]byte(s.NetworkID)); err != nil {
					return "", 0, false
				}
				if depSet.HasChain(chainID) {
					name := strings.TrimSuffix(serviceName, suffix)
					tag, idx := f.serviceTag(strings.TrimPrefix(name, f.l2ServicePrefix))
					return tag, idx, true
				}
			}
			// this supervisor is irrelevant to this chain, skip it
			return "", 0, false
		}

		// Some services don't have a network suffix, as they span multiple chains
		if strings.HasPrefix(serviceName, f.l2ServicePrefix) {
			tag, idx := f.serviceTag(strings.TrimPrefix(serviceName, f.l2ServicePrefix))
			return tag, idx, true
		}
		return "", 0, false
	})
}

// findRPCEndpoints looks for services matching the given predicate that have an RPC port
func (f *ServiceFinder) findRPCEndpoints(matchService func(string) (string, int, bool)) ([]descriptors.Node, descriptors.ServiceMap) {
	serviceMap := make(descriptors.ServiceMap)
	var nodes []descriptors.Node

	for serviceName, ports := range f.services {
		if serviceIdentifier, num, ok := matchService(serviceName); ok {
			var allocated bool
			for _, service := range f.nodeServices {
				if serviceIdentifier == service {
					if num > len(nodes) {
						// Extend the slice to accommodate the required index
						for i := len(nodes); i < num; i++ {
							nodes = append(nodes, descriptors.Node{
								Services: make(descriptors.ServiceMap),
							})
						}
					}
					endpoints := make(descriptors.EndpointMap)
					for portName, portInfo := range ports {
						endpoints[portName] = portInfo
					}
					nodes[num-1].Services[serviceIdentifier] = &descriptors.Service{
						Name:      serviceName,
						Endpoints: endpoints,
					}
					allocated = true
				}
			}
			if !allocated {
				endpoints := make(descriptors.EndpointMap)
				for portName, portInfo := range ports {
					endpoints[portName] = portInfo
				}
				serviceMap[serviceIdentifier] = &descriptors.Service{
					Name:      serviceName,
					Endpoints: endpoints,
				}
			}
		}
	}
	return nodes, serviceMap
}

// serviceTag returns the shorthand service tag and index if it's a service with multiple instances
func (f *ServiceFinder) serviceTag(serviceName string) (string, int) {
	// Find last occurrence of a number sequence
	lastStart := -1
	lastEnd := -1

	// Scan through the string to find number sequences
	for i := 0; i < len(serviceName); i++ {
		if serviceName[i] >= '0' && serviceName[i] <= '9' {
			start := i
			// Find end of this number sequence
			for i < len(serviceName) && serviceName[i] >= '0' && serviceName[i] <= '9' {
				i++
			}
			lastStart = start
			lastEnd = i
		}
	}

	if lastStart == -1 {
		return serviceName, 0
	}

	idx, err := strconv.Atoi(serviceName[lastStart:lastEnd])
	if err != nil {
		return serviceName, 0
	}

	// If there are multiple numbers, return just the base name
	// Find the first number sequence
	firstStart := strings.IndexFunc(serviceName, func(r rune) bool {
		return r >= '0' && r <= '9'
	})
	if firstStart != lastStart {
		// Multiple numbers found, return just the base name
		tag := serviceName[:firstStart]
		tag = strings.TrimRight(tag, "-")
		return tag, idx
	}

	// Single number case
	tag := serviceName[:lastStart]
	tag = strings.TrimRight(tag, "-")
	return tag, idx
}
