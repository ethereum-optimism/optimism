package opcon

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
)

// TestOpConSequencerRecoversFromL1Reorg is the sequencer-side flip of
// TestOpConVerifierRecoversFromL1Reorg: op-con-node runs AS the L2 sequencer, so
// an L1 reorg reorgs BOTH chains it owns at once — the safe chain it derives from
// L1 AND the unsafe chain it produces on L1 origins. It must detect the L1 reorg,
// reset derivation and re-derive the safe chain on the new canonical L1, and
// rebuild its unsafe chain on the new origin: the blocks it built on the reorged
// origin carry a now-stale L1-info deposit and must be replaced, then production
// resumes. The bare op-con verifier re-derives the same safe chain in parallel.
//
// The verifier test only exercises safe-chain recovery (an op-node sequences);
// putting op-con-node in the sequencer slot adds the unsafe-chain rebuild — the
// least-tested sequencer path — which is what this test pins.
//
//	DEVSTACK_L2CL_KIND=op-con-node DEVSTACK_L2EL_KIND=op-reth \
//	RUST_BINARY_PATH_OP_CON_NODE=<opql>/target/debug/op-con-node \
//	go test -run TestOpConSequencerRecoversFromL1Reorg ./op-acceptance-tests/tests/opcon/
//
// With the default op-node CL kind L2CLOpConSequencer() is a no-op, so this
// degrades to an ordinary op-node-sequencer L1-reorg check — a valid suite
// addition rather than an op-con-only test.
//
// SKIPPED pending an op-con-node feature: sequencer-mode L1-reorg recovery is not
// implemented. Observed failure — after the L1 reorg the sequencer's safe head
// never re-derives onto the new L1 (WaitL1OriginHash(Safe) times out). The
// sequencer keeps extending its UNSAFE chain at normal cadence on the
// reorged-away L1 origin (unsafe climbed 7 → ~96) while its safe head stays
// frozen at the pre-reorg block, so the two chains diverge permanently: the
// batcher submits stale-origin blocks that are underivable on the new canonical
// L1. op-con-node's build loop does not rewind its unsafe head / rebuild on an L1
// reorg — the sequencer analog of the follow-mode source-reorg recovery
// (l2_follow.rs) and the engine-boundary divergence floor, neither of which
// drives the self-produced sequencer cursor. The verifier variant
// (TestOpConVerifierRecoversFromL1Reorg) passes because an op-node sequencer
// rebuilds + re-batches on the new L1, feeding the op-con verifier valid batches.
// See docs/op-con-node-sequencer-acceptance-tests.md.
func TestOpConSequencerRecoversFromL1Reorg(gt *testing.T) {
	gt.Skip("op-con-node sequencer does not rebuild its chain on an L1 reorg " +
		"(unsafe keeps extending the reorged-away origin; safe never re-derives); see test doc")
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleChainMultiNodeNoFaultProofsBareVerifierWithTestSeqWithoutCheck(t,
		presets.WithGlobalL2CLOption(sysgo.L2CLOpConSequencer()))
	require := t.Require()
	logger := t.Logger()
	ctx := t.Ctx()

	ts := sys.TestSequencer.Escape().ControlAPI(sys.L1Network.ChainID())
	sys.L1Network.WaitForBlock() // pass the L1 genesis

	// Drive L1 deterministically so we can reorg it.
	sys.L1CL.Stop()
	startL1Block := sys.L1EL.BlockRefByLabel(eth.Unsafe)

	// Advance L1 until the op-con sequencer's safe head has an L1 origin past the
	// start block. The safe origin tracks near the L1 head (it lags only by the
	// batcher channel window), which keeps the reorg shallow enough to win.
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

	// Wait for the (lagging) bare op-con verifier to also derive this safe block,
	// so the reorg invalidates a block it actually produced.
	sys.L2ELB.Reached(eth.Safe, l2BlockBeforeReorg.Number, 60)

	// Snapshot the sequencer's unsafe tip just before the reorg, to later confirm
	// it rebuilds past this height rather than wedging.
	unsafeBeforeReorg := sys.L2EL.BlockRefByLabel(eth.Unsafe)
	logger.Info("sequencer unsafe tip before reorg", "unsafe", unsafeBeforeReorg)

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

	// The op-con sequencer re-derives its safe chain onto the new L1. The reorg
	// drops several L1 blocks (the safe origin structurally lags the L1 head by the
	// batcher channel window), so the batcher must re-submit a handful of epochs of
	// batches against the new L1 before the safe head advances — allow a generous
	// budget. Its block at the reorged height is replaced.
	sys.L2EL.WaitL1OriginHash(eth.Safe, l1BlockAfterReorg.ID(), 90)
	sys.L2EL.ReorgTriggered(l2BlockBeforeReorg, 90)

	// The distinctive sequencer property: it rebuilds its UNSAFE chain on the new
	// canonical L1 origin and RESUMES producing. The unsafe head climbs back past
	// where it stood before the reorg — a reorg that wedged the build loop (failing
	// to rewind + re-produce the stale-origin blocks) would stall it here.
	sys.L2EL.Reached(eth.Unsafe, unsafeBeforeReorg.Number+3, 90)

	// The bare op-con verifier re-derives the same safe chain and converges with
	// the sequencer.
	sys.L2ELB.InSync(sys.L2EL, eth.Safe, 90)
}
