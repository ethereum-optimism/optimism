package opcon

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// TestOpConSequencerResumesAfterRestart is the sequencer-side analogue of
// TestOpConVerifierResumesAfterRestart: op-con-node runs AS the L2 sequencer,
// producing the unsafe chain, and must survive a full node restart — resuming
// block production from where it left off without reorging or duplicating any of
// the blocks it already produced.
//
// The distinctive sequencer property under test is the build cursor's re-anchor.
// The build driver's per-height dedup (SequencerDriverState.highest_built) is
// in-process state that is lost on restart, so on resume it starts empty. It must
// re-anchor from the SQL imported-unsafe tip (sequencer_imported_unsafe_tip,
// backed by the checkpointed engine observation log restored from the Feldera
// shutdown checkpoint) and emit only tip+1 — NOT reset to genesis (which would
// deep-reorg the EL back to block 0 and rebuild the whole chain) and NOT re-offer
// the tip height (which would build a same-height twin, excluding the height from
// the imported tip and wedging production, the documented catch-up-stall failure
// mode).
//
// With the default op-node CL kind L2CLOpConSequencer() is a no-op, so this
// degrades to an ordinary op-node sequencer restart check — a valid suite
// addition rather than an op-con-only test.
//
//	DEVSTACK_L2CL_KIND=op-con-node DEVSTACK_L2EL_KIND=op-reth \
//	RUST_BINARY_PATH_OP_CON_NODE=<opql>/target/debug/op-con-node \
//	go test -run TestOpConSequencerResumesAfterRestart ./op-acceptance-tests/tests/opcon/
func TestOpConSequencerResumesAfterRestart(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimalNoFaultProofs(t, opConSequencerOpt())
	require := t.Require()
	logger := t.Logger()

	// The sequencer produces a run of unsafe blocks on its own cadence.
	sys.L2CL.Advanced(types.LocalUnsafe, 3, 60)

	// Stop the whole sequencer node with a graceful interrupt (SIGINT), which
	// triggers op-con-node's Feldera shutdown checkpoint (--datadir is set). While
	// the CL is down nothing drives new builds, so the EL's unsafe head freezes at
	// the last block the sequencer committed — the tip the restart must re-anchor
	// to.
	sys.L2CL.Stop()
	halted := sys.L2EL.BlockRefByLabel(eth.Unsafe)
	logger.Info("sequencer EL unsafe head while CL stopped", "unsafe", halted)
	require.Greater(halted.Number, uint64(0), "sequencer should have produced blocks before the restart")

	// Restart the sequencer node. It restores the Feldera checkpoint and, with an
	// empty in-process build-dedup cursor, must re-anchor from the restored imported
	// tip and resume producing tip+1 onward.
	sys.L2CL.Start()

	// Single-shot, no retry: the very first sync-status read after the restart must
	// already report the restored unsafe head, never a genesis (unsafeL2 = 0) reset.
	// op-con-node gates its RPC surface behind the post-checkpoint restore step
	// (opql-runtime Program::settle_before_serve): the OUTPUT views are populated
	// before the server accepts requests, so Start() returns only once syncStatus
	// serves the real head. A consumer polling in this window — notably op-batcher
	// reading the sequencer's optimism_syncStatus UnsafeL2 to decide what to batch —
	// can no longer catch a transient reset-to-genesis. (op-node loads its head
	// synchronously before serving, so under the default CL kind this holds too.)
	// Before the gate landed this read raced the restore step and flaked at 0.
	resumedHead := sys.L2CL.SyncStatus().UnsafeL2.Number
	require.GreaterOrEqual(resumedHead, halted.Number,
		"first syncStatus read after restart is below the halted tip: the RPC surface served an un-restored (genesis) head before the checkpoint restore step completed")

	// Production resumes: the unsafe head climbs several blocks past where it halted.
	// A build cursor that reset to genesis or wedged on a duplicate height would
	// stall here (the head would not advance) and this would time out.
	sys.L2EL.Reached(eth.Unsafe, halted.Number+3, 90)

	// The restart did not reorg the chain the sequencer already produced: the block
	// at the halted tip keeps its exact hash. Because every L2 block commits to its
	// parent hash, an unchanged tip hash transitively proves the whole chain from
	// genesis up to the tip is byte-identical — so a genesis re-anchor that rebuilt
	// any block differently, or a tip-height rebuild, would change this hash. (A
	// bit-for-bit identical deterministic rebuild would be indistinguishable here,
	// but the op-con-node logs confirm it restores from the Feldera shutdown
	// checkpoint rather than replaying from genesis.)
	require.Equal(halted.Hash, sys.L2EL.BlockRefByNumber(halted.Number).Hash,
		"restart reorged the already-produced unsafe chain: the build cursor did not re-anchor at the imported tip")
}
