package stack

// ComponentRegistry provides generic component access for systems and networks.
// This interface enables unified component lookup regardless of component type,
// reducing the need for type-specific getter methods on container interfaces.
//
// Components are stored by ComponentID and can be queried by:
// - Exact ID match (Component)
// - Kind (Components, ComponentIDs)
//
// Implementations should use the Registry type internally for storage.
type ComponentRegistry interface {
	// Component returns a component by its ID.
	// Returns (component, true) if found, (nil, false) otherwise.
	Component(id ComponentID) (any, bool)

	// Components returns all components of a given kind.
	// Returns an empty slice if no components of that kind exist.
	Components(kind ComponentKind) []any

	// ComponentIDs returns all component IDs of a given kind.
	// Returns an empty slice if no components of that kind exist.
	ComponentIDs(kind ComponentKind) []ComponentID
}

// --- Free functions for typed component access ---
// These functions provide type-safe access to components without requiring
// type-specific methods on every container interface.

// GetComponent returns a typed component from a registry by ID.
// Returns (component, true) if found and type matches, (nil/zero, false) otherwise.
func GetComponent[T any](r ComponentRegistry, id ComponentID) (T, bool) {
	comp, ok := r.Component(id)
	if !ok {
		var zero T
		return zero, false
	}
	typed, ok := comp.(T)
	return typed, ok
}

// GetComponentsByKind returns all components of a given kind, typed.
// Only components that match the expected type are returned.
func GetComponentsByKind[T any](r ComponentRegistry, kind ComponentKind) []T {
	comps := r.Components(kind)
	result := make([]T, 0, len(comps))
	for _, comp := range comps {
		if typed, ok := comp.(T); ok {
			result = append(result, typed)
		}
	}
	return result
}

// --- Typed getter free functions for L2Network components ---

// GetL2BatcherByID returns an L2Batcher from a network by ID.
func GetL2BatcherByID(n L2Network, id ComponentID) (L2Batcher, bool) {
	return GetComponent[L2Batcher](n, id)
}

// GetL2ProposerByID returns an L2Proposer from a network by ID.
func GetL2ProposerByID(n L2Network, id ComponentID) (L2Proposer, bool) {
	return GetComponent[L2Proposer](n, id)
}

// GetL2ChallengerByID returns an L2Challenger from a network by ID.
func GetL2ChallengerByID(n L2Network, id ComponentID) (L2Challenger, bool) {
	return GetComponent[L2Challenger](n, id)
}

// GetL2CLNodeByID returns an L2CLNode from a network by ID.
func GetL2CLNodeByID(n L2Network, id ComponentID) (L2CLNode, bool) {
	return GetComponent[L2CLNode](n, id)
}

// GetL2ELNodeByID returns an L2ELNode from a network by ID.
func GetL2ELNodeByID(n L2Network, id ComponentID) (L2ELNode, bool) {
	return GetComponent[L2ELNode](n, id)
}

// GetConductorByID returns a Conductor from a network by ID.
func GetConductorByID(n L2Network, id ComponentID) (Conductor, bool) {
	return GetComponent[Conductor](n, id)
}

// GetRollupBoostNodeByID returns a RollupBoostNode from a network by ID.
func GetRollupBoostNodeByID(n L2Network, id ComponentID) (RollupBoostNode, bool) {
	return GetComponent[RollupBoostNode](n, id)
}

// GetOPRBuilderNodeByID returns an OPRBuilderNode from a network by ID.
func GetOPRBuilderNodeByID(n L2Network, id ComponentID) (OPRBuilderNode, bool) {
	return GetComponent[OPRBuilderNode](n, id)
}

// --- Typed getter free functions for L1Network components ---

// GetL1ELNodeByID returns an L1ELNode from a network by ID.
func GetL1ELNodeByID(n L1Network, id ComponentID) (L1ELNode, bool) {
	return GetComponent[L1ELNode](n, id)
}

// GetL1CLNodeByID returns an L1CLNode from a network by ID.
func GetL1CLNodeByID(n L1Network, id ComponentID) (L1CLNode, bool) {
	return GetComponent[L1CLNode](n, id)
}

// --- Typed getter free functions for Network components (shared by L1 and L2) ---

// GetFaucetByID returns a Faucet from a network by ID.
func GetFaucetByID(n Network, id ComponentID) (Faucet, bool) {
	return GetComponent[Faucet](n, id)
}

// GetSyncTesterByID returns a SyncTester from a network by ID.
func GetSyncTesterByID(n Network, id ComponentID) (SyncTester, bool) {
	return GetComponent[SyncTester](n, id)
}

// --- Typed getter free functions for System components ---

