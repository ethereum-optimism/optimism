package system2

import (
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
)

// L2ClusterID identifies a Cluster by name, is type-safe, and can be value-copied and used as map key.
type L2ClusterID genericID

func (id L2ClusterID) String() string {
	return genericID(id).string("L2Cluster")
}

func (id L2ClusterID) MarshalText() ([]byte, error) {
	return genericID(id).marshalText("L2Cluster")
}

func (id *L2ClusterID) UnmarshalText(data []byte) error {
	return (*genericID)(id).unmarshalText("L2Cluster", data)
}

func SortL2ClusterIDs(ids []L2ClusterID) []L2ClusterID {
	return copyAndSortCmp(ids)
}

// Cluster represents a set of L2 chains that interop
type Cluster interface {
	Common
	ID() L2ClusterID

	DependencySet() depset.DependencySet
}

// ClusterConfig is the config to create a default cluster object
type ClusterConfig struct {
	CommonConfig
	DepSet depset.DependencySet
	ID     L2ClusterID
}

// presetCluster implements Cluster with preset values
type presetCluster struct {
	commonImpl
	depSet depset.DependencySet
	id     L2ClusterID
}

var _ Cluster = (*presetCluster)(nil)

func NewCluster(cfg ClusterConfig) Cluster {
	cfg.Log = cfg.Log.New("id", cfg.ID)
	return &presetCluster{
		id:         cfg.ID,
		commonImpl: newCommon(cfg.CommonConfig),
		depSet:     cfg.DepSet,
	}
}

func (p *presetCluster) ID() L2ClusterID {
	return p.id
}

func (p *presetCluster) DependencySet() depset.DependencySet {
	return p.depSet
}
