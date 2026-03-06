package stack

// ComponentRegistry provides generic component access for systems and networks.
// This interface enables unified component lookup regardless of component type,
// reducing the need for type-specific getter methods on container interfaces.
//
// Components are stored by ComponentID and can be queried by:
// - Exact ID match (Component)
// - Kind (Components, ComponentIDs)
//
// Implementations should use the Registry type internally for storage.
type ComponentRegistry interface {
	// Component returns a component by its ID.
	// Returns (component, true) if found, (nil, false) otherwise.
	Component(id ComponentID) (any, bool)

	// Components returns all components of a given kind.
	// Returns an empty slice if no components of that kind exist.
	Components(kind ComponentKind) []any

	// ComponentIDs returns all component IDs of a given kind.
	// Returns an empty slice if no components of that kind exist.
	ComponentIDs(kind ComponentKind) []ComponentID
}
