package shim

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/locks"
)

// SystemConfig sets up a System.
// It is intentionally very minimal, the system is expected to be extended after creation, using Option functions
type SystemConfig struct {
	CommonConfig
}

type presetSystem struct {
	commonImpl

	// timeTravelClock is the clock used to control time. nil if time travel is not enabled
	timeTravelClock stack.TimeTravelClock

	// Unified component registry for generic access
	registry *stack.Registry

	supernodes locks.RWMap[stack.ComponentID, stack.Supernode]
}

var _ stack.ExtensibleSystem = (*presetSystem)(nil)

// NewSystem creates a new empty System
func NewSystem(t devtest.T) stack.ExtensibleSystem {
	return &presetSystem{
		commonImpl: newCommon(NewCommonConfig(t)),
		registry:   stack.NewRegistry(),
	}
}

// --- ComponentRegistry interface implementation ---

func (p *presetSystem) Component(id stack.ComponentID) (any, bool) {
	return p.registry.Get(id)
}

func (p *presetSystem) Components(kind stack.ComponentKind) []any {
	ids := p.registry.IDsByKind(kind)
	result := make([]any, 0, len(ids))
	for _, id := range ids {
		if comp, ok := p.registry.Get(id); ok {
			result = append(result, comp)
		}
	}
	return result
}

func (p *presetSystem) ComponentIDs(kind stack.ComponentKind) []stack.ComponentID {
	return p.registry.IDsByKind(kind)
}

func (p *presetSystem) Superchain(m stack.SuperchainMatcher) stack.Superchain {
	getter := func(id stack.ComponentID) (stack.Superchain, bool) {
		v, ok := p.registry.Get(id)
		if !ok {
			return nil, false
		}
		return v.(stack.Superchain), true
	}
	v, ok := findMatch(m, getter, p.Superchains)
	p.require().True(ok, "must find superchain %s", m)
	return v
}

func (p *presetSystem) AddSuperchain(v stack.Superchain) {
	id := v.ID()
	_, exists := p.registry.Get(id)
	p.require().False(exists, "superchain %s must not already exist", id)
	p.registry.Register(id, v)
}

func (p *presetSystem) Cluster(m stack.ClusterMatcher) stack.Cluster {
	getter := func(id stack.ComponentID) (stack.Cluster, bool) {
		v, ok := p.registry.Get(id)
		if !ok {
			return nil, false
		}
		return v.(stack.Cluster), true
	}
	v, ok := findMatch(m, getter, p.Clusters)
	p.require().True(ok, "must find cluster %s", m)
	return v
}

func (p *presetSystem) AddCluster(v stack.Cluster) {
	id := v.ID()
	_, exists := p.registry.Get(id)
	p.require().False(exists, "cluster %s must not already exist", id)
	p.registry.Register(id, v)
}

// networkExistsByChainID checks if any network (L1 or L2) exists with the given chain ID
func (p *presetSystem) networkExistsByChainID(chainID eth.ChainID) bool {
	l1ID := stack.NewL1NetworkID(chainID)
	if _, ok := p.registry.Get(l1ID); ok {
		return true
	}
	l2ID := stack.NewL2NetworkID(chainID)
	if _, ok := p.registry.Get(l2ID); ok {
		return true
	}
	return false
}

func (p *presetSystem) Network(id eth.ChainID) stack.Network {
	l1ID := stack.NewL1NetworkID(id)
	if l1Net, ok := p.registry.Get(l1ID); ok {
		return l1Net.(stack.L1Network)
	}
	l2ID := stack.NewL2NetworkID(id)
	if l2Net, ok := p.registry.Get(l2ID); ok {
		return l2Net.(stack.L2Network)
	}
	p.t.FailNow()
	return nil
}

func (p *presetSystem) L1Network(m stack.L1NetworkMatcher) stack.L1Network {
	getter := func(id stack.ComponentID) (stack.L1Network, bool) {
		v, ok := p.registry.Get(id)
		if !ok {
			return nil, false
		}
		return v.(stack.L1Network), true
	}
	v, ok := findMatch(m, getter, p.L1Networks)
	p.require().True(ok, "must find l1 network %s", m)
	return v
}

func (p *presetSystem) AddL1Network(v stack.L1Network) {
	id := v.ID()
	p.require().False(p.networkExistsByChainID(id.ChainID()), "chain with id %s must not already exist", id.ChainID())
	_, exists := p.registry.Get(id)
	p.require().False(exists, "L1 chain %s must not already exist", id)
	p.registry.Register(id, v)
}

func (p *presetSystem) L2Network(m stack.L2NetworkMatcher) stack.L2Network {
	getter := func(id stack.ComponentID) (stack.L2Network, bool) {
		v, ok := p.registry.Get(id)
		if !ok {
			return nil, false
		}
		return v.(stack.L2Network), true
	}
	v, ok := findMatch(m, getter, p.L2Networks)
	p.require().True(ok, "must find l2 network %s", m)
	return v
}

func (p *presetSystem) AddL2Network(v stack.L2Network) {
	id := v.ID()
	p.require().False(p.networkExistsByChainID(id.ChainID()), "chain with id %s must not already exist", id.ChainID())
	_, exists := p.registry.Get(id)
	p.require().False(exists, "L2 chain %s must not already exist", id)
	p.registry.Register(id, v)
}

