package opcon

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// These tests put op-con-node in BOTH roles: an op-con-node sequencer produces
// the unsafe chain, and an op-con-node verifier tracks it. They differ only in
// how the verifier receives the unsafe chain — pull (follow mode over the
// sequencer's L2 execution RPC) vs push (the sequencer's signed-payload
// websocket) — while the safe chain always derives from L1.
//
// The op-con sequencer slot is opted in with sysgo.L2CLOpConSequencer() (a no-op
// under the default op-node CL kind, so each test doubles as an ordinary
// op-node-sequencer follow check). Run against op-con-node with:
//
//	DEVSTACK_L2CL_KIND=op-con-node DEVSTACK_L2EL_KIND=op-reth \
//	RUST_BINARY_PATH_OP_CON_NODE=<opql>/target/debug/op-con-node \
//	go test -run TestOpConSequencerFollowed ./op-acceptance-tests/tests/opcon/

// TestOpConSequencerFollowedByVerifier is the op-con-sequencer flip of
// TestOpConVerifierFollowMode: an op-con-node verifier tracks an op-con-node
// SEQUENCER's unsafe head via follow mode (--follow-el pointed at the
// sequencer's L2 EL) while deriving its safe chain from L1, and CurrentL1
// tracks the sequencer. Both nodes are op-con-node — the whole chain, from
// block production to follow-mode consumption, is opql.
func TestOpConSequencerFollowedByVerifier(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleChainMultiNodeNoFaultProofsWithoutP2PWithoutCheck(t,
		presets.WithGlobalL2CLOption(sysgo.L2CLOpConSequencer()))

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

// TestOpConSequencerFollowedByVerifierFinalized extends the above with the
// FINALIZED head, the op-con-sequencer flip of TestOpConVerifierFollowModeFinalized:
// the verifier's unsafe (follow mode), safe (L1 derivation), AND finalized (L1
// finality) heads must all advance and converge with the op-con-node sequencer.
// Both nodes run --l1-finalized-guard=required so op-con-node tracks L1 finality
// (the devstack default is disabled); the finalized InSync assertion genuinely
// needs the guard on BOTH nodes — with it disabled the sequencer's finalized
// head never advances, so it cannot converge with the verifier's.
//
// (Regression note: an op-con-node running AS a sequencer with
// --l1-finalized-guard=required used to crash at startup — "L1 source supervisor
// failed during startup: missing i64 field chain_id" — because
// ensure_rollup_config (runtime/l1_source.rs) mistook the unseeded global-aggregate
// rollup_config row, whose null chain_id column arrow-json omits, for a seeded
// config and took the verify path. Fixed by seeding when chain_id is absent. This
// test guards that path; the plain TestOpConSequencerFollowedByVerifier does not
// exercise the guard.)
func TestOpConSequencerFollowedByVerifierFinalized(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleChainMultiNodeNoFaultProofsWithoutP2PWithoutCheck(t,
		presets.WithGlobalL2CLOption(sysgo.L2CLOpConSequencer()),
		presets.WithGlobalL2CLOption(sysgo.L2CLOptionFn(func(_ devtest.T, _ sysgo.ComponentTarget, cfg *sysgo.L2CLConfig) {
			cfg.OpConL1FinalizedGuard = "required"
		})),
	)
	logger := t.Logger()

	target := uint64(3)
	// L1 finalization takes ~2 minutes in the devstack; ReachedFn/InSync poll
	// every 2s, so give the finalized head a generous budget.
	attempts := 90

	for _, lvl := range []types.SafetyLevel{types.LocalUnsafe, types.LocalSafe, types.Finalized} {
		dsl.CheckAll(t,
			sys.L2CL.ReachedFn(lvl, target, attempts),
			sys.L2CLB.ReachedFn(lvl, target, attempts),
		)
		sys.L2CLB.InSync(sys.L2CL, lvl, attempts)
		logger.Info("head reached + in sync", "level", lvl, "target", target)
	}

	dsl.CheckAll(t, sys.L2CLB.CurrentL1MatchedFn(sys.L2CL, 20))
}

// TestOpConSequencerVerifiedViaSignedPayloadWS exercises the sidecar-less opql
// distribution path: the op-con-node sequencer signs each unsafe block and
// serves the signed envelopes on its payload websocket; an op-con-node verifier
// consumes that feed directly via --follow (the push analog of
// follow mode), verifies each block's signature against the deployed
// SystemConfig unsafe-block signer, and ingests it — no gossip, no sidecar. The
// batcher runs, so the verifier also derives its safe chain from L1 and must
// consolidate it against the WS-fed unsafe chain.
//
// This is the acceptance test for the WS multicast + follow path (Phase 4b/4c),
// previously only validated by hand. Only meaningful for op-con-node (both the
// feed and the follower are opql-specific), so it skips on other CL kinds.
func TestOpConSequencerVerifiedViaSignedPayloadWS(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipUnlessOpConNode(t, "op-con-node signed-payload websocket feed + follower")
	sys := presets.NewSingleChainOpConSequencerWSFollowWithoutCheck(t)

	// The op-con-node sequencer signs and serves unsafe blocks; the verifier
	// receives them over the websocket and executes the same unsafe chain.
	dsl.CheckAll(t,
		sys.L2CL.ReachedFn(types.LocalUnsafe, 3, 60),
		sys.L2CLB.ReachedFn(types.LocalUnsafe, 3, 60),
	)
	sys.L2CLB.InSync(sys.L2CL, types.LocalUnsafe, 60)

	// The batcher lands the sequencer's blocks on L1; both safe heads advance and
	// converge (the verifier derives safe from L1, consolidating it against the
	// WS-fed unsafe chain).
	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalSafe, 1, 60),
		sys.L2CLB.AdvancedFn(types.LocalSafe, 1, 60),
	)
	sys.L2CLB.InSync(sys.L2CL, types.LocalSafe, 60)
}
