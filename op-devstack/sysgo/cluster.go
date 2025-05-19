package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
)

type Cluster struct {
	id     stack.ClusterID
	depset *depset.StaticConfigDependencySet
}

func (c *Cluster) hydrate(system stack.ExtensibleSystem) {
	sysCluster := shim.NewCluster(shim.ClusterConfig{
		CommonConfig:  shim.NewCommonConfig(system.T()),
		ID:            c.id,
		DependencySet: c.depset,
	})
	system.AddCluster(sysCluster)
}
