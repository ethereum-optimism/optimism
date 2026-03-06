package stack

// Identifiable is implemented by all components that have an ID.
type Identifiable interface {
	ID() ComponentID
}

// Matcher abstracts what can be used as getter-method argument.
// All ID types implement this interface, and lookup functions check
// if the argument is an ID before searching for a match.
// This enables lookups such as getting a component by labels,
// by its state, by its relation to other components, etc.
type Matcher[E Identifiable] interface {
	// Match finds the elements that pass the matcher.
	// If no element passes, it returns an empty slice.
	// Callers should guarantee a stable order of ids, to ensure a deterministic match.
	Match(elems []E) []E

	// String must describe the matcher for debugging purposes.
	// This does not get used for matching.
	String() string
}

type ClusterMatcher = Matcher[Cluster]

type L1CLMatcher = Matcher[L1CLNode]

type L1ELMatcher = Matcher[L1ELNode]

type L1NetworkMatcher = Matcher[L1Network]

type L2NetworkMatcher = Matcher[L2Network]

type SuperchainMatcher = Matcher[Superchain]

type L2BatcherMatcher = Matcher[L2Batcher]

type L2ChallengerMatcher = Matcher[L2Challenger]

type L2ProposerMatcher = Matcher[L2Proposer]

type L2CLMatcher = Matcher[L2CLNode]

type SupervisorMatcher = Matcher[Supervisor]

type SupernodeMatcher = Matcher[Supernode]

type TestSequencerMatcher = Matcher[TestSequencer]

type ConductorMatcher = Matcher[Conductor]

type L2ELMatcher = Matcher[L2ELNode]

type FaucetMatcher = Matcher[Faucet]

type SyncTesterMatcher = Matcher[SyncTester]

type RollupBoostNodeMatcher = Matcher[RollupBoostNode]

type OPRBuilderNodeMatcher = Matcher[OPRBuilderNode]
