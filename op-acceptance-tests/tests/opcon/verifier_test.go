package opcon

import (
	"testing"
	"time"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
)

// TestOpConVerifierDerivesSafeHead exercises a consensus-only verifier that
// derives the safe L2 chain purely from L1, with no CL P2P and no follow
// source. The op-node sequencer (L2CL) produces blocks, the batcher lands them
// on L1, and the verifier (L2CLB) must derive the same safe chain and stay in
// sync.
//
// This is the minimal target for running op-con-node as the verifier:
//
//	DEVSTACK_L2CL_KIND=op-con-node \
//	DEVSTACK_L2EL_KIND=op-reth \
//	RUST_BINARY_PATH_OP_CON_NODE=<opql>/target/debug/op-con-node \
//	go test -run TestOpConVerifierDerivesSafeHead ./op-acceptance-tests/tests/opcon/
//
// With the default op-node CL kind it is an ordinary L1-derivation sync check,
// so it is a valid suite addition rather than an op-con-node-only test. The
// without-P2P multinode preset is used deliberately: op-con-node has no P2P
// gossip, so the verifier's only data source is L1 derivation (driving its own
// execution engine over the Engine API).
func TestOpConVerifierDerivesSafeHead(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleChainMultiNodeNoFaultProofsWithoutP2PWithoutCheck(t)

	// The sequencer produces blocks and the batcher submits them to L1.
	sys.L2CL.Advanced(types.LocalSafe, 1, 60)

	// The verifier derives the safe chain from L1 and converges on the
	// sequencer's safe head.
	sys.L2CLB.Advanced(types.LocalSafe, 1, 60)
	sys.L2CLB.InSync(sys.L2CL, types.LocalSafe, 60)
}

// TestOpConVerifierViaP2P exercises the op-conp2p P2P sidecar end to end: the
// op-node sequencer publishes unsafe blocks over gossip, the op-conp2p sidecar
// (peered to the sequencer) receives them and delegates each block's signature
// verdict back to the op-con-node verifier's admin_verifyUnsafePayload, which
// ingests accepted blocks and drives its execution engine. The verifier's UNSAFE
// head must advance and converge on the sequencer's unsafe head — purely over
// P2P, with no follow source.
//
//	DEVSTACK_L2CL_KIND=op-con-node \
//	DEVSTACK_L2EL_KIND=op-reth \
//	RUST_BINARY_PATH_OP_CON_NODE=<opql>/target/debug/op-con-node \
//	OP_CONP2P_BIN=<optimism>/op-conp2p-bin \
//	go test -run TestOpConVerifierViaP2P ./op-acceptance-tests/tests/opcon/
//
// With the default op-node verifier kind it is an ordinary with-P2P unsafe-sync
// check, so it remains a valid suite addition rather than an op-con-node-only test.
func TestOpConVerifierViaP2P(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleChainMultiNodeNoFaultProofsWithP2PWithoutCheck(t)

	// The sequencer produces and gossips unsafe blocks.
	sys.L2CL.Advanced(types.LocalUnsafe, 1, 60)

	// The verifier receives them over P2P (via the sidecar) and converges on the
	// sequencer's unsafe head.
	sys.L2CLB.Advanced(types.LocalUnsafe, 1, 60)
	sys.L2CLB.InSync(sys.L2CL, types.LocalUnsafe, 60)
}

// TestOpConVerifierSafeHeadDatabaseMatches is the op-con-node variant of the
// op-node safeheaddb_clsync verifier test (TestPreserveDatabaseOnCLResync). That
// test was previously unreachable for op-con-node because it requires the
// verifier to sync over P2P; with the op-conp2p sidecar it now is.
//
// It exercises a real op-node verifier assertion against op-con-node:
// VerifySafeHeadDatabaseMatches compares optimism_safeHeadAtL1Block on the
// verifier against the op-node sequencer's SafeDB. op-con-node serves it from its
// SQL-derived rollup_rpc_safe_head_at_l1_history, so the two must agree.
//
// With the default op-node verifier kind this is an ordinary safe-head-DB parity
// check, so it remains a valid suite addition rather than an op-con-node-only test.
func TestOpConVerifierSafeHeadDatabaseMatches(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleChainMultiNodeNoFaultProofsWithP2PWithoutCheck(t,
		// Enable the op-node sequencer's SafeDB so it can answer
		// optimism_safeHeadAtL1Block as the source of truth. (The op-con-node
		// verifier serves its own from SQL and ignores the path — see safeDBOpt.)
		safeDBOpt(),
	)

	// The op-node sequencer batches blocks to L1; both it and the op-con-node
	// verifier advance their safe heads (the verifier derives safe from L1 and
	// gets unsafe over P2P via the sidecar).
	awaitSafeConvergence(t, sys, 60)

	// The verifier's safe-head-at-L1 history must match the sequencer's SafeDB.
	// Pin the walk depth to the converged safe head so a history truncated to its
	// newest entry cannot pass (the walk stops at the first missing lower entry).
	startSafeBlock := sys.L2CLB.HeadBlockRef(types.LocalSafe).Number
	sys.L2CLB.VerifySafeHeadDatabaseMatches(sys.L2CL, dsl.WithMinRequiredL2Block(startSafeBlock))
}

