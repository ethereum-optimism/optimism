package stack

import "github.com/ethereum-optimism/optimism/op-service/eth"

// System represents a collection of L1 and L2 chains, any superchains or clusters, and any peripherals.
type System interface {
	Common

	Superchain(m SuperchainMatcher) Superchain
	Cluster(m ClusterMatcher) Cluster
	L1Network(m L1NetworkMatcher) L1Network
	L2Network(m L2NetworkMatcher) L2Network

	Network(id eth.ChainID) Network

	Supervisor(m SupervisorMatcher) Supervisor
	Sequencer(id SequencerMatcher) Sequencer

	SuperchainIDs() []SuperchainID
	ClusterIDs() []ClusterID
	L1NetworkIDs() []L1NetworkID
	L2NetworkIDs() []L2NetworkID
	SupervisorIDs() []SupervisorID

	Superchains() []Superchain
	Clusters() []Cluster
	L1Networks() []L1Network
	L2Networks() []L2Network
	Supervisors() []Supervisor
	Sequencers() []Sequencer
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
	AddSequencer(v Sequencer)
}
