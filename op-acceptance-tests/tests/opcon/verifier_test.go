package opcon

import (
	"testing"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
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
		presets.WithGlobalL2CLOption(sysgo.L2CLOptionFn(func(p devtest.T, _ sysgo.ComponentTarget, cfg *sysgo.L2CLConfig) {
			// Enable the op-node sequencer's SafeDB so it can answer
			// optimism_safeHeadAtL1Block as the source of truth. (The op-con-node
			// verifier serves its own from SQL and needs no file path.)
			cfg.SafeDBPath = p.TempDir()
		})),
	)

	// The op-node sequencer batches blocks to L1; both it and the op-con-node
	// verifier advance their safe heads (the verifier derives safe from L1 and
	// gets unsafe over P2P via the sidecar).
	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalSafe, 1, 60),
		sys.L2CLB.AdvancedFn(types.LocalSafe, 1, 60),
	)

	sys.L2CLB.InSync(sys.L2CL, types.LocalSafe, 60)

	// The verifier's safe-head-at-L1 history must match the sequencer's SafeDB.
	sys.L2CLB.VerifySafeHeadDatabaseMatches(sys.L2CL)
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

// NOTE: a verifier restart/resync variant (the op-con-node analogue of the
// restart half of the op-node safeheaddb_clsync test) is intentionally NOT added
// here yet: it surfaced a pre-existing op-con-node restore-from-checkpoint panic
// ("More than one value in subquery", an integer scalar subquery in the generated
// circuit on the restore path) that is unrelated to the sidecar P2P work. Add the
// restart variant once that restore bug is fixed.
