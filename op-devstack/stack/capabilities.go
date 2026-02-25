package stack

import (
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Capability interfaces define shared behaviors across component types.
// These enable polymorphic operations without requiring components to
// implement interfaces with incompatible ID() method signatures.
//
// For example, RollupBoostNode and OPRBuilderNode both provide L2 EL
// functionality but can't implement L2ELNode because their ID() methods
// return different types. The L2ELCapable interface captures the shared
// L2 EL behavior, allowing code to work with any L2 EL-like component.

// L2ELCapable is implemented by any component that provides L2 execution layer functionality.
// This includes L2ELNode, RollupBoostNode, and OPRBuilderNode.
//
// Components implementing this interface can:
// - Execute L2 transactions
// - Provide engine API access for consensus layer integration
type L2ELCapable interface {
	L2EthClient() apis.L2EthClient
	L2EngineClient() apis.EngineClient
	ELNode
}

// L2ELCapableKinds returns all ComponentKinds that implement L2ELCapable.
func L2ELCapableKinds() []ComponentKind {
	return []ComponentKind{
		KindL2ELNode,
		KindRollupBoostNode,
		KindOPRBuilderNode,
	}
}

// L1ELCapable is implemented by any component that provides L1 execution layer functionality.
type L1ELCapable interface {
	ELNode
}

// L1ELCapableKinds returns all ComponentKinds that implement L1ELCapable.
func L1ELCapableKinds() []ComponentKind {
	return []ComponentKind{
		KindL1ELNode,
	}
}

// Verify that expected types implement capability interfaces.
// These are compile-time checks.
var (
	_ L2ELCapable = (L2ELNode)(nil)
	_ L2ELCapable = (RollupBoostNode)(nil)
	_ L2ELCapable = (OPRBuilderNode)(nil)
)

// Registry helper functions for capability-based lookups.

// RegistryFindByCapability returns all components that implement the given capability interface.
// This iterates over all components and performs a type assertion.
func RegistryFindByCapability[T any](r *Registry) []T {
	var result []T
	r.Range(func(id ComponentID, component any) bool {
		if capable, ok := component.(T); ok {
			result = append(result, capable)
		}
		return true
	})
	return result
}

// RegistryFindByCapabilityOnChain returns all components on a specific chain
// that implement the given capability interface.
func RegistryFindByCapabilityOnChain[T any](r *Registry, chainID eth.ChainID) []T {
	var result []T
	r.RangeByChainID(chainID, func(id ComponentID, component any) bool {
		if capable, ok := component.(T); ok {
			result = append(result, capable)
		}
		return true
	})
	return result
}

// RegistryFindByKinds returns all components of the specified kinds.
// This is useful when you know which kinds implement a capability.
func RegistryFindByKinds(r *Registry, kinds []ComponentKind) []any {
	var result []any
	for _, kind := range kinds {
		result = append(result, r.GetByKind(kind)...)
	}
	return result
}

// RegistryFindByKindsTyped returns all components of the specified kinds,
// cast to the expected type. Components that don't match are skipped.
func RegistryFindByKindsTyped[T any](r *Registry, kinds []ComponentKind) []T {
	var result []T
	for _, kind := range kinds {
		for _, component := range r.GetByKind(kind) {
			if typed, ok := component.(T); ok {
				result = append(result, typed)
			}
		}
	}
	return result
}

// FindL2ELCapable returns all L2 EL-capable components in the registry.
// This is a convenience function that finds L2ELNode, RollupBoostNode, and OPRBuilderNode.
func FindL2ELCapable(r *Registry) []L2ELCapable {
	return RegistryFindByKindsTyped[L2ELCapable](r, L2ELCapableKinds())
}

// FindL2ELCapableOnChain returns all L2 EL-capable components on a specific chain.
func FindL2ELCapableOnChain(r *Registry, chainID eth.ChainID) []L2ELCapable {
	return RegistryFindByCapabilityOnChain[L2ELCapable](r, chainID)
}

// FindL2ELCapableByKey returns the first L2 EL-capable component with the given key and chainID.
// This enables the polymorphic lookup pattern where you want to find a node by key
// regardless of whether it's an L2ELNode, RollupBoostNode, or OPRBuilderNode.
func FindL2ELCapableByKey(r *Registry, key string, chainID eth.ChainID) (L2ELCapable, bool) {
	for _, kind := range L2ELCapableKinds() {
		id := NewComponentID(kind, key, chainID)
		if component, ok := r.Get(id); ok {
			if capable, ok := component.(L2ELCapable); ok {
				return capable, true
			}
		}
	}
	return nil, false
}
