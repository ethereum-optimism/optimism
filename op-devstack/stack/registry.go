package stack

import (
	"sync"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Registrable is the interface that components must implement to be stored in the Registry.
// It provides a way to get the component's ID as a ComponentID.
type Registrable interface {
	// RegistryID returns the ComponentID for this component.
	// This is used as the key in the unified registry.
	RegistryID() ComponentID
}

// Registry is a unified storage for all components in the system.
// It replaces multiple type-specific maps with a single registry that supports:
// - Type-safe access via generic functions
// - Secondary indexes by Kind and ChainID
// - Thread-safe concurrent access
type Registry struct {
	mu sync.RWMutex

	// Primary storage: ComponentID -> component value
	components map[ComponentID]any

	// Secondary index: ComponentKind -> list of ComponentIDs
	byKind map[ComponentKind][]ComponentID

	// Secondary index: ChainID -> list of ComponentIDs
	byChainID map[eth.ChainID][]ComponentID
}

type registryEntry struct {
	id        ComponentID
	component any
}

// NewRegistry creates a new empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		components: make(map[ComponentID]any),
		byKind:     make(map[ComponentKind][]ComponentID),
		byChainID:  make(map[eth.ChainID][]ComponentID),
	}
}

// Register adds a component to the registry.
// If a component with the same ID already exists, it is replaced.
func (r *Registry) Register(id ComponentID, component any) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if this ID already exists (for index cleanup)
	_, exists := r.components[id]
	if exists {
		// Remove from indexes before re-adding
		r.removeFromIndexesLocked(id)
	}

	// Store in primary map
	r.components[id] = component

	// Add to kind index
	r.byKind[id.Kind()] = append(r.byKind[id.Kind()], id)

	// Add to chainID index (if applicable)
	if id.HasChainID() {
		chainID := id.ChainID()
		if chainID != (eth.ChainID{}) {
			r.byChainID[chainID] = append(r.byChainID[chainID], id)
		}
	}
}

// RegisterComponent registers a Registrable component using its RegistryID.
func (r *Registry) RegisterComponent(component Registrable) {
	r.Register(component.RegistryID(), component)
}

// Unregister removes a component from the registry.
func (r *Registry) Unregister(id ComponentID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.components[id]; !exists {
		return
	}

	delete(r.components, id)
	r.removeFromIndexesLocked(id)
}

// removeFromIndexesLocked removes an ID from secondary indexes.
// Caller must hold the write lock.
func (r *Registry) removeFromIndexesLocked(id ComponentID) {
	// Remove from kind index
	kind := id.Kind()
	ids := r.byKind[kind]
	for i, existingID := range ids {
		if existingID == id {
			r.byKind[kind] = append(ids[:i], ids[i+1:]...)
			break
		}
	}

	// Remove from chainID index
	if id.HasChainID() {
		chainID := id.ChainID()
		if chainID != (eth.ChainID{}) {
			ids := r.byChainID[chainID]
			for i, existingID := range ids {
				if existingID == id {
					r.byChainID[chainID] = append(ids[:i], ids[i+1:]...)
					break
				}
			}
		}
	}
}

// Get retrieves a component by its ID.
// Returns nil and false if the component is not found.
func (r *Registry) Get(id ComponentID) (any, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	component, ok := r.components[id]
	return component, ok
}

// Has returns true if a component with the given ID exists.
func (r *Registry) Has(id ComponentID) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.components[id]
	return ok
}

// GetByKind returns all components of a specific kind.
func (r *Registry) GetByKind(kind ComponentKind) []any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := r.byKind[kind]
	result := make([]any, 0, len(ids))
	for _, id := range ids {
		if component, ok := r.components[id]; ok {
			result = append(result, component)
		}
	}
	return result
}

// GetByChainID returns all components associated with a specific chain.
func (r *Registry) GetByChainID(chainID eth.ChainID) []any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := r.byChainID[chainID]
	result := make([]any, 0, len(ids))
	for _, id := range ids {
		if component, ok := r.components[id]; ok {
			result = append(result, component)
		}
	}
	return result
}

// IDsByKind returns all component IDs of a specific kind.
func (r *Registry) IDsByKind(kind ComponentKind) []ComponentID {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := r.byKind[kind]
	result := make([]ComponentID, len(ids))
	copy(result, ids)
	return result
}

// IDsByChainID returns all component IDs associated with a specific chain.
func (r *Registry) IDsByChainID(chainID eth.ChainID) []ComponentID {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := r.byChainID[chainID]
	result := make([]ComponentID, len(ids))
	copy(result, ids)
	return result
}

