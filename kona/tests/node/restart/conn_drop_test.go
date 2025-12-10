package node_restart

import (
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	node_utils "github.com/op-rs/kona/node/utils"
)

// Ensure that kona-nodes reconnect to the sequencer and sync properly when the connection is dropped.
func TestConnDropSync(gt *testing.T) {
	t := devtest.SerialT(gt)

	out := node_utils.NewMixedOpKona(t)

	nodes := out.L2CLValidatorNodes()
	sequencerNodes := out.L2CLSequencerNodes()
	t.Gate().Greater(len(nodes), 0, "expected at least one validator node")
	t.Gate().Greater(len(sequencerNodes), 0, "expected at least one sequencer node")

	// Ensure that the nodes are advancing.
	var preCheckFuns []dsl.CheckFunc
	for _, node := range out.L2CLNodes() {
		preCheckFuns = append(preCheckFuns, node.AdvancedFn(types.LocalSafe, 20, 100), node.AdvancedFn(types.LocalUnsafe, 20, 100))
	}
	dsl.CheckAll(t, preCheckFuns...)

	sequencer := sequencerNodes[0]

	var postDisconnectCheckFuns []dsl.CheckFunc
	for _, node := range nodes {
		clName := node.Escape().ID().Key()

		node.DisconnectPeer(&sequencer)

		// Ensure that the node is no longer connected to the sequencer
		t.Logf("node %s is disconnected from sequencer %s", clName, sequencer.Escape().ID().Key())
		seqPeers := sequencer.Peers()
		for _, peer := range seqPeers.Peers {
			t.Require().NotEqual(peer.PeerID, node.PeerInfo().PeerID, "expected node %s to be disconnected from sequencer %s", clName, sequencer.Escape().ID().Key())
		}

		peers := node.Peers()
		for _, peer := range peers.Peers {
			t.Require().NotEqual(peer.PeerID, sequencer.PeerInfo().PeerID, "expected node %s to be disconnected from sequencer %s", clName, sequencer.Escape().ID().Key())
		}

		currentUnsafeHead := node.ChainSyncStatus(node.ChainID(), types.LocalUnsafe)

		endSignal := make(chan struct{})

		safeHeads := node_utils.GetKonaWsAsync(t, &node, "safe_head", endSignal)
		unsafeHeads := node_utils.GetKonaWsAsync(t, &node, "unsafe_head", endSignal)

		// Ensures that....
		// - the node's safe head is advancing and eventually catches up with the unsafe head
		// - the node's unsafe head is NOT advancing during this time
		check := func() error {
		outer_loop:
			for {
				select {
				case safeHead := <-safeHeads:
					t.Logf("node %s safe head is advancing", clName)
					if safeHead.Number >= currentUnsafeHead.Number {
						t.Logf("node %s safe head caught up with unsafe head", clName)
						break outer_loop
					}
				case unsafeHead := <-unsafeHeads:
					return fmt.Errorf("node %s unsafe head is advancing: %d", clName, unsafeHead.Number)
				}
			}

			endSignal <- struct{}{}

			return nil
		}

		// Check that...
		// - the node's safe head is advancing
		// - the node's unsafe head is advancing (through consolidation)
		// - the node's safe head's number is catching up with the unsafe head's number
		// - the node's unsafe head is strictly lagging behind the sequencer's unsafe head
		postDisconnectCheckFuns = append(postDisconnectCheckFuns, node.AdvancedFn(types.LocalSafe, 50, 200), node.AdvancedFn(types.LocalUnsafe, 50, 200), check)
	}

	postDisconnectCheckFuns = append(postDisconnectCheckFuns, sequencer.AdvancedFn(types.LocalUnsafe, 50, 200))

	dsl.CheckAll(t, postDisconnectCheckFuns...)

	var postReconnectCheckFuns []dsl.CheckFunc
	for _, node := range nodes {
		clName := node.Escape().ID().Key()

		node.ConnectPeer(&sequencer)

		// Check that the node is connected to the reference node
		peers := node.Peers()
		t.Require().Greater(len(peers.Peers), 0, "expected at least one peer")

		// Check that there is at least a peer with the same ID as the ref node
		found := false
		for _, peer := range peers.Peers {
			if peer.PeerID == sequencer.PeerInfo().PeerID {
				t.Logf("node %s is connected to reference node %s", clName, sequencer.Escape().ID().Key())
				found = true
				break
			}
		}

		t.Require().True(found, "expected node %s to be connected to reference node %s", clName, sequencer.Escape().ID().Key())

		// Check that the node is resyncing with the unsafe head network
		postReconnectCheckFuns = append(postReconnectCheckFuns, node_utils.MatchedWithinRange(t, node, sequencer, 3, types.LocalSafe, 50), node.AdvancedFn(types.LocalUnsafe, 50, 100), node_utils.MatchedWithinRange(t, node, sequencer, 3, types.LocalUnsafe, 100))
	}

	dsl.CheckAll(t, postReconnectCheckFuns...)
}
