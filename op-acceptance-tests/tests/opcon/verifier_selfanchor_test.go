package opcon

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// TestOpConVerifierSelfAnchorsAfterDatadirWipe is the devstack analog of a
// snapshot-restored boot: a FRESH op-con-node circuit (no --checkpoint file and
// no restorable datadir) started against a MID-CHAIN execution engine must
// derive its derivation anchor from the own EL's current head — decoding the
// L1 origin and SystemConfig from the L1-attributes deposit in
// transactions[0] — instead of re-anchoring at genesis. (Every ordinary
// devstack boot exercises the degenerate case of the same path: an EL still at
// genesis self-anchors at genesis.)
//
// Shape: the verifier syncs normally, its CL is stopped, the CL's datadir
// (Feldera checkpoints) is wiped while its EL keeps the chain at head H, and
// the CL is restarted. The restarted CL must:
//   - self-anchor at H (op-con-node logs "self-anchored derivation at the own
//     EL's head"): the boot-window syncStatus watch below tolerates only
//     un-settled reads (unsafeL2 = 0) before the first >= H read, and fails
//     fast on any value in between — a genesis re-anchor would re-derive /
//     re-feed upward through exactly those values;
//   - treat the fresh anchor as the safe baseline: LocalSafe reads >= H
//     without re-deriving the pre-wipe chain from L1 genesis;
//   - not reorg the EL: the halted head keeps its exact hash, which by
//     parent-hash linking pins the whole pre-wipe chain;
//   - resume follow mode and L1 derivation from the anchor's (recent) L1
//     origin: unsafe advances past H and the safe head keeps advancing as the
//     batcher lands new batches.
//
// Unlike its sibling TestOpConVerifierResumesAfterRestart, the first
// syncStatus read is deliberately NOT single-shot: op-con-node's
// settle-before-serve RPC gate only engages when a Feldera checkpoint is
// restored, and after a wipe there is none — a fresh circuit serves RPC
// immediately, so its OUTPUT views may briefly read as empty before the seeded
// anchor settles. awaitSelfAnchoredUnsafeHead encodes exactly which boot-window
// transients are legitimate.
//
// Requires DEVSTACK_L2CL_KIND=op-con-node (datadir wipe and self-anchor are
// op-con-node launcher primitives); skipped under other CL kinds.
//
//	DEVSTACK_L2CL_KIND=op-con-node DEVSTACK_L2EL_KIND=op-reth \
//	RUST_BINARY_PATH_OP_CON_NODE=<opql>/target/debug/op-con-node \
//	go test -run TestOpConVerifierSelfAnchorsAfterDatadirWipe ./op-acceptance-tests/tests/opcon/
func TestOpConVerifierSelfAnchorsAfterDatadirWipe(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipUnlessOpConNode(t, "datadir wipe + EL-head self-anchor are op-con-node boot primitives")
	sys := presets.NewSingleChainMultiNodeNoFaultProofsWithoutP2PWithoutCheck(t)
	logger := t.Logger()

	// The verifier syncs a real prefix of the chain (unsafe via follow mode,
	// safe via L1 derivation) so the wipe discards genuinely derived state.
	awaitSafeConvergence(t, sys, 60)

	// Stop the verifier CL; nothing drives its EL while it is down, so the EL
	// freezes mid-chain at head H — the state a snapshot restore would produce.
	sys.L2CLB.Stop()
	halted := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	logger.Info("verifier EL unsafe head while CL stopped", "unsafe", halted)

	// Wipe the CL's datadir: the next Start() boots a FRESH circuit (nothing to
	// restore) against the untouched mid-chain EL.
	presets.SysgoOpConNode(t, sys.L2CLB).WipeDatadir()
	sys.L2CLB.Start()

	// The fresh circuit self-anchors at the EL head, never at genesis: the
	// boot window must go straight from un-settled reads to unsafeL2 >= H.
	awaitSelfAnchoredUnsafeHead(t, sys.L2CLB, halted.Number, 30*time.Second)

	// The fresh anchor IS the safe baseline: LocalSafe reads >= H from the
	// anchor seed alone, not from re-deriving the pre-wipe chain out of L1.
	sys.L2CLB.Reached(types.LocalSafe, halted.Number, 30)

	// The self-anchored boot did not reorg the EL: the halted head keeps its
	// exact hash, pinning the whole pre-wipe chain by parent-hash linking.
	sys.L2ELB.VerifyNotReorged(halted)

	// Follow mode resumes from the anchor: the unsafe head climbs past H
	// toward the sequencer tip (the sequencer kept producing throughout).
	sys.L2ELB.Reached(eth.Unsafe, halted.Number+3, 90)

	// L1 derivation resumes from the anchor's recent L1 origin: the safe head
	// advances past the anchor as the batcher lands new batches. A derivation
	// that stalled (or restarted from L1 genesis) would time out here.
	sys.L2CLB.Advanced(types.LocalSafe, 1, 90)
}

// awaitSelfAnchoredUnsafeHead watches the boot window of a FRESH op-con-node
// circuit started against a mid-chain EL and requires the first meaningful
// syncStatus unsafe head to already sit at the self-anchor (>= anchor, the EL
// head at boot).
//
// A fresh circuit has no checkpoint to restore, so the settle-before-serve RPC
// gate does not engage and the RPC can answer before the seeded anchor settles
// into the OUTPUT views. Legitimate boot-window reads are therefore only:
//   - RPC errors (server warming up) and unsafeL2 = 0 (views not settled) —
//     retried until the deadline;
//   - unsafeL2 >= anchor (the settled self-anchor, or follow-mode progress
//     beyond it) — success.
//
// Any read in (0, anchor) fails immediately: the node's unsafe head can only
// pass through those values by re-anchoring at genesis and re-deriving or
// re-feeding the chain upward — the exact regression this test pins. (The safe
// head cannot explain such a read either: at boot it equals the anchor.)
func awaitSelfAnchoredUnsafeHead(t devtest.T, cl *dsl.L2CLNode, anchor uint64, budget time.Duration) {
	require := t.Require()
	logger := t.Logger()
	rollupAPI := cl.Escape().RollupAPI()
	deadline := time.Now().Add(budget)
	for {
		ctx, cancel := context.WithTimeout(t.Ctx(), 2*time.Second)
		status, err := rollupAPI.SyncStatus(ctx)
		cancel()
		switch {
		case err != nil:
			logger.Info("boot-window syncStatus not answering yet; will retry", "err", err)
		case status.UnsafeL2.Number == 0:
			logger.Info("boot-window OUTPUT views not settled yet (unsafeL2 = 0); will retry")
		case status.UnsafeL2.Number < anchor:
			require.GreaterOrEqual(status.UnsafeL2.Number, anchor,
				"fresh-circuit boot reported an unsafe head below the EL head: the node re-anchored at genesis instead of self-anchoring at the EL's head")
		default:
			logger.Info("self-anchored unsafe head observed", "unsafe", status.UnsafeL2, "anchor", anchor)
			return
		}
		require.Falsef(time.Now().After(deadline),
			"no unsafe head >= %d (the EL head at boot) within %s: the fresh circuit never settled a self-anchored head", anchor, budget)
		select {
		case <-t.Ctx().Done():
			require.NoError(t.Ctx().Err(), "test context done while awaiting the self-anchored unsafe head")
			return
		case <-time.After(250 * time.Millisecond):
		}
	}
}
