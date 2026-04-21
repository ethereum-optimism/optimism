package superfaultproofs

import (
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/crypto"
)

// RunInteropActivationBoundaryTest verifies that the fault proof system
// correctly handles super-root transitions at the interop activation boundary.
// The agreed prestate is the super root at the activation timestamp and the
// disputed claim targets the very next block timestamp. This is the first
// super-root transition that occurs under the new interop rules.
//
// The system must be configured with a non-zero interop activation offset
// (via WithSuggestedInteropActivationOffset) so that early blocks are
// pre-interop and later blocks are post-interop.
func RunInteropActivationBoundaryTest(t devtest.T, sys *presets.SimpleInterop) {
	t.Require().NotNil(sys.SuperRoots, "supernode is required for this test")

	chains := orderedChains(sys)
	t.Require().Len(chains, 2, "expected exactly 2 interop chains")

	// Determine the interop activation timestamp from the rollup config.
	interopTime := chains[0].Cfg.InteropTime
	t.Require().NotNilf(interopTime, "interop fork must be scheduled")
	activationTimestamp := *interopTime
	t.Require().NotZero(activationTimestamp, "interop must not activate at genesis for this test")

	startTimestamp := activationTimestamp - 1
	endTimestamp := activationTimestamp
	t.Require().False(chains[0].Cfg.IsInterop(startTimestamp), "startTimestamp must not be interop-active")
	t.Require().True(chains[0].Cfg.IsInterop(endTimestamp), "endTimestamp must be interop-active")

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

	// Build the standard transition tests across the activation boundary.
	tests := []*transitionTest{
		{
			Name:               "FirstChainOptimisticBlock",
			AgreedClaim:        start.Marshal(),
			DisputedClaim:      step1,
			DisputedTraceIndex: 0,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
		{
			Name:               "SecondChainOptimisticBlock",
			AgreedClaim:        step1,
			DisputedClaim:      step2,
			DisputedTraceIndex: 1,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
		{
			Name:               "FirstPaddingStep",
			AgreedClaim:        step2,
			DisputedClaim:      padding(3),
			DisputedTraceIndex: 2,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
		{
			Name:               "Consolidate",
			AgreedClaim:        padding(consolidateStep),
			DisputedClaim:      end.Marshal(),
			DisputedTraceIndex: consolidateStep,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
	}

	challengerCfg := sys.L2ChainA.Escape().L2Challengers()[0].Config()
	gameDepth := sys.DisputeGameFactory().GameImpl(gameTypes.SuperCannonKonaGameType).SplitDepth()

	for _, test := range tests {
		t.Run(test.Name+"-fpp", func(t devtest.T) {
			runKonaInteropProgram(t, challengerCfg.CannonKona, test.L1Head.Hash,
				test.AgreedClaim, crypto.Keccak256Hash(test.DisputedClaim),
				test.ClaimTimestamp, test.ExpectValid)
		})
		t.Run(test.Name+"-challenger", func(t devtest.T) {
			runChallengerProviderTest(t, sys.SuperRoots.QueryAPI(), gameDepth, startTimestamp, test.ClaimTimestamp, test)
		})
	}
}
