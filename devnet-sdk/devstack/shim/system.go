package shim

import (
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/devtest"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/locks"
)

// SystemConfig sets up a System.
// It is intentially very minimal, the system is expected to be extended after creation, using Option functions
type SystemConfig struct {
	CommonConfig
}

type presetSystem struct {
	commonImpl

	superchains locks.RWMap[stack.SuperchainID, stack.Superchain]
	clusters    locks.RWMap[stack.ClusterID, stack.Cluster]

	// tracks L1 networks by name
	l1Networks locks.RWMap[stack.L1NetworkID, stack.L1Network]
	// tracks L2 networks by name
	l2Networks locks.RWMap[stack.L2NetworkID, stack.L2Network]

	// tracks IDs of L1 networks by eth.ChainID
	l1ChainIDs locks.RWMap[eth.ChainID, stack.L1NetworkID]
	// tracks IDs of L2 networks by eth.ChainID
	l2ChainIDs locks.RWMap[eth.ChainID, stack.L2NetworkID]

	// tracks all networks, and ensures there are no networks with the same eth.ChainID
	networks locks.RWMap[eth.ChainID, stack.Network]

	supervisors locks.RWMap[stack.SupervisorID, stack.Supervisor]
}

var _ stack.ExtensibleSystem = (*presetSystem)(nil)

// NewSystem creates a new empty System
func NewSystem(t devtest.T) stack.ExtensibleSystem {
	return &presetSystem{
		commonImpl: newCommon(NewCommonConfig(t)),
	}
}

func (p *presetSystem) Superchain(id stack.SuperchainID) stack.Superchain {
	v, ok := p.superchains.Get(id)
	p.require().True(ok, "superchain %s must exist", id)
	return v
}

func (p *presetSystem) AddSuperchain(v stack.Superchain) {
	p.require().True(p.superchains.SetIfMissing(v.ID(), v), "superchain %s must not already exist", v.ID())
}

func (p *presetSystem) Cluster(id stack.ClusterID) stack.Cluster {
	v, ok := p.clusters.Get(id)
	p.require().True(ok, "cluster %s must exist", id)
	return v
}

func (p *presetSystem) AddCluster(v stack.Cluster) {
	p.require().True(p.clusters.SetIfMissing(v.ID(), v), "cluster %s must not already exist", v.ID())
}

func (p *presetSystem) L1Network(id stack.L1NetworkID) stack.L1Network {
	v, ok := p.l1Networks.Get(id)
	p.require().True(ok, "l1 chain %s must exist", id)
	return v
}

func (p *presetSystem) AddL1Network(v stack.L1Network) {
	id := v.ID()
	p.require().True(p.networks.SetIfMissing(id.ChainID, v), "chain with id %s must not already exist", id.ChainID)
	p.require().True(p.l1ChainIDs.SetIfMissing(id.ChainID, id), "l1 chain id %s mapping must not already exist", id)
	p.require().True(p.l1Networks.SetIfMissing(id, v), "l1 chain %s must not already exist", id)
}

func (p *presetSystem) L2Network(id stack.L2NetworkID) stack.L2Network {
	v, ok := p.l2Networks.Get(id)
	p.require().True(ok, "l2 chain %s must exist", id)
	return v
}

func (p *presetSystem) AddL2Network(v stack.L2Network) {
	id := v.ID()
	p.require().True(p.networks.SetIfMissing(id.ChainID, v), "chain with id %s must not already exist", id.ChainID)
	p.require().True(p.l2ChainIDs.SetIfMissing(id.ChainID, id), "l2 chain id %s mapping must not already exist", id)
	p.require().True(p.l2Networks.SetIfMissing(id, v), "l2 chain %s must not already exist", id)
}

func (p *presetSystem) L1NetworkID(id eth.ChainID) stack.L1NetworkID {
	v, ok := p.l1ChainIDs.Get(id)
	p.require().True(ok, "l1 chain id %s mapping must exist", id)
	return v
}

func (p *presetSystem) L2NetworkID(id eth.ChainID) stack.L2NetworkID {
	v, ok := p.l2ChainIDs.Get(id)
	p.require().True(ok, "l2 chain id %s mapping must exist", id)
	return v
}

func (p *presetSystem) Supervisor(id stack.SupervisorID) stack.Supervisor {
	v, ok := p.supervisors.Get(id)
	p.require().True(ok, "supervisor %s must exist", id)
	return v
}

func (p *presetSystem) AddSupervisor(v stack.Supervisor) {
	p.require().True(p.supervisors.SetIfMissing(v.ID(), v), "supervisor %s must not already exist", v.ID())
}

func (p *presetSystem) Superchains() []stack.SuperchainID {
	return stack.SortSuperchainIDs(p.superchains.Keys())
}

func (p *presetSystem) Clusters() []stack.ClusterID {
	return stack.SortClusterIDs(p.clusters.Keys())
}

func (p *presetSystem) L1Networks() []stack.L1NetworkID {
	return stack.SortL1NetworkIDs(p.l1Networks.Keys())
}

func (p *presetSystem) L2Networks() []stack.L2NetworkID {
	return stack.SortL2NetworkIDs(p.l2Networks.Keys())
}

func (p *presetSystem) Supervisors() []stack.SupervisorID {
	return stack.SortSupervisorIDs(p.supervisors.Keys())
}
