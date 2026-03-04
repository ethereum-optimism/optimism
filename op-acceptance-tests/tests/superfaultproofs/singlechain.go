package superfaultproofs

import (
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/super"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/crypto"
)

// singleChain bundles the DSL handles for the single L2 chain in a SingleChainInterop system.
func singleChainFrom(sys *presets.SingleChainInterop) *chain {
	return &chain{
		ID:      sys.L2ChainA.ChainID(),
		Cfg:     sys.L2ChainA.Escape().RollupConfig(),
		Rollup:  sys.L2CLA.Escape().RollupAPI(),
		EL:      sys.L2ELA,
		CLNode:  sys.L2CLA,
		Batcher: sys.L2BatcherA,
	}
}

// RunSingleChainSuperFaultProofSmokeTest is a minimal smoke test for single-chain super fault proofs.
// It verifies that the super-root transition works correctly when the dependency set has only one chain.
// The test stops the batcher, waits for the safe head to stall, then resumes batching and verifies
// a basic set of valid/invalid transitions through both the FPP and challenger trace provider.
func RunSingleChainSuperFaultProofSmokeTest(t devtest.T, sys *presets.SingleChainInterop) {
	t.Require().NotNil(sys.SuperRoots, "supernode is required for this test")

	c := singleChainFrom(sys)
	chains := []*chain{c}

	// Stop batch submission so safe head stalls, then we have a known boundary.
	c.Batcher.Stop()
	t.Cleanup(c.Batcher.Start)
	awaitSafeHeadsStalled(t, sys.L2CLA)

	endTimestamp := nextTimestampAfterSafeHeads(t, chains)
	startTimestamp := endTimestamp - 1

	// Ensure the chain has produced the target block as unsafe.
	target, err := c.Cfg.TargetBlockNumber(endTimestamp)
	t.Require().NoError(err)
	c.EL.Reached(eth.Unsafe, target, 60)

	// L1 head where chain has no batch data at endTimestamp.
	respBefore := awaitOptimisticPattern(t, sys.SuperRoots, endTimestamp,
		nil, []eth.ChainID{c.ID})
	l1HeadBefore := respBefore.CurrentL1

	// Resume batching so the chain's data at endTimestamp becomes available.
	c.Batcher.Start()
	sys.SuperRoots.AwaitValidatedTimestamp(endTimestamp)
	l1HeadCurrent := latestRequiredL1(sys.SuperRoots.SuperRootAtTimestamp(endTimestamp))
	c.Batcher.Stop()

	// Build expected transition states for a single chain.
	start := superRootAtTimestamp(t, chains, startTimestamp)
	end := superRootAtTimestamp(t, chains, endTimestamp)

	optimistic := optimisticBlockAtTimestamp(t, c, endTimestamp)

	// With one chain: step 0 = chain's optimistic block, steps 1..consolidateStep-1 = padding,
	// consolidateStep = consolidation to next super root.
	step1 := marshalTransition(start, 1, optimistic)
	padding := func(step uint64) []byte {
		return marshalTransition(start, step, optimistic)
	}

	tests := []*transitionTest{
		{
			Name:               "ClaimDirectToNextTimestamp",
			AgreedClaim:        start.Marshal(),
			DisputedClaim:      end.Marshal(),
			DisputedTraceIndex: 0,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        false,
		},
		{
			Name:               "ChainOptimisticBlock",
			AgreedClaim:        start.Marshal(),
			DisputedClaim:      step1,
			DisputedTraceIndex: 0,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
		{
			Name:               "ChainOptimisticBlock-InvalidNoChange",
			AgreedClaim:        start.Marshal(),
			DisputedClaim:      start.Marshal(),
			DisputedTraceIndex: 0,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        false,
		},
		{
			Name:               "FirstPaddingStep",
			AgreedClaim:        step1,
			DisputedClaim:      padding(2),
			DisputedTraceIndex: 1,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
		{
			Name:               "ConsolidateStep",
			AgreedClaim:        padding(consolidateStep),
			DisputedClaim:      end.Marshal(),
			DisputedTraceIndex: consolidateStep,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
		{
			Name:               "ConsolidateStep-InvalidNoChange",
			AgreedClaim:        padding(consolidateStep),
			DisputedClaim:      padding(consolidateStep),
			DisputedTraceIndex: consolidateStep,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        false,
		},
		{
			Name:               "ChainReachesL1Head",
			AgreedClaim:        start.Marshal(),
			DisputedClaim:      super.InvalidTransition,
			DisputedTraceIndex: 0,
			L1Head:             l1HeadBefore,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
		{
			Name:               "SuperRootInvalidIfUnsupportedByL1Data",
			AgreedClaim:        start.Marshal(),
			DisputedClaim:      step1,
			DisputedTraceIndex: 0,
			L1Head:             l1HeadBefore,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        false,
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