// TestOpConVerifierFillsUnsafeGaps is the op-con-node variant of the op-node
// sync/clsync/gap_clp2p test (TestSyncAfterInitialELSync): unsafe payloads posted
// out of order must be applied in order, with the canonical head only advancing
// once each gap is filled.
//
// The verifier is bare (no follow source, no P2P); the batcher is stopped so the
// safe head never advances and the unsafe head is driven purely by the posted
// payloads (admin_postUnsafePayload). op-con-node stores posted payloads and its
// SQL keeps only the contiguous valid prefix, so the head advances exactly when
// the gap closes — the same observable behavior as op-node's unsafe-payload queue.
//
// Valid suite addition: with the default op-node verifier it exercises op-node's
// own out-of-order unsafe-payload handling.
func TestOpConVerifierFillsUnsafeGaps(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleChainMultiNodeNoFaultProofsBareVerifierWithoutCheck(t,
		presets.WithBatcherOption(func(_ sysgo.ComponentTarget, cfg *bss.CLIConfig) {
			// Stop derivation so the safe head never advances; the unsafe head is
			// driven solely by the payloads this test posts.
			cfg.Stopped = true
		}),
	)
	require := t.Require()

	// Sequencer produces unsafe blocks the test will hand to the verifier.
	sys.L2CL.Advanced(types.LocalUnsafe, 7, 30)
	require.Zero(sys.L2CL.HeadBlockRef(types.LocalSafe).Number)
	require.Zero(sys.L2CLB.HeadBlockRef(types.LocalSafe).Number)

	startNum := sys.L2CLB.HeadBlockRef(types.LocalUnsafe).Number

	// Supply the first block; the verifier applies it and its head advances by 1.
	sys.L2CLB.SignalTarget(sys.L2EL, startNum+1)
	sys.L2ELB.WaitForBlockNumber(startNum + 1)

	// Post later blocks out of order with a gap at startNum+2: the head must stay.
	for _, delta := range []uint64{5, 3, 4, 7} {
		sys.L2CLB.SignalTarget(sys.L2EL, startNum+delta)
		require.Equal(startNum+1, sys.L2ELB.BlockRefByLabel(eth.Unsafe).Number)
	}

	// Fill the gap at startNum+2: the contiguous run startNum+1..startNum+5 becomes
	// canonical (startNum+7 still waits behind the gap at startNum+6).
	sys.L2CLB.SignalTarget(sys.L2EL, startNum+2)
	sys.L2ELB.Reached(eth.Unsafe, startNum+5, 2)

	// Fill the last gap at startNum+6: startNum+6, startNum+7 become canonical.
	sys.L2CLB.SignalTarget(sys.L2EL, startNum+6)
	sys.L2ELB.Reached(eth.Unsafe, startNum+7, 2)
}

// TestOpConVerifierResumesAfterRestart is the op-con-node analogue of the verifier
// restart in the op-node safeheaddb_clsync test: the verifier is stopped while the
// sequencer keeps batching to L1, then restarted and must resume from its shutdown
// checkpoint and catch back up to the safe head. (op-con-node re-derives its safe
// head from L1 rather than reading a persisted SafeDB, so this exercises restart
// resume + re-derivation through the Feldera checkpoint restore path.)
func TestOpConVerifierResumesAfterRestart(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleChainMultiNodeNoFaultProofsWithoutP2PWithoutCheck(t)

	// Verifier derives the safe chain from L1 and converges on the sequencer.
	awaitSafeConvergence(t, sys, 60)

	// Stop the verifier (writing a shutdown checkpoint); the sequencer keeps
	// batching to L1.
	sys.L2CLB.Stop()
	sys.L2CL.Advanced(types.LocalSafe, 3, 60)

	// Restart the verifier; it restores from the checkpoint, re-derives from L1,
	// and catches back up to the sequencer's safe head.
	sys.L2CLB.Start()
	sys.L2CLB.InSync(sys.L2CL, types.LocalSafe, 90)
	sys.L2CLB.Advanced(types.LocalSafe, 1, 60)
}

