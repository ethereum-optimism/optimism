package stack

import (
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
)

// ClusterID identifies a Cluster by name, is type-safe, and can be value-copied and used as map key.
type ClusterID genericID

const ClusterKind Kind = "Cluster"

func (id ClusterID) String() string {
	return genericID(id).string(ClusterKind)
}

func (id ClusterID) MarshalText() ([]byte, error) {
	return genericID(id).marshalText(ClusterKind)
}

func (id *ClusterID) UnmarshalText(data []byte) error {
	return (*genericID)(id).unmarshalText(ClusterKind, data)
}

func SortClusterIDs(ids []ClusterID) []ClusterID {
	return copyAndSortCmp(ids)
}

// Cluster represents a set of chains that interop with each other.
// This may include L1 chains (although potentially not two-way interop due to consensus-layer limitations).
type Cluster interface {
	Common
	ID() ClusterID

	DependencySet() depset.DependencySet
}
