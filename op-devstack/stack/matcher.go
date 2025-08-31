package stack

type Identifiable[I comparable] interface {
	ID() I
}

// Matcher abstracts what can be used as getter-method argument.
// All ID types implement this interface, and lookup functions check
// if the argument is an ID before searching for a match.
// This enables lookups such as getting a component by labels,
// by its state, by its relation to other components, etc.
type Matcher[I comparable, E Identifiable[I]] interface {
	// Match finds the elements that pass the matcher.
	// If no element passes, it returns an empty slice.
	// Callers should guarantee a stable order of ids, to ensure a deterministic match.
	Match(elems []E) []E

	// String must describe the matcher for debugging purposes.
	// This does not get used for matching.
	String() string
}

func findByID[I comparable, E Identifiable[I]](id I, elems []E) []E {
	for i, elem := range elems {
		if elem.ID() == id {
			return elems[i : i+1]
		}
	}
	return nil
}

type ClusterMatcher = Matcher[ClusterID, Cluster]

type L1CLMatcher = Matcher[L1CLNodeID, L1CLNode]

type L1ELMatcher = Matcher[L1ELNodeID, L1ELNode]

type L1NetworkMatcher = Matcher[L1NetworkID, L1Network]

type L2NetworkMatcher = Matcher[L2NetworkID, L2Network]

type SuperchainMatcher = Matcher[SuperchainID, Superchain]

type L2BatcherMatcher = Matcher[L2BatcherID, L2Batcher]

type L2ChallengerMatcher = Matcher[L2ChallengerID, L2Challenger]

type L2ProposerMatcher = Matcher[L2ProposerID, L2Proposer]

type L2CLMatcher = Matcher[L2CLNodeID, L2CLNode]

type SupervisorMatcher = Matcher[SupervisorID, Supervisor]

type TestSequencerMatcher = Matcher[TestSequencerID, TestSequencer]

type ConductorMatcher = Matcher[ConductorID, Conductor]

type FlashblocksBuilderMatcher = Matcher[FlashblocksBuilderID, FlashblocksBuilderNode]

type L2ELMatcher = Matcher[L2ELNodeID, L2ELNode]

type FaucetMatcher = Matcher[FaucetID, Faucet]

type SyncTesterMatcher = Matcher[SyncTesterID, SyncTester]