// TestOpConVerifierFollowMode is the op-con-node analogue of the op-node
// sync/follow_l2 test (which skips kona — "single-chain test sequencer requires
// an op-node CL node"). op-node's CL follow source has no direct op-con-node
// equivalent, but op-con-node's own follow mode (--l2-follow-rpc, pointed at the
// sequencer's L2 execution RPC) is the same idea: the verifier's unsafe head
// tracks the sequencer via the follow source while its safe head derives from L1.
//
// This asserts that follow-mode kernel against op-con-node: the unsafe head
// advances and stays in sync via follow-mode (no P2P, no sidecar), the safe head
// converges via L1 derivation, and CurrentL1 (optimism_syncStatus) tracks the
// sequencer. With the default op-node verifier it is an ordinary follow-source
// sync check, so it remains a valid suite addition.
func TestOpConVerifierFollowMode(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleChainMultiNodeNoFaultProofsWithoutP2PWithoutCheck(t)

	target := uint64(3)
	attempts := 60

	// Unsafe head tracks the sequencer via follow-mode; safe head derives from L1.
	dsl.CheckAll(t,
		sys.L2CL.ReachedFn(types.LocalUnsafe, target, attempts),
		sys.L2CLB.ReachedFn(types.LocalUnsafe, target, attempts),
	)
	sys.L2CLB.InSync(sys.L2CL, types.LocalUnsafe, attempts)
	sys.L2CLB.InSync(sys.L2CL, types.LocalSafe, attempts)

	// CurrentL1 (from optimism_syncStatus) follows the source.
	dsl.CheckAll(t, sys.L2CLB.CurrentL1MatchedFn(sys.L2CL, 20))
}

// TestOpConVerifierFollowModeFinalized is the op-con-node analogue of the op-node
// sync/follow_l2 test TestFollowL2_Safe_Finalized_CurrentL1. It extends
// TestOpConVerifierFollowMode with the FINALIZED head: the verifier's unsafe head
// (follow mode), safe head (L1 derivation), AND finalized head must all advance to
// the target and converge with the sequencer, plus CurrentL1 tracking.
//
// The finalized head is the property under test. op-con-node finalizes by the
// batch-inclusion L1 block — the canonical OP finality rule: an L2 block is final
// once the L1 data (its batch) proving it is final. (finalized_head in
// engine_boundary/module.sql gates the highest derived block on
// `l1_inclusion_input_id <= finalized L1`, the batch-inclusion L1 coordinate.) This
// matches op-node, so exact finalized parity holds. An earlier revision keyed on the
// L2 block's L1 *origin*, which runs ~a batch-window ahead of op-node and finalizes
// before the L1 data proving the block is final — that gap is what this test caught
// and the finalized_head fix closes.
//
// The devstack op-con-node default is --l1-finalized-guard=disabled (so startup never
// blocks on a finalized tag against a non-finalizing L1), which suppresses the
// finalized L1 status ref entirely. This test runs against a finalizing devstack L1,
// so it sets the per-node OpConL1FinalizedGuard="required" to make op-con-node track
// L1 finality and advance its L2 finalized head.
//
// With the default op-node verifier kind it is an ordinary follow-source safe/
// finalized/CurrentL1 sync check, so it remains a valid suite addition.
func TestOpConVerifierFollowModeFinalized(gt *testing.T) {
	t := devtest.ParallelT(gt)

	// Make op-con-node track L1 finality (devstack default is disabled). Per-node via
	// the L2CL option, so parallel sibling tests keep the non-finalizing default.
	sys := presets.NewSingleChainMultiNodeNoFaultProofsWithoutP2PWithoutCheck(t,
		presets.WithGlobalL2CLOption(sysgo.L2CLOptionFn(func(_ devtest.T, _ sysgo.ComponentTarget, cfg *sysgo.L2CLConfig) {
			cfg.OpConL1FinalizedGuard = "required"
		})),
	)
	logger := t.Logger()

	target := uint64(3)
	// L1 finalization takes ~2 minutes in the devstack, so the finalized head needs a
	// generous budget; ReachedFn/InSyncFn poll every 2s.
	attempts := 90

	// Unsafe (follow mode), safe (L1 derivation), and finalized (L1 finality) heads
	// must each reach the target on both the sequencer and the op-con-node verifier,
	// and the verifier must stay in sync with the sequencer at each level.
	for _, lvl := range []types.SafetyLevel{types.LocalUnsafe, types.LocalSafe, types.Finalized} {
		dsl.CheckAll(t,
			sys.L2CL.ReachedFn(lvl, target, attempts),
			sys.L2CLB.ReachedFn(lvl, target, attempts),
		)
		sys.L2CLB.InSync(sys.L2CL, lvl, attempts)
		logger.Info("head reached + in sync", "level", lvl, "target", target)
	}

	// CurrentL1 (from optimism_syncStatus) follows the source.
	dsl.CheckAll(t, sys.L2CLB.CurrentL1MatchedFn(sys.L2CL, 20))
}

