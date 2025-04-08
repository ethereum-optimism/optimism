package shim

import (
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
)

// ClusterConfig is the config to create a default cluster object
type ClusterConfig struct {
	CommonConfig
	DependencySet depset.DependencySet
	ID            stack.ClusterID
}

// presetCluster implements Cluster with preset values
type presetCluster struct {
	commonImpl
	depSet depset.DependencySet
	id     stack.ClusterID
}

var _ stack.Cluster = (*presetCluster)(nil)

func NewCluster(cfg ClusterConfig) stack.Cluster {
	cfg.Log = cfg.Log.New("id", cfg.ID)
	return &presetCluster{
		id:         cfg.ID,
		commonImpl: newCommon(cfg.CommonConfig),
		depSet:     cfg.DependencySet,
	}
}

func (p *presetCluster) ID() stack.ClusterID {
	return p.id
}

func (p *presetCluster) DependencySet() depset.DependencySet {
	return p.depSet
}
