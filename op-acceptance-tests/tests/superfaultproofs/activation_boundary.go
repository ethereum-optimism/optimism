package superfaultproofs

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// RunInteropActivationBoundaryTest verifies that the fault proof system correctly handles
// super-root transitions around the interop activation boundary.
//
// It proves two consecutive transitions:
//
//   - Activation — the first super-root transition under the new interop rules, whose blocks carry
//     the activation deposits and the one-time upgrade gas.
//   - PostActivation — the transition immediately after it, whose blocks must have reverted to the
//     steady-state gas limit. Proving it is what pins the upgrade gas to the activation block; a
//     leak would change the post-activation block hash and the claim would no longer validate.
//
// The system must be configured with a non-zero interop activation offset (via
// WithSuggestedInteropActivationOffset) so that early blocks are pre-interop and later blocks are
// post-interop.
func RunInteropActivationBoundaryTest(t devtest.T, sys *presets.SimpleInterop, runners ...ProofRunner) {
	t.Require().NotNil(sys.SuperRoots, "supernode is required for this test")

	chains := orderedChains(sys)
	t.Require().Len(chains, 2, "expected exactly 2 interop chains")

	// Determine the Lagoon (interop activation) timestamp from the rollup config.
	interopTime := chains[0].Cfg.LagoonTime
	t.Require().NotNilf(interopTime, "Lagoon (interop activation) fork must be scheduled")
	activationTimestamp := *interopTime
	t.Require().NotZero(activationTimestamp, "Lagoon must not activate at genesis for this test")

	t.Require().False(chains[0].Cfg.IsInterop(activationTimestamp-1), "the block before activation must not be interop-active")
	t.Require().True(chains[0].Cfg.IsInterop(activationTimestamp), "the activation block must be interop-active")

	runBoundarySpan(t, sys, chains, "Activation", activationTimestamp, runners)

	// The next timestamp at which both chains produce a block — the first block after activation.
	postActivation := advanceToCommonBlockBoundary(t, chains, activationTimestamp+1)
	for _, c := range chains {
		activationBlock, err := c.Cfg.TargetBlockNumber(activationTimestamp)
		t.Require().NoError(err)
		postBlock, err := c.Cfg.TargetBlockNumber(postActivation)
		t.Require().NoError(err)
		t.Require().Greaterf(postBlock, activationBlock,
			"chain %s: post-activation span must derive a block after the activation block", c.ID)
	}
	runBoundarySpan(t, sys, chains, "PostActivation", postActivation, runners)
}

// runBoundarySpan proves the single super-root transition that lands on endTimestamp, i.e. the
// transition that advances every chain by the one block produced at that timestamp. namePrefix
// distinguishes the resulting subtests when a test proves more than one span.
func runBoundarySpan(
	t devtest.T,
	sys *presets.SimpleInterop,
	chains []*chain,
	namePrefix string,
	endTimestamp uint64,
	runners []ProofRunner,
) {
	startTimestamp := endTimestamp - 1

	// Wait for chains to produce blocks past the end timestamp.
	for _, c := range chains {
		target, err := c.Cfg.TargetBlockNumber(endTimestamp)
		t.Require().NoError(err)
		c.EL.Reached(eth.Unsafe, target, 60)
	}

	// Wait for supernode to validate the end timestamp.
	sys.SuperRoots.AwaitValidatedTimestamp(endTimestamp)
	l1HeadCurrent := latestRequiredL1(sys.SuperRoots.SuperRootAtTimestamp(endTimestamp))

	start := superRootAtTimestamp(t, chains, startTimestamp)
	end := superRootAtTimestamp(t, chains, endTimestamp)

	firstOptimistic := optimisticBlockAtTimestamp(t, sys.SuperRoots.QueryAPI(), chains[0].ID, endTimestamp)
	secondOptimistic := optimisticBlockAtTimestamp(t, sys.SuperRoots.QueryAPI(), chains[1].ID, endTimestamp)

	step1 := marshalTransition(start, 1, firstOptimistic)
	step2 := marshalTransition(start, 2, firstOptimistic, secondOptimistic)
	padding := func(step uint64) []byte {
		return marshalTransition(start, step, firstOptimistic, secondOptimistic)
	}

	tests := []*transitionTest{
		{
			Name:               namePrefix + "FirstChainOptimisticBlock",
			AgreedClaim:        start.Marshal(),
			DisputedClaim:      step1,
			DisputedTraceIndex: 0,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
		{
			Name:               namePrefix + "SecondChainOptimisticBlock",
			AgreedClaim:        step1,
			DisputedClaim:      step2,
			DisputedTraceIndex: 1,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
		{
			Name:               namePrefix + "FirstPaddingStep",
			AgreedClaim:        step2,
			DisputedClaim:      padding(3),
			DisputedTraceIndex: 2,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
		{
			Name:               namePrefix + "Consolidate",
			AgreedClaim:        padding(consolidateStep),
			DisputedClaim:      end.Marshal(),
			DisputedTraceIndex: consolidateStep,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
	}

	runScenarioProofs(t, sys, &scenarioProofData{
		fpvmTransitions:    tests,
		fpvmStartTimestamp: startTimestamp,
		zkCheckpoint:       newZKCheckpointForRunners(t, sys, endTimestamp, false, runners),
	}, runners...)
}