// TestOpConVerifierRecoversFromL1Reorg is the op-con-node variant of the op-node
// L1-reorg-recovery test (sync/elsync/reorg TestUnsafeGapFillAfterSafeReorg): an
// L1 reorg must reset op-con-node's derivation pipeline and re-derive the safe
// chain on the new canonical L1.
//
// The verifier is bare (no follow/unsafe source) so it derives only the safe
// chain from L1 — the property under test. Mirroring the op-node test, the reorg
// targets the SEQUENCER's safe head: its L1 origin tracks near the L1 head, so the
// single-block fork (built via the test-sequencer + auto-advance) is shallow
// enough to win the fork choice. Targeting the verifier's safe origin instead
// fails — op-con-node's safe head lags L1 during catch-up, so that origin is deep
// and a short fork there loses to the longer canonical chain. We additionally
// wait for the (lagging) op-con-node verifier to reach the reorg target so the
// reorg invalidates a block it actually derived. Exercises op-con-node's
// canonical-L1-tracker reorg handling + safe re-derivation onto the new L1.
//
// This caught a real op-con-node freeze (since fixed): the reorg path wrote its L1
// horizon row keyed by a fresh PayloadId instead of the reorg's max block number,
// so that row permanently won the horizon's ARG_MAX and pinned visible L1 at the
// reorg boundary — every block appended after the reorg stayed invisible to
// derivation and the safe chain froze. Fixed in op-con-node row_writers.rs
// (apply_canonical_l1_delta keys the horizon row by max_input_id, matching the
// append path). See docs/sidecar-p2p-design.md.
func TestOpConVerifierRecoversFromL1Reorg(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleChainMultiNodeNoFaultProofsBareVerifierWithTestSeqWithoutCheck(t)
	require := t.Require()
	logger := t.Logger()
	ctx := t.Ctx()

	ts := sys.TestSequencer.Escape().ControlAPI(sys.L1Network.ChainID())
	sys.L1Network.WaitForBlock() // pass the L1 genesis

	// Drive L1 deterministically so we can reorg it.
	sys.L1CL.Stop()
	startL1Block := sys.L1EL.BlockRefByLabel(eth.Unsafe)

	// Advance L1 until the SEQUENCER's safe head has an L1 origin past the start
	// block. The sequencer's safe origin stays near the L1 head (small lag), which
	// keeps the reorg shallow enough to win.
	require.Eventually(func() bool {
		if err := ts.Next(ctx); err != nil {
			logger.Warn("ts.Next failed, will retry", "err", err)
			return false
		}
		l2Safe := sys.L2EL.BlockRefByLabel(eth.Safe)
		logger.Info("driving L1", "l1_head", sys.L1EL.BlockRefByLabel(eth.Unsafe), "seq_safe", l2Safe)
		return l2Safe.Number > 0 && l2Safe.L1Origin.Number > startL1Block.Number
	}, 120*time.Second, 2*time.Second)

	l2BlockBeforeReorg := sys.L2EL.BlockRefByLabel(eth.Safe)
	logger.Info("target safe block to reorg", "l2", l2BlockBeforeReorg, "l1_origin", l2BlockBeforeReorg.L1Origin)

	// Wait for the (lagging) op-con-node verifier to also derive this safe block,
	// so the reorg invalidates a block it actually produced.
	sys.L2ELB.Reached(eth.Safe, l2BlockBeforeReorg.Number, 60)

	// Reorg the L1 block that the safe block's L1 origin points to.
	l1BlockBeforeReorg := sys.L1EL.BlockRefByNumber(l2BlockBeforeReorg.L1Origin.Number)
	logger.Info("triggering L1 reorg", "l1", l1BlockBeforeReorg)
	require.NoError(ts.New(ctx, seqtypes.BuildOpts{Parent: l1BlockBeforeReorg.ParentHash}))
	require.NoError(ts.Next(ctx))
	sys.L1CL.Start()

	// Confirm the L1 reorg took (the new fork won at that height).
	sys.L1EL.WaitForBlockNumber(l1BlockBeforeReorg.Number)
	l1BlockAfterReorg := sys.L1EL.BlockRefByNumber(l1BlockBeforeReorg.Number)
	require.NotEqual(l1BlockAfterReorg.Hash, l1BlockBeforeReorg.Hash)
	logger.Info("L1 reorg confirmed", "l1", l1BlockAfterReorg)

	// The sequencer's safe head switches onto the new L1 chain. The reorg drops
	// several L1 blocks (the safe origin structurally lags the L1 head by the
	// batcher channel window), so the sequencer's batcher must re-submit a handful
	// of epochs of batches against the new L1 before its safe head advances —
	// allow a generous budget. op-con-node stays live throughout (unlike op-node's
	// reference test, which stops its verifier to free CPU during this window).
	sys.L2EL.WaitL1OriginHash(eth.Safe, l1BlockAfterReorg.ID(), 90)

	// op-con-node detects the L1 reorg, resets derivation, and re-derives the new
	// safe chain: its block at the reorged height is replaced, and its safe head
	// converges with the sequencer's.
	sys.L2ELB.ReorgTriggered(l2BlockBeforeReorg, 90)
	sys.L2ELB.InSync(sys.L2EL, eth.Safe, 90)
}

