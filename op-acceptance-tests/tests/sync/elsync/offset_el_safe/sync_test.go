package offset_el_safe

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
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
//  5. Assert safe/finalized are behind unsafe.
func TestELSyncSafeRetractedByOffset(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnOpReth(t, "not supported (peering issue)")
	sys := newOffsetELSafeSystem(t)
	require := t.Require()
	logger := t.Logger()

	// Advance both CLs to LocalSafe so the verifier CL has valid finalized state
	// before we stop it (prevents "forkchoice not initialized" after EL wipe).
	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalSafe, 1, 30),
		sys.L2CLB.AdvancedFn(types.LocalSafe, 1, 30))
	sys.L2CLB.InSync(sys.L2CL, types.LocalSafe, 30)

	// Stop verifier and wipe its EL to force a full EL sync on restart.
	sys.L2ELB.Stop()
	sys.L2CLB.Stop()

	// Stop the batcher so new sequencer blocks are unsafe-only.
	// This creates a permanent gap between safe and unsafe after EL sync:
	// derivation can only advance safe up to the last batched block.
	sys.L2Batcher.Stop()

	// Advance sequencer further while verifier is down (unsafe only, no batches).
	sys.L2CL.Advanced(types.LocalUnsafe, 3, 30)

	sys.L2ELB.Start()
	sys.L2CLB.Start()
	sys.L2ELB.PeerWith(sys.L2EL)

	// Wait for the verifier CL to advance LocalSafe, which confirms:
	//  - EL sync completed (blocks transferred via p2p)
	//  - CL detected completion and sent the forkchoice update (setting safe/finalized)
	//  - Derivation began processing old batched data
	// Because the batcher is stopped, derivation can only reach the last batched
	// block — the gap between safe and unsafe is permanent.
	sys.L2CLB.Advanced(types.LocalSafe, 1, 30)

	unsafeHead := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	safeHead := sys.L2ELB.SafeHead().BlockRef
	finalizedHead := sys.L2ELB.FinalizedHead().BlockRef

	logger.Info("Verifier heads after EL sync",
		"unsafe", unsafeHead.Number,
		"safe", safeHead.Number,
		"finalized", finalizedHead.Number)

	// Safe and finalized must be behind unsafe when offset is configured.
	// The batcher is stopped, so derivation can never close the gap — these
	// assertions are deterministic regardless of timing.
	require.Greater(unsafeHead.Number, safeHead.Number,
		"safe head should be behind unsafe head when OffsetELSafe is configured")
	require.Greater(unsafeHead.Number, finalizedHead.Number,
		"finalized head should be behind unsafe head when OffsetELSafe is configured")
	require.GreaterOrEqual(safeHead.Number, finalizedHead.Number,
		"safe head should be at or ahead of finalized head")

	retraction := unsafeHead.Number - safeHead.Number
	logger.Info("Observed retraction", "blocks", retraction)
	require.Greater(retraction, uint64(0), "retraction must be nonzero")
}
