package opcon

import (
	"testing"
	"time"

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
// the publish path: the op-con-node sequencer signs each unsafe block with a key
// that is NOT the deployed SystemConfig unsafe-block signer. The publish sidecar
// still relays the (hash-consistent but wrongly-signed) envelope onto gossip
// verbatim — it holds no signer policy — so the acceptance decision falls to the
// receiver. A stock op-node verifier must reject every such block in gossip
// signature validation, so its unsafe head must never advance even though the
// sequencer's own head does. This proves end-to-end signature attribution over
// the publish path (the property the publish path exists to carry).
//
// Only meaningful for an op-con-node sequencer (a stock op-node sequencer signs
// natively), so the test skips on other CL kinds.
func TestOpConSequencerWrongSignerRejectedViaP2P(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipUnlessOpConNode(t, "op-con-node is the sequencer under test")
	sys := presets.NewSingleChainOpConSequencerP2PWrongSignerWithoutCheck(t)

	// Local block production does not depend on gossip acceptance: the sequencer
	// advances its own unsafe head past a few blocks, so wrongly-signed blocks
	// have definitely been published to gossip.
	sys.L2CL.Advanced(types.LocalUnsafe, 3, 60)

	// The op-node verifier rejects each block's signature (wrong signer) in
	// gossip validation, so its unsafe head stays at genesis. Confirm it holds
	// across a window covering several more sequencer blocks (no batcher, no
	// follow source ⇒ gossip is its only unsafe source).
	blockTime := sys.L2Chain.Escape().RollupConfig().BlockTime
	require.Zero(t, sys.L2CLB.SyncStatus().UnsafeL2.Number,
		"verifier accepted a wrongly-signed block")
	for range 6 {
		time.Sleep(time.Duration(blockTime) * time.Second)
		require.Zero(t, sys.L2CLB.SyncStatus().UnsafeL2.Number,
			"verifier unsafe head advanced on a wrongly-signed gossip block")
	}
}
