package shim

import (
	"github.com/HashKeyChain/verse/op-devstack/stack"
	"github.com/HashKeyChain/verse/op-supervisor/supervisor/backend/depset"
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
	cfg.T = cfg.T.WithCtx(stack.ContextWithID(cfg.T.Ctx(), cfg.ID))
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
