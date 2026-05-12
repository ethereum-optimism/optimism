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

// RunSuperrootOptimisticPairingTest exercises the OPTIMISTIC branch of the
// super-root transition when chain A has a local-safe block at endTimestamp
// whose exec message references a fabricated log in chain B's would-be
// at-endTimestamp block.
//
// Test sequencing builds an explicit L1 gap between chain A's batch and chain
// B's batch so the supernode can clearly distinguish them. See
// runOptimisticPairingTest for the full step-by-step setup.
func RunSuperrootOptimisticPairingTest(t devtest.T, sys *presets.SimpleInterop) {
	runOptimisticPairingTest(t, sys, true)
}

// RunSuperrootOptimisticPairingNoReplacementTest is the same scenario as
// RunSuperrootOptimisticPairingTest except chain B's at-endTimestamp block is
// never built or batched. Chain A's invalid block therefore stays local-safe
// at endTimestamp forever and no replacement occurs.
func RunSuperrootOptimisticPairingNoReplacementTest(t devtest.T, sys *presets.SimpleInterop) {
	runOptimisticPairingTest(t, sys, false)
}

// l1GapBetweenChainBatches is the number of additional L1 blocks the test
// waits after chain A's batch lands and before chain B's batch lands. The
// gap ensures the supernode records distinct L1 inclusion blocks for each
// chain's batch — without a gap the supernode greedily promotes both at the
// same L1 and the timing window for ethereum-optimism/optimism#20657 closes.
const l1GapBetweenChainBatches = 3

func runOptimisticPairingTest(t devtest.T, sys *presets.SimpleInterop, withReplacement bool) {
	t.Require().NotNil(sys.SuperRoots, "supernode is required for this test")
	t.Require().NotNil(sys.TestSequencer, "test sequencer is required for controlled block building")

	chains := orderedChains(sys)
	t.Require().Len(chains, 2, "expected exactly 2 interop chains")
	t.Require().Equal(chains[0].ID, sys.L2ChainA.ChainID(), "expected chain A as chains[0]")
	t.Require().Equal(chains[1].ID, sys.L2ChainB.ChainID(), "expected chain B as chains[1]")

	// --- Fund EOAs and deploy chain B's event logger while sequencers are
	// still live. The fabricated invalid exec message references this address
	// as Origin; the log itself will never exist.
	aliceA := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)
	eventLoggerB := bob.DeployEventLogger()

	// --- Freeze chains: only TestSequencer advances unsafe, only explicit
	// Batcher.Start/Stop advances local-safe.
	freezeChains(chains)

	// Drive both chains to startTimestamp and batch so both are local-safe
	// AND (after consolidation) cross-safe at startTimestamp.
	endTimestamp := nextTimestampAfterSafeHeads(t, chains)
	startTimestamp := endTimestamp - 1
	advanceUnsafeToTimestamp(t, sys, chains, startTimestamp)

	l1HeadBeforeAnyBatch := sys.L1EL.BlockRefByLabel(eth.Unsafe).ID()
	advanceSafeToCurrentUnsafe(t, chains[0])
	l1HeadAfterPreSafeABatch := sys.L1EL.BlockRefByLabel(eth.Unsafe).ID()
	advanceSafeToCurrentUnsafe(t, chains[1])
	l1HeadAfterPreSafeBBatch := sys.L1EL.BlockRefByLabel(eth.Unsafe).ID()

	sys.SuperRoots.AwaitValidatedTimestamp(startTimestamp)

	// --- Build chain A's at-endTimestamp block with a fabricated invalid
	// exec message. nextTimestampAfterSafeHeads guarantees each chain's next
	// scheduled block from here lands at endTimestamp.
	unsafeA := sys.L2ELA.BlockRefByLabel(eth.Unsafe)
	unsafeB := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	t.Require().Equalf(endTimestamp, unsafeA.Time+chains[0].Cfg.BlockTime,
		"chain A's next scheduled block must land at endTimestamp %d (head time %d, blockTime %d)",
		endTimestamp, unsafeA.Time, chains[0].Cfg.BlockTime)
	t.Require().Equalf(endTimestamp, unsafeB.Time+chains[1].Cfg.BlockTime,
		"chain B's next scheduled block must land at endTimestamp %d (head time %d, blockTime %d)",
		endTimestamp, unsafeB.Time, chains[1].Cfg.BlockTime)
	expectedBlockNumB := unsafeB.Number + 1

	topic := crypto.Keccak256Hash([]byte("DataEmitted(bytes)"))
	msgHash := crypto.Keccak256Hash([]byte("optimistic pairing fabricated msg"))
	fabricatedPayload := make([]byte, 0, 64)
	fabricatedPayload = append(fabricatedPayload, topic.Bytes()...)
	fabricatedPayload = append(fabricatedPayload, msgHash.Bytes()...)

	fabricatedMsg := suptypes.Message{
		Identifier: suptypes.Identifier{
			Origin:      eventLoggerB,
			BlockNumber: expectedBlockNumB,
			LogIndex:    0,
			Timestamp:   endTimestamp,
			ChainID:     chains[1].ID,
		},
		PayloadHash: crypto.Keccak256Hash(fabricatedPayload),
	}

	execTx := dsl.SubmitExecForMessage(fabricatedMsg, aliceA)
	txplan.WithStaticNonce(aliceA.PendingNonce())(execTx)
	signedTx, err := execTx.Signed.Eval(t.Ctx())
	t.Require().NoError(err, "failed to sign chain A invalid exec tx")
	rawExecTx, err := signedTx.MarshalBinary()
	t.Require().NoError(err, "failed to marshal chain A invalid exec tx")

	sys.TestSequencer.SequenceBlockWithTxs(t, chains[0].ID, unsafeA.Hash, [][]byte{rawExecTx})
	newHeadA := sys.L2ELA.BlockRefByLabel(eth.Unsafe)
	t.Require().Equalf(endTimestamp, newHeadA.Time,
		"chain A's invalid exec block must land at endTimestamp %d, got %d",
		endTimestamp, newHeadA.Time)

	// Batch chain A's at-endTimestamp block — A is now local-safe at endTimestamp.
	l1HeadBeforeABatch := sys.L1EL.BlockRefByLabel(eth.Unsafe).ID()
	advanceSafeToCurrentUnsafe(t, chains[0])
	l1HeadAfterABatch := sys.L1EL.BlockRefByLabel(eth.Unsafe).ID()

	// Force an L1 gap so chain A's batch L1 is strictly less than chain B's.
	for i := 0; i < l1GapBetweenChainBatches; i++ {
		sys.L1Network.WaitForBlock()
	}

	// Capture game L1 head at the START of the gap window: any chain whose batch
	// landed at-or-before this L1 should be derivable optimistically at gameL1Head.
	gameL1Head := l1HeadAfterABatch

	t.Logf("[optimistic-pairing] L1 heads during setup:"+
		" before-any-batch=%d, after-preSafe-A=%d, after-preSafe-B=%d,"+
		" before-A-at-end=%d, after-A-at-end=%d, gameL1Head=%d",
		l1HeadBeforeAnyBatch.Number,
		l1HeadAfterPreSafeABatch.Number,
		l1HeadAfterPreSafeBBatch.Number,
		l1HeadBeforeABatch.Number,
		l1HeadAfterABatch.Number,
		gameL1Head.Number,
	)

	var l1HeadBeforeBBatch, l1HeadAfterBBatch eth.BlockID
	if withReplacement {
		// Build chain B's empty at-endTimestamp block and batch it. By
		// construction the batch lands at an L1 height strictly greater than
		// gameL1Head (we waited l1GapBetweenChainBatches L1 blocks first).
		sys.TestSequencer.SequenceBlock(t, chains[1].ID, unsafeB.Hash)
		newHeadB := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
		t.Require().Equalf(endTimestamp, newHeadB.Time,
			"chain B's at-endTimestamp block must land at endTimestamp %d, got %d",
			endTimestamp, newHeadB.Time)
		l1HeadBeforeBBatch = sys.L1EL.BlockRefByLabel(eth.Unsafe).ID()
		advanceSafeToCurrentUnsafe(t, chains[1])
		l1HeadAfterBBatch = sys.L1EL.BlockRefByLabel(eth.Unsafe).ID()

		t.Logf("[optimistic-pairing] chain B batch L1 heads: before=%d after=%d",
			l1HeadBeforeBBatch.Number, l1HeadAfterBBatch.Number)
		t.Require().Greaterf(l1HeadBeforeBBatch.Number, gameL1Head.Number,
			"chain B's batch must land strictly after gameL1Head=%d (was before-B=%d)",
			gameL1Head.Number, l1HeadBeforeBBatch.Number)

		// Wait for chain A's at-endTimestamp position to reach cross-safe via
		// replacement.
		sys.L2CLA.Reached(suptypes.CrossSafe, newHeadA.Number, 60)
	}

	// --- Setup diagnostics: report what the supernode currently sees for both
	// timestamps and what L1 it ties each chain's optimistic output to.
	prevRoot := sys.SuperRoots.SuperRootAtTimestamp(startTimestamp)
	endRoot := sys.SuperRoots.SuperRootAtTimestamp(endTimestamp)
	logSuperRoot(t, "startTimestamp", startTimestamp, prevRoot, chains)
	logSuperRoot(t, "endTimestamp", endTimestamp, endRoot, chains)

	// Pre-condition assertion: prev super root must be fully verifiable at
	// gameL1Head — otherwise the test setup itself is broken.
	t.Require().NotNil(prevRoot.Data, "super root at startTimestamp must have data")
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

	runPairingTransitionTests(t, sys, tests, startTimestamp)
}