// AllIDs returns all component IDs in the registry.
func (r *Registry) AllIDs() []ComponentID {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ComponentID, 0, len(r.components))
	for id := range r.components {
		result = append(result, id)
	}
	return result
}

// All returns all components in the registry.
func (r *Registry) All() []any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]any, 0, len(r.components))
	for _, component := range r.components {
		result = append(result, component)
	}
	return result
}

// Len returns the number of components in the registry.
func (r *Registry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.components)
}

// Range calls fn for each component in the registry.
// If fn returns false, iteration stops.
func (r *Registry) Range(fn func(id ComponentID, component any) bool) {
	r.mu.RLock()
	entries := make([]registryEntry, 0, len(r.components))
	for id, component := range r.components {
		entries = append(entries, registryEntry{id: id, component: component})
	}
	r.mu.RUnlock()

	for _, entry := range entries {
		if !fn(entry.id, entry.component) {
			break
		}
	}
}

// RangeByKind calls fn for each component of a specific kind.
// If fn returns false, iteration stops.
func (r *Registry) RangeByKind(kind ComponentKind, fn func(id ComponentID, component any) bool) {
	r.mu.RLock()
	ids := r.byKind[kind]
	entries := make([]registryEntry, 0, len(ids))
	for _, id := range ids {
		if component, ok := r.components[id]; ok {
			entries = append(entries, registryEntry{id: id, component: component})
		}
	}
	r.mu.RUnlock()

	for _, entry := range entries {
		if !fn(entry.id, entry.component) {
			break
		}
	}
}

// RangeByChainID calls fn for each component associated with a specific chain.
// If fn returns false, iteration stops.
func (r *Registry) RangeByChainID(chainID eth.ChainID, fn func(id ComponentID, component any) bool) {
	r.mu.RLock()
	ids := r.byChainID[chainID]
	entries := make([]registryEntry, 0, len(ids))
	for _, id := range ids {
		if component, ok := r.components[id]; ok {
			entries = append(entries, registryEntry{id: id, component: component})
		}
	}
	r.mu.RUnlock()

	for _, entry := range entries {
		if !fn(entry.id, entry.component) {
			break
		}
	}
}

// Clear removes all components from the registry.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.components = make(map[ComponentID]any)
	r.byKind = make(map[ComponentKind][]ComponentID)
	r.byChainID = make(map[eth.ChainID][]ComponentID)
}

// Type-safe generic accessor functions.
// These provide compile-time type safety when working with the registry.

// RegistryGet retrieves a component by its ID and returns it as the expected type.
// Returns the zero value and false if not found or if the type doesn't match.
func RegistryGet[T any](r *Registry, id ComponentID) (T, bool) {
	component, ok := r.Get(id)
	if !ok {
		var zero T
		return zero, false
	}

	typed, ok := component.(T)
	if !ok {
		var zero T
		return zero, false
	}

	return typed, true
}

// RegistryGetByKind retrieves all components of a specific kind and casts them to the expected type.
// Components that don't match the expected type are skipped.
func RegistryGetByKind[T any](r *Registry, kind ComponentKind) []T {
	components := r.GetByKind(kind)
	result := make([]T, 0, len(components))
	for _, component := range components {
		if typed, ok := component.(T); ok {
			result = append(result, typed)
		}
	}
	return result
}

// RegistryGetByChainID retrieves all components for a chain and casts them to the expected type.
// Components that don't match the expected type are skipped.
func RegistryGetByChainID[T any](r *Registry, chainID eth.ChainID) []T {
	components := r.GetByChainID(chainID)
	result := make([]T, 0, len(components))
	for _, component := range components {
		if typed, ok := component.(T); ok {
			result = append(result, typed)
		}
	}
	return result
}

// RegistryRange calls fn for each component of the expected type.
// Components that don't match the expected type are skipped.
func RegistryRange[T any](r *Registry, fn func(id ComponentID, component T) bool) {
	r.Range(func(id ComponentID, component any) bool {
		if typed, ok := component.(T); ok {
			return fn(id, typed)
		}
		return true // skip non-matching types
	})
}

// RegistryRangeByKind calls fn for each component of a specific kind that matches the expected type.
func RegistryRangeByKind[T any](r *Registry, kind ComponentKind, fn func(id ComponentID, component T) bool) {
	r.RangeByKind(kind, func(id ComponentID, component any) bool {
		if typed, ok := component.(T); ok {
			return fn(id, typed)
		}
		return true
	})
}

// RegistryRegister is a type-safe way to register a component with an ID.
func RegistryRegister[T any](r *Registry, id ComponentID, component T) {
	r.Register(id, component)
}
