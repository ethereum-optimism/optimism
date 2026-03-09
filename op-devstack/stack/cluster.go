package stack

import (
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
)

// Cluster represents a set of chains that interop with each other.
// This may include L1 chains (although potentially not two-way interop due to consensus-layer limitations).
type Cluster interface {
	Common
	ID() ComponentID

	DependencySet() depset.DependencySet
}
