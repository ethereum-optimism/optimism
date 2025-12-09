package node_restart

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	node_utils "github.com/op-rs/kona/node/utils"
)

// Ensure that kona-nodes reconnect to the sequencer and sync properly when the connection is dropped.
func TestRestartSync(gt *testing.T) {
	t := devtest.SerialT(gt)

	out := node_utils.NewMixedOpKona(t)

	nodes := out.L2CLValidatorNodes()
	sequencerNodes := out.L2CLSequencerNodes()
	t.Gate().Greater(len(nodes), 0, "expected at least one validator node")
	t.Gate().Greater(len(sequencerNodes), 0, "expected at least one sequencer node")

	sequencer := sequencerNodes[0]

	// Ensure that the nodes are advancing.
	var preCheckFuns []dsl.CheckFunc
	for _, node := range out.L2CLNodes() {
		preCheckFuns = append(preCheckFuns, node.AdvancedFn(types.LocalSafe, 20, 100), node.AdvancedFn(types.LocalUnsafe, 20, 100))
	}
	dsl.CheckAll(t, preCheckFuns...)

	for _, node := range nodes {
		t.Logf("testing restarts for node %s", node.Escape().ID().Key())
		clName := node.Escape().ID().Key()
		nodePeerId := node.PeerInfo().PeerID

		t.Logf("stopping node %s", clName)
		node.Stop()

		// Ensure that the node is no longer connected to the sequencer
		// Retry with an exponential backoff because the node may take a few seconds to stop.
		_, err := retry.Do(t.Ctx(), 5, &retry.ExponentialStrategy{Max: 10 * time.Second, Min: 1 * time.Second, MaxJitter: 250 * time.Millisecond}, func() (any, error) {
			seqPeers := sequencer.Peers()
			for _, peer := range seqPeers.Peers {
				if peer.PeerID == nodePeerId {
					return nil, fmt.Errorf("expected node %s to be disconnected from sequencer %s", clName, sequencer.Escape().ID().Key())
				}
			}
			return nil, nil
		})

		t.Require().NoError(err)

		// Ensure that the node is stopped
		// Check that calling any rpc method returns an error
		rpc := node_utils.GetNodeRPCEndpoint(&node)
		var out *eth.SyncStatus
		err = rpc.CallContext(context.Background(), &out, "opp2p_syncStatus")
		t.Require().Error(err, "expected node %s to be stopped", clName)
	}

	sequencer.Advanced(types.LocalUnsafe, 50, 200)

	var postStartCheckFuns []dsl.CheckFunc
	for _, node := range nodes {
		clName := node.Escape().ID().Key()
		t.Logf("starting node %s", clName)
		node.Start()

		node.ConnectPeer(&sequencer)

		// Check that the node is resyncing with the network
		postStartCheckFuns = append(postStartCheckFuns, node_utils.MatchedWithinRange(t, node, sequencer, 3, types.LocalSafe, 100), node_utils.MatchedWithinRange(t, node, sequencer, 3, types.LocalUnsafe, 100))

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
	}

	dsl.CheckAll(t, postStartCheckFuns...)
}
