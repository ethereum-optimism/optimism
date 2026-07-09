package opcon

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// TestOpConSequencerSafeHeadDatabaseMatches is the sequencer-side flip of
// TestOpConVerifierSafeHeadDatabaseMatches: op-con-node runs AS the L2 sequencer
// and serves optimism_safeHeadAtL1Block as the source of truth, and an
// independently-deriving op-con verifier's own history must match it.
//
// The verifier variant pins op-con-node's safe-head-at-L1 history while op-con
// only VERIFIES (an op-node sequences and is the source of truth). Here op-con-node
// PRODUCES the chain and batches it to L1, then serves the history for the blocks
// it sequenced: its SQL-derived rollup_rpc_safe_head_at_l1_history must be a
// correct, op-node-SafeDB-compatible answer that a second, independently-deriving
// op-con-node reproduces exactly. A wrong L1-origin selection or a batch/derive
// mismatch on the produced chain would show up as a safe-head-at-L1 divergence
// between the two nodes.
//
// optimism_safeHeadAtL1Block compares the checking node's history against the
// source of truth's, walking back one L1 block at a time (see the shared
// VerifySafeHeadDatabaseMatches DSL); every recorded (L1 block -> safe L2 head)
// entry must agree down to genesis.
//
// With the default op-node CL kind L2CLOpConSequencer() is a no-op and both slots
// are op-node with SafeDB enabled (the global SafeDBPath option below), so this
// degrades to an ordinary op-node safe-head-DB parity check — a valid suite
// addition rather than an op-con-only test.
//
//	DEVSTACK_L2CL_KIND=op-con-node DEVSTACK_L2EL_KIND=op-reth \
//	RUST_BINARY_PATH_OP_CON_NODE=<opql>/target/debug/op-con-node \
//	go test -run TestOpConSequencerSafeHeadDatabaseMatches ./op-acceptance-tests/tests/opcon/
func TestOpConSequencerSafeHeadDatabaseMatches(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleChainMultiNodeNoFaultProofsWithoutP2PWithoutCheck(t,
		// Route the sequencer slot to op-con-node (no-op for other CL kinds).
		opConSequencerOpt(),
		// Under the default op-node CL kind both slots need a SafeDB to answer
		// optimism_safeHeadAtL1Block; op-con-node ignores the path (see safeDBOpt).
		safeDBOpt(),
	)

	// The op-con sequencer produces blocks and the batcher lands them on L1; both
	// the sequencer's own L1 derivation and the verifier's advance their safe heads
	// and converge on the same safe chain.
	awaitSafeConvergence(t, sys, 60)

	// The verifier's safe-head-at-L1 history must match the op-con sequencer's, which
	// serves it as the source of truth from rollup_rpc_safe_head_at_l1_history. Pin
	// the walk depth to the converged safe head (mirroring the upstream safeheaddb
	// tests): without it, a history truncated to its newest entry would still pass,
	// since the walk stops at the first missing lower entry.
	startSafeBlock := sys.L2CLB.HeadBlockRef(types.LocalSafe).Number
	sys.L2CLB.VerifySafeHeadDatabaseMatches(sys.L2CL, dsl.WithMinRequiredL2Block(startSafeBlock))
}
