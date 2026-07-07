package opcon

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
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