// logSuperRoot prints a compact summary of what the supernode returned for
// SuperRootAtTimestamp(ts) — current L1, verified-required L1, and each
// chain's optimistic-output RequiredL1.
func logSuperRoot(t devtest.T, label string, ts uint64, resp eth.SuperRootAtTimestampResponse, chains []*chain) {
	t.Logf("[optimistic-pairing] superroot_atTimestamp(%s=%d): CurrentL1=%d CurrentSafeTimestamp=%d CurrentLocalSafeTimestamp=%d",
		label, ts, resp.CurrentL1.Number, resp.CurrentSafeTimestamp, resp.CurrentLocalSafeTimestamp)
	if resp.Data != nil {
		t.Logf("[optimistic-pairing] superroot_atTimestamp(%s).Data: VerifiedRequiredL1=%d SuperRoot=%s",
			label, resp.Data.VerifiedRequiredL1.Number, resp.Data.SuperRoot)
	} else {
		t.Logf("[optimistic-pairing] superroot_atTimestamp(%s).Data is nil (some chain has no verified block here)", label)
	}
	for _, c := range chains {
		out, ok := resp.OptimisticAtTimestamp[c.ID]
		if !ok {
			t.Logf("[optimistic-pairing] superroot_atTimestamp(%s) OptimisticAtTimestamp[chain %s]: MISSING", label, c.ID)
			continue
		}
		t.Logf("[optimistic-pairing] superroot_atTimestamp(%s) OptimisticAtTimestamp[chain %s]: RequiredL1=%d OutputRoot=%s BlockHash=%s",
			label, c.ID, out.RequiredL1.Number, out.OutputRoot, out.Output.BlockHash)
	}
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
