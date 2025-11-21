package shim

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
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

	superchains locks.RWMap[stack.ComponentID, stack.Superchain]

	// tracks L1 networks by L1NetworkID (a typed eth.ChainID)
	l1Networks locks.RWMap[stack.ComponentID, stack.L1Network]
	// tracks L2 networks by L2NetworkID (a typed eth.ChainID)
	l2Networks locks.RWMap[stack.ComponentID, stack.L2Network]

	// tracks all networks, and ensures there are no networks with the same eth.ChainID
	networks locks.RWMap[stack.ComponentID, stack.Network]

	supervisors locks.RWMap[stack.ComponentID, stack.Supervisor]
	sequencers  locks.RWMap[stack.ComponentID, stack.TestSequencer]
	syncTesters locks.RWMap[stack.ComponentID, stack.SyncTester]
}

var _ stack.ExtensibleSystem = (*presetSystem)(nil)

// NewSystem creates a new empty System
func NewSystem(t devtest.T) stack.ExtensibleSystem {
	return &presetSystem{
		commonImpl: newCommon(NewCommonConfig(t)),
	}
}

func (p *presetSystem) Superchain(m stack.SuperchainMatcher) stack.Superchain {
	v, ok := findMatch(m, p.superchains.Values())
	p.require().True(ok, "must find superchain %s", m)
	return v
}

func (p *presetSystem) AddSuperchain(v stack.Superchain) {
	p.require().True(p.superchains.SetIfMissing(v.ID(), v), "superchain %s must not already exist", v.ID())
}

func (p *presetSystem) Network(id stack.ComponentID) stack.Network {
	if l1Net, ok := p.l1Networks.Get(id); ok {
		return l1Net
	}
	if l2Net, ok := p.l2Networks.Get(id); ok {
		return l2Net
	}
	p.t.FailNow()
	return nil
}

func (p *presetSystem) L1Network(m stack.L1NetworkMatcher) stack.L1Network {
	v, ok := findMatch(m, p.l1Networks.Values())

	p.require().True(ok, "must find l1 network %s", m)
	return v
}

func (p *presetSystem) AddL1Network(v stack.L1Network) {
	id := v.ID()
	p.require().True(p.networks.SetIfMissing(id.ChainID, v), "chain with id %s must not already exist", id.ChainID)
	p.require().True(p.l1Networks.SetIfMissing(id, v), "L1 chain %s must not already exist", id)
}

func (p *presetSystem) L2Network(m stack.L2NetworkMatcher) stack.L2Network {
	v, ok := findMatch(m, p.l2Networks.Values())
	p.require().True(ok, "must find l2 network %s", m)
	return v
}

func (p *presetSystem) AddL2Network(v stack.L2Network) {
	id := v.ID()
	p.require().True(p.networks.SetIfMissing(id, v), "chain with id %s must not already exist", id)
	p.require().True(p.l2Networks.SetIfMissing(id, v), "L2 chain %s must not already exist", id)
}

func (p *presetSystem) Supervisor(m stack.SupervisorMatcher) stack.Supervisor {
	v, ok := findMatch(m, p.supervisors.Values())
	p.require().True(ok, "must find supervisor %s", m)
	return v
}

func (p *presetSystem) AddSupervisor(v stack.Supervisor) {
	p.require().True(p.supervisors.SetIfMissing(v.ID(), v), "supervisor %s must not already exist", v.ID())
}

func (p *presetSystem) TestSequencer(m stack.TestSequencerMatcher) stack.TestSequencer {
	v, ok := findMatch(m, p.sequencers.Values())
	p.require().True(ok, "must find sequencer %s", m)
	return v
}

func (p *presetSystem) AddTestSequencer(v stack.TestSequencer) {
	p.require().True(p.sequencers.SetIfMissing(v.ID(), v), "sequencer %s must not already exist", v.ID())
}

func (p *presetSystem) AddSyncTester(v stack.SyncTester) {
	p.require().True(p.syncTesters.SetIfMissing(v.ID(), v), "sync tester %s must not already exist", v.ID())
}

func (p *presetSystem) Superchains() []stack.Superchain {
	return stack.SortSuperchains(p.superchains.Values())
}

func (p *presetSystem) L1Networks() []stack.L1Network {
	return stack.SortL1Networks(p.l1Networks.Values())
}

func (p *presetSystem) L2Networks() []stack.L2Network {
	return stack.SortL2Networks(p.l2Networks.Values())
}

func (p *presetSystem) Supervisors() []stack.Supervisor {
	return stack.SortSupervisors(p.supervisors.Values())
}

func (p *presetSystem) TestSequencers() []stack.TestSequencer {
	return stack.SortTestSequencers(p.sequencers.Values())
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
