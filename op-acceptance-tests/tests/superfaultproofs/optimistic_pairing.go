package superfaultproofs

import (
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/super"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// RunOptimisticPairingTest exercises the optimistic branch of a super-root
// transition when chain A has an invalid-exec block at endTimestamp whose
// exec message references a fabricated log at chain B's would-be
// at-endTimestamp block.
//
// If withReplacement is true, chain B's at-endTimestamp block is built
// (empty) and batched after the game L1 head, causing chain A's invalid
// block to be replaced via cross-validation. Otherwise chain B's
// at-endTimestamp block is never built and chain A's invalid block stays
// local-safe at endTimestamp forever.
func RunOptimisticPairingTest(t devtest.T, sys *presets.SimpleInterop, withReplacement bool) {
	t.Require().NotNil(sys.SuperRoots, "supernode is required")
	t.Require().NotNil(sys.TestSequencer, "test sequencer is required")

	chains := orderedChains(sys)
	t.Require().Len(chains, 2)
	t.Require().Equal(chains[0].ID, sys.L2ChainA.ChainID())
	t.Require().Equal(chains[1].ID, sys.L2ChainB.ChainID())

	// EOAs and event logger must be funded/deployed before freezeChains stops
	// the sequencers. The event logger address only fills the fabricated
	// message's Origin field; the log itself never exists.
	aliceA := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)
	eventLoggerB := bob.DeployEventLogger()

	freezeChains(chains)

	// Drive both chains to startTimestamp and batch so they're cross-safe at X.
	endTimestamp := nextTimestampAfterSafeHeads(t, chains)
	startTimestamp := endTimestamp - 1
	advanceUnsafeToTimestamp(t, sys, chains, startTimestamp)
	advanceSafeToCurrentUnsafe(t, chains[0])
	advanceSafeToCurrentUnsafe(t, chains[1])
	sys.SuperRoots.AwaitValidatedTimestamp(startTimestamp)

	// Build chain A's at-endTimestamp block with a fabricated invalid exec
	// message referencing a log at (chain B, expectedBlockNumB, logIndex 0)
	// that will never exist.
	unsafeA := sys.L2ELA.BlockRefByLabel(eth.Unsafe)
	unsafeB := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	t.Require().Equalf(endTimestamp, unsafeA.Time+chains[0].Cfg.BlockTime,
		"chain A's next scheduled block must land at endTimestamp %d (head time %d, blockTime %d)",
		endTimestamp, unsafeA.Time, chains[0].Cfg.BlockTime)
	t.Require().Equalf(endTimestamp, unsafeB.Time+chains[1].Cfg.BlockTime,
		"chain B's next scheduled block must land at endTimestamp %d (head time %d, blockTime %d)",
		endTimestamp, unsafeB.Time, chains[1].Cfg.BlockTime)

	topic := crypto.Keccak256Hash([]byte("DataEmitted(bytes)"))
	msgHash := crypto.Keccak256Hash([]byte("optimistic pairing fabricated msg"))
	fabricatedPayload := append(append(make([]byte, 0, 64), topic.Bytes()...), msgHash.Bytes()...)
	fabricatedMsg := suptypes.Message{
		Identifier: suptypes.Identifier{
			Origin:      eventLoggerB,
			BlockNumber: unsafeB.Number + 1,
			LogIndex:    0,
			Timestamp:   endTimestamp,
			ChainID:     chains[1].ID,
		},
		PayloadHash: crypto.Keccak256Hash(fabricatedPayload),
	}

	execTx := dsl.SubmitExecForMessage(fabricatedMsg, aliceA)
	txplan.WithStaticNonce(aliceA.PendingNonce())(execTx)
	signedTx, err := execTx.Signed.Eval(t.Ctx())
	t.Require().NoError(err)
	rawExecTx, err := signedTx.MarshalBinary()
	t.Require().NoError(err)

	sys.TestSequencer.SequenceBlockWithTxs(t, chains[0].ID, unsafeA.Hash, [][]byte{rawExecTx})
	newHeadA := sys.L2ELA.BlockRefByLabel(eth.Unsafe)
	t.Require().Equal(endTimestamp, newHeadA.Time)

	advanceSafeToCurrentUnsafe(t, chains[0])
	gameL1Head := sys.L1EL.BlockRefByLabel(eth.Unsafe).ID()

	// One L1 block is enough separation: chain B's batch will land strictly
	// after gameL1Head.
	sys.L1Network.WaitForBlock()

	if withReplacement {
		sys.TestSequencer.SequenceBlock(t, chains[1].ID, unsafeB.Hash)
		t.Require().Equal(endTimestamp, sys.L2ELB.BlockRefByLabel(eth.Unsafe).Time)
		advanceSafeToCurrentUnsafe(t, chains[1])
		sys.L2CLA.Reached(suptypes.CrossSafe, newHeadA.Number, 60)
	}

	// The super root at startTimestamp must be fully verifiable at gameL1Head;
	// otherwise the setup itself is broken.
	prevRoot := sys.SuperRoots.SuperRootAtTimestamp(startTimestamp)
	t.Require().LessOrEqualf(prevRoot.Data.VerifiedRequiredL1.Number, gameL1Head.Number,
		"prev super root VerifiedRequiredL1 %d must be <= gameL1Head %d",
		prevRoot.Data.VerifiedRequiredL1.Number, gameL1Head.Number)

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
