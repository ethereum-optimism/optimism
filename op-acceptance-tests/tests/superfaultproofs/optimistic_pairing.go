package superfaultproofs

import (
	"math/rand"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/super"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// RunSuperrootOptimisticPairingTest invalidates chain A's block at T and pins
// the game L1 head between chain A's original-batch L1 inclusion and chain B's
// at-T batch L1 inclusion. Kona derives chain A's original optimistic block
// from the L1 batches at this L1 head; op-challenger should agree.
//
// The bug it exposes (ethereum-optimism/optimism#20657): the supernode's
// OptimisticAtTimestamp[A].RequiredL1 reflects the L1 at which the replacement
// was cross-safe-promoted, not the L1 at which the original block was
// local-safe. That cross-safe-promotion L1 is strictly later than the game L1
// head here, so the trace provider short-circuits to InvalidTransition while
// kona produces the proper transition.
func RunSuperrootOptimisticPairingTest(t devtest.T, sys *presets.SimpleInterop) {
	t.Require().NotNil(sys.SuperRoots, "supernode is required for this test")
	rng := rand.New(rand.NewSource(20657))

	chains := orderedChains(sys)
	t.Require().Len(chains, 2, "expected exactly 2 interop chains")
	t.Require().Equal(chains[0].ID, sys.L2ChainA.ChainID(), "expected chain A as chains[0]")
	t.Require().Equal(chains[1].ID, sys.L2ChainB.ChainID(), "expected chain B as chains[1]")

	aliceA := sys.FunderA.NewFundedEOA(eth.OneEther)
	aliceB := sys.FunderB.NewFundedEOA(eth.OneEther)
	eventLoggerB := aliceB.DeployEventLogger()

	// Hold chain B's at-T batch until chain A's has landed, so the original
	// chain A batch reaches L1 strictly before chain B's.
	sys.L2BatcherB.Stop()

	initMsg := aliceB.SendRandomInitMessage(rng, eventLoggerB, 2, 10)
	execMsg := aliceA.SendInvalidExecMessage(initMsg)

	execBlockNumA := bigs.Uint64Strict(execMsg.BlockNumber())
	endTimestamp := sys.L2ChainA.TimestampForBlockNum(execBlockNumA)
	startTimestamp := endTimestamp - 1

	sys.L2CLA.Reached(types.LocalSafe, execBlockNumA, 30)
	sys.L2BatcherB.Start()

	sys.SuperRoots.AwaitValidatedTimestamp(endTimestamp)
	sys.L2CLA.Reached(types.CrossSafe, execBlockNumA, 30)
	sys.L2ELA.AssertExecMessageNotInBlock(execMsg)

	// L1 head one below chain B's at-T batch inclusion: chain A's at-T block
	// (original or replacement) is derivable, chain B's is not.
	chainBRequired := sys.SuperRoots.SuperRootAtTimestamp(endTimestamp).
		OptimisticAtTimestamp[chains[1].ID].RequiredL1
	gameL1Head := sys.L1EL.BlockRefByNumber(chainBRequired.Number - 1).ID()

	firstOptimistic := optimisticBlockAtTimestamp(t, sys.SuperRoots.QueryAPI(), chains[0].ID, endTimestamp)

	start := superRootAtTimestamp(t, chains, startTimestamp)
	step1Trace := marshalTransition(start, 1, firstOptimistic)

	tests := []*transitionTest{
		{
			Name:               "FirstChainOptimisticBlock",
			AgreedClaim:        start.Marshal(),
			DisputedClaim:      step1Trace,
			DisputedTraceIndex: 0,
			L1Head:             gameL1Head,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
		{
			Name:               "SecondChainOptimisticBlock-Invalid",
			AgreedClaim:        step1Trace,
			DisputedClaim:      super.InvalidTransition,
			DisputedTraceIndex: 1,
			L1Head:             gameL1Head,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
	}

	runPairingTransitionTests(t, sys, tests, startTimestamp)
}

// RunSuperrootOptimisticPairingNoReplacementTest is the same scenario as
// RunSuperrootOptimisticPairingTest but chain B's batcher is never restarted,
// so chain B's at-T batch never lands on L1 and interop validation cannot
// proceed. No replacement is generated; chain A's invalid block stays local-
// safe at T. The game L1 head is pinned one L1 block past where chain A
// reached local-safe.
//
// Same bug (ethereum-optimism/optimism#20657), independent of replacement:
// because the supernode's safeDB only records cross-safe-promotion L1s,
// chain A's optimistic block is never recorded with its local-safe L1, and
// RequiredL1 ends up above the game L1 head — so the trace provider returns
// InvalidTransition where kona builds the proper transition.
func RunSuperrootOptimisticPairingNoReplacementTest(t devtest.T, sys *presets.SimpleInterop) {
	t.Require().NotNil(sys.SuperRoots, "supernode is required for this test")
	rng := rand.New(rand.NewSource(20657))

	chains := orderedChains(sys)
	t.Require().Len(chains, 2, "expected exactly 2 interop chains")
	t.Require().Equal(chains[0].ID, sys.L2ChainA.ChainID(), "expected chain A as chains[0]")
	t.Require().Equal(chains[1].ID, sys.L2ChainB.ChainID(), "expected chain B as chains[1]")

	aliceA := sys.FunderA.NewFundedEOA(eth.OneEther)
	aliceB := sys.FunderB.NewFundedEOA(eth.OneEther)
	eventLoggerB := aliceB.DeployEventLogger()

	sys.L2BatcherB.Stop()

	initMsg := aliceB.SendRandomInitMessage(rng, eventLoggerB, 2, 10)
	execMsg := aliceA.SendInvalidExecMessage(initMsg)

	execBlockNumA := bigs.Uint64Strict(execMsg.BlockNumber())
	endTimestamp := sys.L2ChainA.TimestampForBlockNum(execBlockNumA)
	startTimestamp := endTimestamp - 1

	sys.L2CLA.Reached(types.LocalSafe, execBlockNumA, 30)

	// Game head one L1 block past where chain A's invalid block became safe;
	// chain B's batch never lands so its at-T block is not derivable.
	gameL1Head := sys.L1Network.WaitForBlock().ID()

	firstOptimistic := optimisticBlockAtTimestamp(t, sys.SuperRoots.QueryAPI(), chains[0].ID, endTimestamp)

	start := superRootAtTimestamp(t, chains, startTimestamp)
	step1Trace := marshalTransition(start, 1, firstOptimistic)

	tests := []*transitionTest{
		{
			Name:               "FirstChainOptimisticBlock",
			AgreedClaim:        start.Marshal(),
			DisputedClaim:      step1Trace,
			DisputedTraceIndex: 0,
			L1Head:             gameL1Head,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
		{
			Name:               "SecondChainOptimisticBlock-Invalid",
			AgreedClaim:        step1Trace,
			DisputedClaim:      super.InvalidTransition,
			DisputedTraceIndex: 1,
			L1Head:             gameL1Head,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
	}

	runPairingTransitionTests(t, sys, tests, startTimestamp)
}

func runPairingTransitionTests(t devtest.T, sys *presets.SimpleInterop, tests []*transitionTest, startTimestamp uint64) {
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
