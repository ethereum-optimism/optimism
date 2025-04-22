package kurtosis

import (
	"strconv"
	"strings"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/sources/inspect"
)

// ServiceFinder is the main entry point for finding services and their endpoints
type ServiceFinder struct {
	services        inspect.ServiceMap
	nodeServices    []string
	l2ServicePrefix string
	l2Networks      []string
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
func WithL2Networks(networks []string) ServiceFinderOption {
	return func(f *ServiceFinder) {
		f.l2Networks = networks
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
	return f
}

// FindL1Services finds L1 nodes.
func (f *ServiceFinder) FindL1Services() ([]descriptors.Node, descriptors.ServiceMap) {
	return f.findRPCEndpoints(func(serviceName string) (string, int, bool) {
		// Only match services that start with one of the node service identifiers.
		// We might have to change this if we need to support L1 services beyond nodes.
		for _, service := range f.nodeServices {
			if strings.HasPrefix(serviceName, service) {
				tag, idx := f.serviceTag(serviceName)
				return tag, idx, true
			}
		}
		return "", 0, false
	})
}

// FindL2Services finds L2 nodes and services for a specific network
func (f *ServiceFinder) FindL2Services(network string) ([]descriptors.Node, descriptors.ServiceMap) {
	networkSuffix := "-" + network
	return f.findRPCEndpoints(func(serviceName string) (string, int, bool) {
		if strings.HasSuffix(serviceName, networkSuffix) {
			name := strings.TrimSuffix(serviceName, networkSuffix)
			tag, idx := f.serviceTag(strings.TrimPrefix(name, f.l2ServicePrefix))
			return tag, idx, true
		}

		// skip over the other L2 services
		for _, l2Network := range f.l2Networks {
			if strings.HasSuffix(serviceName, "-"+l2Network) {
				return "", 0, false
			}
		}

		// Some services don't have a network suffix, as they span multiple chains
		// TODO(14849): ideally we'd need to handle *partial* chain coverage.
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