// GetSuperchainByID returns a Superchain from a system by ID.
func GetSuperchainByID(s System, id ComponentID) (Superchain, bool) {
	return GetComponent[Superchain](s, id)
}

// GetClusterByID returns a Cluster from a system by ID.
func GetClusterByID(s System, id ComponentID) (Cluster, bool) {
	return GetComponent[Cluster](s, id)
}

// GetL1NetworkByID returns an L1Network from a system by ID.
func GetL1NetworkByID(s System, id ComponentID) (L1Network, bool) {
	return GetComponent[L1Network](s, id)
}

// GetL2NetworkByID returns an L2Network from a system by ID.
func GetL2NetworkByID(s System, id ComponentID) (L2Network, bool) {
	return GetComponent[L2Network](s, id)
}

// GetSupervisorByID returns a Supervisor from a system by ID.
func GetSupervisorByID(s System, id ComponentID) (Supervisor, bool) {
	return GetComponent[Supervisor](s, id)
}

// GetTestSequencerByID returns a TestSequencer from a system by ID.
func GetTestSequencerByID(s System, id ComponentID) (TestSequencer, bool) {
	return GetComponent[TestSequencer](s, id)
}

// --- List getter free functions ---

// GetL2Batchers returns all L2Batchers from a network.
func GetL2Batchers(n L2Network) []L2Batcher {
	return GetComponentsByKind[L2Batcher](n, KindL2Batcher)
}

// GetL2Proposers returns all L2Proposers from a network.
func GetL2Proposers(n L2Network) []L2Proposer {
	return GetComponentsByKind[L2Proposer](n, KindL2Proposer)
}

// GetL2Challengers returns all L2Challengers from a network.
func GetL2Challengers(n L2Network) []L2Challenger {
	return GetComponentsByKind[L2Challenger](n, KindL2Challenger)
}

// GetL2CLNodes returns all L2CLNodes from a network.
func GetL2CLNodes(n L2Network) []L2CLNode {
	return GetComponentsByKind[L2CLNode](n, KindL2CLNode)
}

// GetL2ELNodes returns all L2ELNodes from a network.
func GetL2ELNodes(n L2Network) []L2ELNode {
	return GetComponentsByKind[L2ELNode](n, KindL2ELNode)
}

// GetConductors returns all Conductors from a network.
func GetConductors(n L2Network) []Conductor {
	return GetComponentsByKind[Conductor](n, KindConductor)
}

// GetRollupBoostNodes returns all RollupBoostNodes from a network.
func GetRollupBoostNodes(n L2Network) []RollupBoostNode {
	return GetComponentsByKind[RollupBoostNode](n, KindRollupBoostNode)
}

// GetOPRBuilderNodes returns all OPRBuilderNodes from a network.
func GetOPRBuilderNodes(n L2Network) []OPRBuilderNode {
	return GetComponentsByKind[OPRBuilderNode](n, KindOPRBuilderNode)
}

// GetL1ELNodes returns all L1ELNodes from a network.
func GetL1ELNodes(n L1Network) []L1ELNode {
	return GetComponentsByKind[L1ELNode](n, KindL1ELNode)
}

// GetL1CLNodes returns all L1CLNodes from a network.
func GetL1CLNodes(n L1Network) []L1CLNode {
	return GetComponentsByKind[L1CLNode](n, KindL1CLNode)
}

// GetFaucets returns all Faucets from a network.
func GetFaucets(n Network) []Faucet {
	return GetComponentsByKind[Faucet](n, KindFaucet)
}

// GetSyncTesters returns all SyncTesters from a network.
func GetSyncTesters(n Network) []SyncTester {
	return GetComponentsByKind[SyncTester](n, KindSyncTester)
}

// GetSuperchains returns all Superchains from a system.
func GetSuperchains(s System) []Superchain {
	return GetComponentsByKind[Superchain](s, KindSuperchain)
}

// GetClusters returns all Clusters from a system.
func GetClusters(s System) []Cluster {
	return GetComponentsByKind[Cluster](s, KindCluster)
}

// GetL1Networks returns all L1Networks from a system.
func GetL1Networks(s System) []L1Network {
	return GetComponentsByKind[L1Network](s, KindL1Network)
}

// GetL2Networks returns all L2Networks from a system.
func GetL2Networks(s System) []L2Network {
	return GetComponentsByKind[L2Network](s, KindL2Network)
}

// GetSupervisors returns all Supervisors from a system.
func GetSupervisors(s System) []Supervisor {
	return GetComponentsByKind[Supervisor](s, KindSupervisor)
}

// GetTestSequencers returns all TestSequencers from a system.
func GetTestSequencers(s System) []TestSequencer {
	return GetComponentsByKind[TestSequencer](s, KindTestSequencer)
}
