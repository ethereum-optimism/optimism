package opcon

import (
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// TestOpConVerifierResumesAfterCrash is the hard-kill sibling of
// TestOpConVerifierResumesAfterRestart: the verifier is SIGKILLed — no
// shutdown checkpoint — and must come back from its last PERIODIC Feldera
// checkpoint, which is OLDER than its EL head (blocks fed after the last
// checkpoint reached the EL but not the persisted circuit). That is the
// crash-restart shape the graceful test cannot produce, and it exercises two
// boot behaviors together:
//   - the stored anchor is authoritative on a restored circuit: the
//     freshly-derived EL-head boot anchor (self-anchor is first-boot-only) is
//     discarded rather than fed into restored state;
//   - the follow engine clamps its start cursor to the restored confirmed
//     head + 1 when the boot anchor sits ahead of restored state, re-feeding
//     the checkpoint-to-EL-head overlap idempotently (op-con-node logs "sync
//     start cursor clamped to the local confirmed head + 1"). The pre-clamp
//     failure mode — a start cursor beyond the restored confirmed head — shows
//     as a frozen unsafe head, which the sequencer-tip convergence below
//     catches.
//
// Periodic checkpoints default to 60s, longer than this test's whole pre-kill
// phase — the datadir would hold no checkpoint at all and the restart would
// degrade into a fresh self-anchored boot instead of a restore. The
// DEVSTACK_OPCON_CHECKPOINT_INTERVAL_SECS launcher env shrinks the interval so
// a periodic checkpoint predating the EL head reliably exists at kill time.
// (The env is process-global; parallel sibling tests just checkpoint more
// often, which is behaviorally harmless, so it is set once and never unset.)
//
// Requires DEVSTACK_L2CL_KIND=op-con-node (SIGKILL without a shutdown
// checkpoint and the Feldera restore path are op-con-node launcher
// primitives); skipped under other CL kinds.
//
//	DEVSTACK_L2CL_KIND=op-con-node DEVSTACK_L2EL_KIND=op-reth \
//	RUST_BINARY_PATH_OP_CON_NODE=<opql>/target/debug/op-con-node \
//	go test -run TestOpConVerifierResumesAfterCrash ./op-acceptance-tests/tests/opcon/
func TestOpConVerifierResumesAfterCrash(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipUnlessOpConNode(t, "SIGKILL crash + periodic-Feldera-checkpoint restore are op-con-node boot primitives")
	require := t.Require()
	logger := t.Logger()

	// Shrink the periodic checkpoint interval (see the doc comment) BEFORE the
	// preset launches the nodes; the launcher reads it at start time.
	require.NoError(os.Setenv("DEVSTACK_OPCON_CHECKPOINT_INTERVAL_SECS", "5"),
		"set op-con-node periodic checkpoint interval env")

	sys := presets.NewSingleChainMultiNodeNoFaultProofsWithoutP2PWithoutCheck(t)

	// The verifier syncs a real prefix (unsafe via follow mode, safe via L1
	// derivation); the >= 30s this takes guarantees several 5s periodic
	// checkpoints have been written by kill time.
	awaitSafeConvergence(t, sys, 60)

	// SIGKILL the verifier CL: no shutdown checkpoint, so the datadir keeps
	// only periodic checkpoints — the newest up to 5s (a few blocks) behind
	// the EL head frozen below.
	presets.SysgoOpConNode(t, sys.L2CLB).Kill()
	halted := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	logger.Info("verifier EL unsafe head after crash", "unsafe", halted)

	// The sequencer keeps producing and batching while the verifier is down,
	// so the restarted verifier has real catch-up to do on both heads.
	sys.L2CL.Advanced(types.LocalSafe, 2, 60)

	// Restart: the node restores the periodic checkpoint, discards the
	// EL-head boot anchor, clamps the follow cursor to the restored confirmed
	// head + 1, and re-feeds through the overlap. (A startup bail would fail
	// inside Start(), which waits for the RPC to come up.)
	sys.L2CLB.Start()

	// The unsafe head resumes and converges on the sequencer tip: a wedged
	// follow cursor (the pre-clamp failure mode) would freeze it at or below
	// the crash head and time out here.
	sys.L2CLB.InSync(sys.L2CL, types.LocalUnsafe, 90)

	// Re-feeding the checkpoint-to-EL-head overlap was idempotent: the crash
	// head keeps its exact hash, pinning the whole pre-crash chain.
	sys.L2ELB.VerifyNotReorged(halted)

	// L1 derivation also resumes from the restored checkpoint: the safe head
	// catches back up to the sequencer's and keeps advancing.
	sys.L2CLB.InSync(sys.L2CL, types.LocalSafe, 90)
	sys.L2CLB.Advanced(types.LocalSafe, 1, 60)
}
