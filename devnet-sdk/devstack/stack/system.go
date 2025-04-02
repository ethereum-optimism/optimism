package stack

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
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