// TestOpConVerifierFollowModeReorgRecovery is the op-con-node variant of the
// op-node sync/follow_l2 test TestFollowL2_ReorgRecovery (which skips kona —
// "single-chain test sequencer requires an op-node CL node"). It extends
// TestOpConVerifierRecoversFromL1Reorg's safe-side reorg check with the unsafe
// (follow-mode) side: the verifier additionally runs op-con-node's follow mode
// (--l2-follow-rpc pointed at the sequencer's L2 EL), so an L1 reorg reorgs *both*
// of the verifier's inputs at once — the L1-derived safe chain and the follow
// source's unsafe chain — and the verifier must recover both.
//
// op-node's follow_l2 test splits these roles across two verifiers (a
// derivation-enabled verifier and a follow-source verifier); op-con-node combines
// them in one node (follow mode for the unsafe head, L1 derivation for the safe
// head), so the single op-con-node verifier exercises both reset paths together.
// This is the unsafe-side analogue of the safe-side L1-reorg recovery, and it
// specifically exercises op-con-node's follow-mode source-reorg handling
// (l2_follow.rs: parent-linkage divergence detection → prefetcher reset → re-feed)
// plus the engine-boundary divergence-floor cap that lets the re-derived safe chain
// drive op-reth onto the new canonical chain past the now-stale fed unsafe tip.
//
// With the default op-node verifier kind this is an ordinary follow-source +
// derivation L1-reorg-recovery check, so it remains a valid suite addition rather
// than an op-con-node-only test. Reorg mechanics (target the sequencer's safe head,
// auto-advance L1 to win the fork choice, wait for the lagging verifier to derive
// the target first) mirror TestOpConVerifierRecoversFromL1Reorg — see its comment.
func TestOpConVerifierFollowModeReorgRecovery(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleChainMultiNodeNoFaultProofsFollowVerifierWithTestSeqWithoutCheck(t)
	require := t.Require()
	logger := t.Logger()
	ctx := t.Ctx()

	ts := sys.TestSequencer.Escape().ControlAPI(sys.L1Network.ChainID())
	sys.L1Network.WaitForBlock() // pass the L1 genesis

	// Drive L1 deterministically so we can reorg it.
	sys.L1CL.Stop()
	startL1Block := sys.L1EL.BlockRefByLabel(eth.Unsafe)

	// Advance L1 until the SEQUENCER's safe head has an L1 origin past the start
	// block (keeps the reorg shallow enough to win — see the bare-verifier test).
	require.Eventually(func() bool {
		if err := ts.Next(ctx); err != nil {
			logger.Warn("ts.Next failed, will retry", "err", err)
			return false
		}
		l2Safe := sys.L2EL.BlockRefByLabel(eth.Safe)
		logger.Info("driving L1", "l1_head", sys.L1EL.BlockRefByLabel(eth.Unsafe), "seq_safe", l2Safe)
		return l2Safe.Number > 0 && l2Safe.L1Origin.Number > startL1Block.Number
	}, 120*time.Second, 2*time.Second)

	// Confirm op-con-node's follow mode is live before the reorg: the verifier's
	// op-reth unsafe (canonical) head, fed by follow mode, must lead its L1-derived
	// safe head, so the upcoming reorg reorgs the *fed* unsafe chain and not only the
	// derived safe chain. We read op-reth directly (sys.L2ELB) rather than
	// op-con-node's syncStatus: while L1 is paused the unsafe lead is only a block or
	// two, and op-con-node's reported unsafe head (the sparse forkchoice-accepted view,
	// which needs a contiguous FCU-accepted chain) can momentarily fall back to the
	// safe head — but op-reth's actual canonical head reliably leads the safe block it
	// was FCU'd with.
	require.Eventually(func() bool {
		return sys.L2ELB.BlockRefByLabel(eth.Unsafe).Number > sys.L2ELB.BlockRefByLabel(eth.Safe).Number
	}, 120*time.Second, 2*time.Second)

	l2BlockBeforeReorg := sys.L2EL.BlockRefByLabel(eth.Safe)
	logger.Info("target safe block to reorg", "l2", l2BlockBeforeReorg, "l1_origin", l2BlockBeforeReorg.L1Origin)

	// Wait for the (lagging) op-con-node verifier to also derive this safe block,
	// so the reorg invalidates a block it actually produced.
	sys.L2ELB.Reached(eth.Safe, l2BlockBeforeReorg.Number, 60)

	// Reorg the L1 block that the safe block's L1 origin points to.
	l1BlockBeforeReorg := sys.L1EL.BlockRefByNumber(l2BlockBeforeReorg.L1Origin.Number)
	logger.Info("triggering L1 reorg", "l1", l1BlockBeforeReorg)
	require.NoError(ts.New(ctx, seqtypes.BuildOpts{Parent: l1BlockBeforeReorg.ParentHash}))
	require.NoError(ts.Next(ctx))
	sys.L1CL.Start()

	// Confirm the L1 reorg took (the new fork won at that height).
	sys.L1EL.WaitForBlockNumber(l1BlockBeforeReorg.Number)
	l1BlockAfterReorg := sys.L1EL.BlockRefByNumber(l1BlockBeforeReorg.Number)
	require.NotEqual(l1BlockAfterReorg.Hash, l1BlockBeforeReorg.Hash)
	logger.Info("L1 reorg confirmed", "l1", l1BlockAfterReorg)

	// The sequencer's safe head — and, because the sequencer rebuilds its unsafe
	// chain on the new L1 origin, its unsafe head — switches onto the new L1 chain.
	// The follow source (the sequencer's L2 EL) therefore serves a reorged unsafe
	// chain to the verifier.
	sys.L2EL.WaitL1OriginHash(eth.Safe, l1BlockAfterReorg.ID(), 90)

	// op-con-node must recover on both inputs: it detects the L1 reorg and resets
	// derivation (re-deriving the new safe chain), and its follow-mode prefetcher
	// detects the source reorg and re-feeds the new unsafe chain. Its block at the
	// reorged height is replaced, and both its safe and unsafe heads converge with
	// the sequencer's.
	sys.L2ELB.ReorgTriggered(l2BlockBeforeReorg, 90)
	sys.L2ELB.InSync(sys.L2EL, eth.Safe, 90)
	sys.L2CLB.InSync(sys.L2CL, types.LocalSafe, 90)
	sys.L2CLB.InSync(sys.L2CL, types.LocalUnsafe, 90)
}
