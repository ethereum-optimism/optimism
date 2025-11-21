package stack

import (
	"time"
)

// System represents a collection of L1 and L2 chains, any superchains or clusters, and any peripherals.
type System interface {
	Common

	Superchain(m SuperchainMatcher) Superchain
	L1Network(m L1NetworkMatcher) L1Network
	L2Network(m L2NetworkMatcher) L2Network

	Network(id ComponentID) Network

	Supervisor(m SupervisorMatcher) Supervisor
	TestSequencer(id TestSequencerMatcher) TestSequencer

	Superchains() []Superchain
	L1Networks() []L1Network
	L2Networks() []L2Network
	Supervisors() []Supervisor
	TestSequencers() []TestSequencer
}

// ExtensibleSystem is an extension-interface to add new components to the system.
// Regular tests should not be modifying the system.
// Test gates may use this to remediate any shortcomings of an existing system.
type ExtensibleSystem interface {
	System
	AddSuperchain(v Superchain)
	AddL1Network(v L1Network)
	AddL2Network(v L2Network)
	AddSupervisor(v Supervisor)
	AddTestSequencer(v TestSequencer)
	AddSyncTester(v SyncTester)
}

type TimeTravelClock interface {
	AdvanceTime(d time.Duration)
}

// TimeTravelSystem is an extension-interface to support time travel.
type TimeTravelSystem interface {
	System
	SetTimeTravelClock(cl TimeTravelClock)
	TimeTravelEnabled() bool
	AdvanceTime(amount time.Duration)
}
