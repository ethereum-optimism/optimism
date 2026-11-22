package offset_el_safe

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// TestELSyncSafeRetractedByOffset verifies that when OffsetELSafe is configured on
// the verifier, the safe and finalized EL heads are set behind the unsafe head after
// EL sync completes.
//
// Flow:
//  1. Both nodes advance LocalSafe (ensures the CL has real finalized state).
//  2. Stop/wipe the verifier EL to force a full EL sync on restart.
//  3. Stop the batcher, then advance the sequencer's unsafe chain further.
//     Stopping the batcher makes the gap permanent: derivation can never advance
//     safe past the last batched block, eliminating timing flakes.
//  4. Restart verifier, peer ELs, and wait for EL sync to complete.
//  5. Assert safe/finalized are set the configured offset behind the EL-sync tip.
func TestELSyncSafeRetractedByOffset(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newOffsetELSafeSystem(t)
	require := t.Require()
	logger := t.Logger()

	// Advance both CLs to LocalSafe so the verifier CL has valid finalized state
	// before we stop it (prevents "forkchoice not initialized" after EL wipe).
	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(safety.LocalSafe, 1, 30),
		sys.L2CLB.AdvancedFn(safety.LocalSafe, 1, 30))
	sys.L2CLB.InSync(sys.L2CL, safety.LocalSafe, 30)
	// Batches up to this block are on L1, so the verifier can re-derive to it after EL sync.
	batchedSafe := sys.L2CLB.HeadBlockRef(safety.LocalSafe).Number

	// Stop verifier and wipe its EL to force a full EL sync on restart.
	sys.L2ELB.Stop()
	sys.L2ELB.Wipe()
	sys.L2CLB.Stop()

	// Stop the batcher so new sequencer blocks are unsafe-only.
	// This creates a permanent gap between safe and unsafe after EL sync:
	// derivation can only advance safe up to the last batched block.
	sys.L2Batcher.Stop()

	// Advance sequencer further while verifier is down (unsafe only, no batches).
	// Going more than the offset past the last batched block keeps derivation from
	// reaching the offset head, so the offset alone decides where safe lands.
	offsetBlocks := nodeSync.DurationToBlocks(testOffset, sys.L2Chain.Escape().RollupConfig().BlockTime)
	require.NotZero(offsetBlocks)
	sys.L2CL.Advanced(safety.LocalUnsafe, offsetBlocks+3, 30)
	sequencerUnsafe := sys.L2CL.HeadBlockRef(safety.LocalUnsafe).Number

	sys.L2ELB.Start()
	t.Require().Zero(sys.L2ELB.BlockRefByLabel(eth.Unsafe).Number, "EL resync must start from genesis")
	sys.L2CLB.Start()
	sys.L2ELB.WaitForOnline()
	sys.L2ELB.PeerWith(sys.L2EL)

	// Wait for absolute heights rather than for progress from whatever the heads are
	// when the wait starts: a fast node can finish EL sync before then, and with the
	// batcher stopped the heads would not move again.
	//  - Unsafe reaching the sequencer's tip shows the verifier received the tip over gossip.
	//  - LocalSafe reaching the pre-wipe safe head shows EL sync completed: until then
	//    local-safe is at genesis. After completion, safe is the offset head, which is
	//    above every batched block.
	// A full EL sync after a wipe is slower than the initial sync, so allow the
	// same budget as TestTruncateDatabaseOnELResync.
	sys.L2CLB.Reached(safety.LocalUnsafe, sequencerUnsafe, 90)
	sys.L2CLB.Reached(safety.LocalSafe, batchedSafe, 90)

	// Read one CL snapshot: the CL can move a label before the matching forkchoice
	// update reaches the EL, so separate EL label reads could be inconsistent.
	status := sys.L2CLB.SyncStatus()
	unsafeHead, safeHead, finalizedHead := status.UnsafeL2, status.SafeL2, status.FinalizedL2

	logger.Info("Verifier heads after EL sync",
		"unsafe", unsafeHead.Number,
		"safe", safeHead.Number,
		"finalized", finalizedHead.Number,
		"sequencer_unsafe_before_restart", sequencerUnsafe,
		"offset_blocks", offsetBlocks)

	// EL sync completes at a tip at or above sequencerUnsafe and at or below the
	// current unsafe head, and safe/finalized land offsetBlocks behind it. The
	// batcher is stopped and every batched block is below the offset head, so
	// derivation moves neither label. Checking both sides fails if the offset is
	// ignored, whether that leaves safe at the tip or applies the 12h default.
	for _, head := range []eth.L2BlockRef{safeHead, finalizedHead} {
		require.GreaterOrEqual(head.Number, sequencerUnsafe-offsetBlocks,
			"head %d retracted by more than the %s offset", head.Number, testOffset)
		require.GreaterOrEqual(unsafeHead.Number-head.Number, offsetBlocks,
			"head %d retracted by less than the %s offset", head.Number, testOffset)
	}
	require.Equal(safeHead, finalizedHead, "EL sync sets safe and finalized to the same block")
}
