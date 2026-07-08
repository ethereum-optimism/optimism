package opcon

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/stretchr/testify/require"
)

// TestOpConSequencerPublishesViaP2P is the mirror image of
// TestOpConVerifierViaP2P: op-con-node runs AS the sequencer. It builds L2
// blocks on its op-reth, signs each one with the SequencerP2PRole key, and
// multicasts the signed envelopes on its payload websocket; the op-conp2p
// sidecar subscribes to that feed and PUBLISHES every block to the OP gossip
// /blocks topic (with the node's signature — the sidecar holds no key); a stock
// op-node verifier, connected to the sidecar only over gossipsub, must accept
// the blocks (signature check against the deployed SystemConfig unsafe-block
// signer) and execute the same chain on its own op-reth.
//
//	DEVSTACK_L2CL_KIND=op-con-node \
//	DEVSTACK_L2EL_KIND=op-reth \
//	RUST_BINARY_PATH_OP_CON_NODE=<opql>/target/debug/op-con-node \
//	OP_CONP2P_BIN=<optimism>/op-conp2p-bin \
//	go test -run TestOpConSequencerPublishesViaP2P ./op-acceptance-tests/tests/opcon/
//
// Unlike the verifier variants this topology only exists for op-con-node (a
// stock op-node sequencer publishes to gossip natively), so the test skips on
// other CL kinds.
func TestOpConSequencerPublishesViaP2P(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipUnlessOpConNode(t, "op-con-node is the sequencer under test")
	sys := presets.NewSingleChainOpConSequencerP2PWithoutCheck(t)

	// The op-con-node sequencer builds and signs unsafe blocks.
	sys.L2CL.Advanced(types.LocalUnsafe, 1, 60)

	// The op-node verifier receives them purely over gossip (via the sidecar's
	// publish path) and executes the same chain.
	sys.L2CLB.Advanced(types.LocalUnsafe, 1, 60)
	sys.L2CLB.InSync(sys.L2CL, types.LocalUnsafe, 60)
}

// TestOpConSequencerFanOutViaP2P is the full op-con↔op-con gossip loop: the
// op-con-node sequencer signs and publishes one unsafe-block feed that fans out
// over gossip to two verifiers — a stock op-node (the gossip hub, proven by
// TestOpConSequencerPublishesViaP2P) AND an op-con-node verifier fronted by a
// receive sidecar (L2CLB, the assertion target here). This exercises both opql
// gossip directions at once — op-con-node signs → publish sidecar gossips →
// receive sidecar delegates the verdict back to an op-con-node verifier, which
// executes the block — from a single signed feed, with no follow source and no
// batcher (the verifier's unsafe head can only come from gossip).
//
//	DEVSTACK_L2CL_KIND=op-con-node \
//	DEVSTACK_L2EL_KIND=op-reth \
//	RUST_BINARY_PATH_OP_CON_NODE=<opql>/target/debug/op-con-node \
//	OP_CONP2P_BIN=<optimism>/op-conp2p-bin \
//	go test -run TestOpConSequencerFanOutViaP2P ./op-acceptance-tests/tests/opcon/
//
// This topology only exists for op-con-node, so the test skips on other CL kinds.
func TestOpConSequencerFanOutViaP2P(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipUnlessOpConNode(t, "op-con-node is the sequencer under test")
	sys := presets.NewSingleChainOpConSequencerFanOutP2PWithoutCheck(t)

	// The op-con-node sequencer builds and signs unsafe blocks.
	sys.L2CL.Advanced(types.LocalUnsafe, 1, 60)

	// The op-con-node verifier, fed only by its receive sidecar over gossip,
	// executes the same chain and stays in sync — the full op-con↔op-con loop.
	sys.L2CLB.Advanced(types.LocalUnsafe, 1, 60)
	sys.L2CLB.InSync(sys.L2CL, types.LocalUnsafe, 60)
}

// TestOpConSequencerWrongSignerRejectedViaP2P is the negative security case for
// the publish path: an op-con-node sequencer signs each unsafe block with a key
// that is NOT the canonical (deployed SystemConfig) unsafe-block signer, and
// those blocks must NOT propagate to a verifier over gossip.
//
// The rejection happens at the source, not at the far verifier. The op-conp2p
// publish sidecar delegates its gossipsub validator to the sequencer's own
// admin_verifyUnsafePayload, and go-libp2p-pubsub runs that validator even on
// locally-published messages — so the sequencer, which recognizes only the
// canonical signer, rejects its own wrongly-signed block and the sidecar never
// forwards it. (The sidecar cannot be made to forward it by pointing the
// sequencer at the wrong signer: the sequencer resolves the canonical signer
// from L1 derivation regardless of the fallback env.) The op-node verifier
// therefore never receives the block, and its unsafe head — for which gossip is
// the only source here (no batcher, no follow source) — stays at genesis. This
// proves the publish path refuses to gossip a non-canonically-signed block; a
// regression that let the sidecar propagate it would advance the verifier.
//
// Only meaningful for an op-con-node sequencer (a stock op-node sequencer signs
// natively), so the test skips on other CL kinds.
func TestOpConSequencerWrongSignerRejectedViaP2P(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipUnlessOpConNode(t, "op-con-node is the sequencer under test")
	sys := presets.NewSingleChainOpConSequencerP2PWrongSignerWithoutCheck(t)

	// Local block production does not depend on gossip acceptance: the sequencer
	// builds and signs (with the wrong key) a run of unsafe blocks, advancing its
	// OWN head, and attempts to publish each over gossip. Use its advance as the
	// clock (a positive DSL wait, not a fixed sleep) so many wrongly-signed blocks
	// have been produced and offered to the publish path.
	sys.L2CL.Advanced(types.LocalUnsafe, 8, 60)

	// None of those blocks are propagated (the publish sidecar's validator rejects
	// each at the source), so the op-node verifier never receives one and its
	// unsafe head never leaves genesis. Unsafe heads never regress, so a single
	// check after the sequencer has produced many blocks catches any propagation.
	require.Zero(t, sys.L2CLB.SyncStatus().UnsafeL2.Number,
		"a wrongly-signed block propagated over gossip to the verifier")
}
