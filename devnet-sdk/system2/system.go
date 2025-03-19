package system2

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/locks"
)

// System represents a collection of L1 and L2 chains, any superchains or clusters, and any peripherals.
type System interface {
	Common

	Superchain(id SuperchainID) Superchain
	Cluster(id L2ClusterID) Cluster
	L1Chain(id L1ChainID) L1Chain
	L2Chain(id L2ChainID) L2Chain

	Superchains() []SuperchainID
	Clusters() []L2ClusterID
	L1Chains() []L1ChainID
	L2Chains() []L2ChainID

	// L1ChainID looks up the L1ChainID (system name) by eth ChainID
	L1ChainID(id eth.ChainID) L1ChainID
	// L2ChainID looks up the L2ChainID (system name) by eth ChainID
	L2ChainID(id eth.ChainID) L2ChainID

	User(id UserID) User
	Users() []UserID
}

// ExtensibleSystem is an extension-interface to add new components to the system.
// Regular tests should not be modifying the system.
// Test gates may use this to remediate any shortcomings of an existing system.
type ExtensibleSystem interface {
	System
	AddSuperchain(v Superchain)
	AddCluster(v Cluster)
	AddL1Chain(v L1Chain)
	AddL2Chain(v L2Chain)
	AddUser(v User)
}

// SystemConfig sets up a System.
// It is intentially very minimal, the system is expected to be extended after creation, using Option functions
type SystemConfig struct {
	CommonConfig
}

type presetSystem struct {
	commonImpl

	superchains locks.RWMap[SuperchainID, Superchain]
	clusters    locks.RWMap[L2ClusterID, Cluster]

	// tracks L1 chains by name
	l1Chains locks.RWMap[L1ChainID, L1Chain]
	// tracks L2 chains by name
	l2Chains locks.RWMap[L2ChainID, L2Chain]

	// tracks names of L1 chains by eth.ChainID
	l1ChainIDs locks.RWMap[eth.ChainID, L1ChainID]
	// tracks names of L2 chains by eth.ChainID
	l2ChainIDs locks.RWMap[eth.ChainID, L2ChainID]

	// tracks all chains, and ensures there are no chains with the same eth.ChainID
	chains locks.RWMap[eth.ChainID, Chain]

	users locks.RWMap[UserID, User]
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

func (p *presetSystem) Cluster(id L2ClusterID) Cluster {
	v, ok := p.clusters.Get(id)
	p.require().True(ok, "cluster %s must exist", id)
	return v
}

func (p *presetSystem) AddCluster(v Cluster) {
	p.require().True(p.clusters.SetIfMissing(v.ID(), v), "cluster %s must not already exist", v.ID())
}

func (p *presetSystem) L1Chain(id L1ChainID) L1Chain {
	v, ok := p.l1Chains.Get(id)
	p.require().True(ok, "l1 chain %s must exist", id)
	return v
}

func (p *presetSystem) AddL1Chain(v L1Chain) {
	id := v.ID()
	p.require().True(p.chains.SetIfMissing(id.ChainID, v), "chain with id %s must not already exist", id.ChainID)
	p.require().True(p.l1ChainIDs.SetIfMissing(id.ChainID, id), "l1 chain id %s mapping must not already exist", id)
	p.require().True(p.l1Chains.SetIfMissing(id, v), "l1 chain %s must not already exist", id)
}

func (p *presetSystem) L2Chain(id L2ChainID) L2Chain {
	v, ok := p.l2Chains.Get(id)
	p.require().True(ok, "l2 chain %s must exist", id)
	return v
}

func (p *presetSystem) AddL2Chain(v L2Chain) {
	id := v.ID()
	p.require().True(p.chains.SetIfMissing(id.ChainID, v), "chain with id %s must not already exist", id.ChainID)
	p.require().True(p.l2ChainIDs.SetIfMissing(id.ChainID, id), "l2 chain id %s mapping must not already exist", id)
	p.require().True(p.l2Chains.SetIfMissing(id, v), "l2 chain %s must not already exist", id)
}

func (p *presetSystem) L1ChainID(id eth.ChainID) L1ChainID {
	v, ok := p.l1ChainIDs.Get(id)
	p.require().True(ok, "l1 chain id %s mapping must exist", id)
	return v
}

func (p *presetSystem) L2ChainID(id eth.ChainID) L2ChainID {
	v, ok := p.l2ChainIDs.Get(id)
	p.require().True(ok, "l2 chain id %s mapping must exist", id)
	return v
}

func (p *presetSystem) User(id UserID) User {
	v, ok := p.users.Get(id)
	p.require().True(ok, "user %s must exist", id)
	return v
}

func (p *presetSystem) AddUser(v User) {
	p.require().True(p.users.SetIfMissing(v.ID(), v), "user %s must not already exist", v.ID())
}

func (p *presetSystem) Superchains() []SuperchainID {
	return SortSuperchainIDs(p.superchains.Keys())
}

func (p *presetSystem) Clusters() []L2ClusterID {
	return SortL2ClusterIDs(p.clusters.Keys())
}

func (p *presetSystem) L1Chains() []L1ChainID {
	return SortL1ChainIDs(p.l1Chains.Keys())
}

func (p *presetSystem) L2Chains() []L2ChainID {
	return SortL2ChainIDs(p.l2Chains.Keys())
}

func (p *presetSystem) Users() []UserID {
	return SortUserIDs(p.users.Keys())
}