func (p *presetSystem) Supervisor(m stack.SupervisorMatcher) stack.Supervisor {
	getter := func(id stack.ComponentID) (stack.Supervisor, bool) {
		v, ok := p.registry.Get(id)
		if !ok {
			return nil, false
		}
		return v.(stack.Supervisor), true
	}
	v, ok := findMatch(m, getter, p.Supervisors)
	p.require().True(ok, "must find supervisor %s", m)
	return v
}

func (p *presetSystem) AddSupervisor(v stack.Supervisor) {
	id := v.ID()
	_, exists := p.registry.Get(id)
	p.require().False(exists, "supervisor %s must not already exist", id)
	p.registry.Register(id, v)
}

func (p *presetSystem) Supernode(m stack.SupernodeMatcher) stack.Supernode {
	v, ok := findMatch(m, p.supernodes.Get, p.Supernodes)
	p.require().True(ok, "must find supernode %s", m)
	return v
}

func (p *presetSystem) AddSupernode(v stack.Supernode) {
	p.require().True(p.supernodes.SetIfMissing(v.ID(), v), "supernode %s must not already exist", v.ID())
}

func (p *presetSystem) TestSequencer(m stack.TestSequencerMatcher) stack.TestSequencer {
	getter := func(id stack.ComponentID) (stack.TestSequencer, bool) {
		v, ok := p.registry.Get(id)
		if !ok {
			return nil, false
		}
		return v.(stack.TestSequencer), true
	}
	v, ok := findMatch(m, getter, p.TestSequencers)
	p.require().True(ok, "must find sequencer %s", m)
	return v
}

func (p *presetSystem) AddTestSequencer(v stack.TestSequencer) {
	id := v.ID()
	_, exists := p.registry.Get(id)
	p.require().False(exists, "sequencer %s must not already exist", id)
	p.registry.Register(id, v)
}

func (p *presetSystem) AddSyncTester(v stack.SyncTester) {
	id := v.ID()
	_, exists := p.registry.Get(id)
	p.require().False(exists, "sync tester %s must not already exist", id)
	p.registry.Register(id, v)
}

func (p *presetSystem) SuperchainIDs() []stack.ComponentID {
	return sortByID(p.registry.IDsByKind(stack.KindSuperchain))
}

func (p *presetSystem) Superchains() []stack.Superchain {
	ids := p.registry.IDsByKind(stack.KindSuperchain)
	result := make([]stack.Superchain, 0, len(ids))
	for _, id := range ids {
		if v, ok := p.registry.Get(id); ok {
			result = append(result, v.(stack.Superchain))
		}
	}
	return sortByIDFunc(result)
}

func (p *presetSystem) ClusterIDs() []stack.ComponentID {
	return sortByID(p.registry.IDsByKind(stack.KindCluster))
}

func (p *presetSystem) Clusters() []stack.Cluster {
	ids := p.registry.IDsByKind(stack.KindCluster)
	result := make([]stack.Cluster, 0, len(ids))
	for _, id := range ids {
		if v, ok := p.registry.Get(id); ok {
			result = append(result, v.(stack.Cluster))
		}
	}
	return sortByIDFunc(result)
}

func (p *presetSystem) L1NetworkIDs() []stack.ComponentID {
	return sortByID(p.registry.IDsByKind(stack.KindL1Network))
}

func (p *presetSystem) L1Networks() []stack.L1Network {
	ids := p.registry.IDsByKind(stack.KindL1Network)
	result := make([]stack.L1Network, 0, len(ids))
	for _, id := range ids {
		if v, ok := p.registry.Get(id); ok {
			result = append(result, v.(stack.L1Network))
		}
	}
	return sortByIDFunc(result)
}

func (p *presetSystem) L2NetworkIDs() []stack.ComponentID {
	return sortByID(p.registry.IDsByKind(stack.KindL2Network))
}

func (p *presetSystem) L2Networks() []stack.L2Network {
	ids := p.registry.IDsByKind(stack.KindL2Network)
	result := make([]stack.L2Network, 0, len(ids))
	for _, id := range ids {
		if v, ok := p.registry.Get(id); ok {
			result = append(result, v.(stack.L2Network))
		}
	}
	return sortByIDFunc(result)
}

func (p *presetSystem) SupervisorIDs() []stack.ComponentID {
	return sortByID(p.registry.IDsByKind(stack.KindSupervisor))
}

func (p *presetSystem) Supervisors() []stack.Supervisor {
	ids := p.registry.IDsByKind(stack.KindSupervisor)
	result := make([]stack.Supervisor, 0, len(ids))
	for _, id := range ids {
		if v, ok := p.registry.Get(id); ok {
			result = append(result, v.(stack.Supervisor))
		}
	}
	return sortByIDFunc(result)
}

func (p *presetSystem) Supernodes() []stack.Supernode {
	return stack.SortSupernodes(p.supernodes.Values())
}

func (p *presetSystem) TestSequencers() []stack.TestSequencer {
	ids := p.registry.IDsByKind(stack.KindTestSequencer)
	result := make([]stack.TestSequencer, 0, len(ids))
	for _, id := range ids {
		if v, ok := p.registry.Get(id); ok {
			result = append(result, v.(stack.TestSequencer))
		}
	}
	return sortByIDFunc(result)
}

func (p *presetSystem) SetTimeTravelClock(cl stack.TimeTravelClock) {
	p.timeTravelClock = cl
}

func (p *presetSystem) TimeTravelEnabled() bool {
	return p.timeTravelClock != nil
}

func (p *presetSystem) AdvanceTime(amount time.Duration) {
	p.require().True(p.TimeTravelEnabled(), "Attempting to advance time when time travel is not enabled")
	p.timeTravelClock.AdvanceTime(amount)
}
