package stack

import "github.com/ethereum-optimism/optimism/op-service/eth"

type ComponentIdentifiable interface {
	ID() ComponentID
}

// ComponentIDMatcher abstracts what can be used as getter-method argument.
// All ID types implement this interface, and lookup functions check
// if the argument is an ID before searching for a match.
// This enables lookups such as getting a component by labels,
// by its state, by its relation to other components, etc.
type ComponentIDMatcher[E ComponentIdentifiable] interface {
	// Match finds the elements that pass the matcher.
	// If no element passes, it returns an empty slice.
	// Callers should guarantee a stable order of ids, to ensure a deterministic match.
	Match(elems []E) []E

	// String must describe the matcher for debugging purposes.
	// This does not get used for matching.
	String() string
}

type ChainIdentifiable interface {
	ID() eth.ChainID
}

// Matcher abstracts what can be used as getter-method argument.
// All ID types implement this interface, and lookup functions check
// if the argument is an ID before searching for a match.
// This enables lookups such as getting a component by labels,
// by its state, by its relation to other components, etc.
type ChainIDMatcher[E ChainIdentifiable] interface {
	// Match finds the elements that pass the matcher.
	// If no element passes, it returns an empty slice.
	// Callers should guarantee a stable order of ids, to ensure a deterministic match.
	Match(elems []E) []E

	// String must describe the matcher for debugging purposes.
	// This does not get used for matching.
	String() string
}

func findByChainID[E ChainIdentifiable](id eth.ChainID, elems []E) []E {
	for i, elem := range elems {
		if elem.ID() == id {
			return elems[i : i+1]
		}
	}
	return nil
}

type L1CLMatcher = ComponentIDMatcher[L1CLNode]

type L1ELMatcher = ComponentIDMatcher[L1ELNode]

type L1NetworkMatcher = ChainIDMatcher[L1Network]

type L2NetworkMatcher = ChainIDMatcher[L2Network]

type SuperchainMatcher = ComponentIDMatcher[Superchain]

type L2BatcherMatcher = ComponentIDMatcher[L2Batcher]

type L2ChallengerMatcher = ComponentIDMatcher[L2Challenger]

type L2ProposerMatcher = ComponentIDMatcher[L2Proposer]

type L2CLMatcher = ComponentIDMatcher[L2CLNode]

type SupervisorMatcher = ComponentIDMatcher[Supervisor]

type TestSequencerMatcher = ComponentIDMatcher[TestSequencer]

type ConductorMatcher = ComponentIDMatcher[Conductor]

type L2ELMatcher = ComponentIDMatcher[L2ELNode]

type FaucetMatcher = ComponentIDMatcher[Faucet]

type SyncTesterMatcher = ComponentIDMatcher[SyncTester]

type RollupBoostNodeMatcher = ComponentIDMatcher[RollupBoostNode]

type OPRBuilderNodeMatcher = ComponentIDMatcher[OPRBuilderNode]
