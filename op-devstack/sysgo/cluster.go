package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
)

type Cluster struct {
	id     stack.ClusterID
	cfgset depset.FullConfigSetMerged
}

func (c *Cluster) hydrate(system stack.ExtensibleSystem) {
	sysCluster := shim.NewCluster(shim.ClusterConfig{
		CommonConfig:  shim.NewCommonConfig(system.T()),
		ID:            c.id,
		DependencySet: c.cfgset.DependencySet,
	})
	system.AddCluster(sysCluster)
}

func (c *Cluster) DepSet() *depset.StaticConfigDependencySet {
	return c.cfgset.DependencySet.(*depset.StaticConfigDependencySet)
}
