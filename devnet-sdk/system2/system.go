package system2

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/locks"
)

// System represents a collection of L1 and L2 chains, any superchains or clusters, and any peripherals.
type System interface {
	Common

	Superchain(id SuperchainID) Superchain
	Cluster(id ClusterID) Cluster
	L1Network(id L1NetworkID) L1Network
	L2Network(id L2NetworkID) L2Network

	Superchains() []SuperchainID
	Clusters() []ClusterID
	L1Networks() []L1NetworkID
	L2Networks() []L2NetworkID

	// L1NetworkID looks up the L1NetworkID (system name) by eth ChainID
	L1NetworkID(id eth.ChainID) L1NetworkID
	// L2NetworkID looks up the L2NetworkID (system name) by eth ChainID
	L2NetworkID(id eth.ChainID) L2NetworkID

	Supervisor(id SupervisorID) Supervisor
	Supervisors() []SupervisorID
}

// ExtensibleSystem is an extension-interface to add new components to the system.
// Regular tests should not be modifying the system.
// Test gates may use this to remediate any shortcomings of an existing system.
type ExtensibleSystem interface {
	System
	AddSuperchain(v Superchain)
	AddCluster(v Cluster)
	AddL1Network(v L1Network)
	AddL2Network(v L2Network)
	AddSupervisor(v Supervisor)
}

// SystemConfig sets up a System.
// It is intentially very minimal, the system is expected to be extended after creation, using Option functions
type SystemConfig struct {
	CommonConfig
}

type presetSystem struct {
	commonImpl

	superchains locks.RWMap[SuperchainID, Superchain]
	clusters    locks.RWMap[ClusterID, Cluster]

	// tracks L1 networks by name
	l1Networks locks.RWMap[L1NetworkID, L1Network]
	// tracks L2 networks by name
	l2Networks locks.RWMap[L2NetworkID, L2Network]

	// tracks IDs of L1 networks by eth.ChainID
	l1ChainIDs locks.RWMap[eth.ChainID, L1NetworkID]
	// tracks IDs of L2 networks by eth.ChainID
	l2ChainIDs locks.RWMap[eth.ChainID, L2NetworkID]

	// tracks all networks, and ensures there are no networks with the same eth.ChainID
	networks locks.RWMap[eth.ChainID, Network]

	supervisors locks.RWMap[SupervisorID, Supervisor]
}

var _ ExtensibleSystem = (*presetSystem)(nil)

// NewSystem creates a new empty System
func NewSystem(cfg SystemConfig) ExtensibleSystem {
	return &presetSystem{
		commonImpl: newCommon(cfg.CommonConfig),
	}
}

func (p *presetSystem) Superchain(id SuperchainID) Superchain {
	v, ok := p.superchains.Get(id)
	p.require().True(ok, "superchain %s must exist", id)
	return v
}

func (p *presetSystem) AddSuperchain(v Superchain) {
	p.require().True(p.superchains.SetIfMissing(v.ID(), v), "superchain %s must not already exist", v.ID())
}

func (p *presetSystem) Cluster(id ClusterID) Cluster {
	v, ok := p.clusters.Get(id)
	p.require().True(ok, "cluster %s must exist", id)
	return v
}

func (p *presetSystem) AddCluster(v Cluster) {
	p.require().True(p.clusters.SetIfMissing(v.ID(), v), "cluster %s must not already exist", v.ID())
}

func (p *presetSystem) L1Network(id L1NetworkID) L1Network {
	v, ok := p.l1Networks.Get(id)
	p.require().True(ok, "l1 chain %s must exist", id)
	return v
}

func (p *presetSystem) AddL1Network(v L1Network) {
	id := v.ID()
	p.require().True(p.networks.SetIfMissing(id.ChainID, v), "chain with id %s must not already exist", id.ChainID)
	p.require().True(p.l1ChainIDs.SetIfMissing(id.ChainID, id), "l1 chain id %s mapping must not already exist", id)
	p.require().True(p.l1Networks.SetIfMissing(id, v), "l1 chain %s must not already exist", id)
}

func (p *presetSystem) L2Network(id L2NetworkID) L2Network {
	v, ok := p.l2Networks.Get(id)
	p.require().True(ok, "l2 chain %s must exist", id)
	return v
}

func (p *presetSystem) AddL2Network(v L2Network) {
	id := v.ID()
	p.require().True(p.networks.SetIfMissing(id.ChainID, v), "chain with id %s must not already exist", id.ChainID)
	p.require().True(p.l2ChainIDs.SetIfMissing(id.ChainID, id), "l2 chain id %s mapping must not already exist", id)
	p.require().True(p.l2Networks.SetIfMissing(id, v), "l2 chain %s must not already exist", id)
}

func (p *presetSystem) L1NetworkID(id eth.ChainID) L1NetworkID {
	v, ok := p.l1ChainIDs.Get(id)
	p.require().True(ok, "l1 chain id %s mapping must exist", id)
	return v
}

func (p *presetSystem) L2NetworkID(id eth.ChainID) L2NetworkID {
	v, ok := p.l2ChainIDs.Get(id)
	p.require().True(ok, "l2 chain id %s mapping must exist", id)
	return v
}

func (p *presetSystem) Supervisor(id SupervisorID) Supervisor {
	v, ok := p.supervisors.Get(id)
	p.require().True(ok, "supervisor %s must exist", id)
	return v
}

func (p *presetSystem) AddSupervisor(v Supervisor) {
	p.require().True(p.supervisors.SetIfMissing(v.ID(), v), "supervisor %s must not already exist", v.ID())
}

func (p *presetSystem) Superchains() []SuperchainID {
	return SortSuperchainIDs(p.superchains.Keys())
}

func (p *presetSystem) Clusters() []ClusterID {
	return SortClusterIDs(p.clusters.Keys())
}

func (p *presetSystem) L1Networks() []L1NetworkID {
	return SortL1NetworkIDs(p.l1Networks.Keys())
}

func (p *presetSystem) L2Networks() []L2NetworkID {
	return SortL2NetworkIDs(p.l2Networks.Keys())
}

func (p *presetSystem) Supervisors() []SupervisorID {
	return SortSupervisorIDs(p.supervisors.Keys())
}
