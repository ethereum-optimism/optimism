package node_restart

import (
	"fmt"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	node_utils "github.com/op-rs/kona/node/utils"
)

func TestSequencerRestart(gt *testing.T) {
	t := devtest.SerialT(gt)

	out := node_utils.NewMixedOpKona(t)

	nodes := out.L2CLValidatorNodes()
	sequencerNodes := out.L2CLSequencerNodes()
	t.Gate().Greater(len(nodes), 0, "expected at least one validator node")
	t.Gate().Greater(len(sequencerNodes), 0, "expected at least one sequencer node")

	sequencer := sequencerNodes[0]
	seqPeerId := sequencer.PeerInfo().PeerID

	// Let's ensure that all the nodes are properly advancing.
	var preCheckFuns []dsl.CheckFunc
	for _, node := range nodes {
		preCheckFuns = append(preCheckFuns, node.LaggedFn(&sequencer, types.CrossUnsafe, 20, true), node.AdvancedFn(types.LocalSafe, 20, 40))
	}
	dsl.CheckAll(t, preCheckFuns...)

	// Let's stop the sequencer node.
	t.Logf("Stopping sequencer %s", sequencer.Escape().ID().Key())
	sequencer.Stop()

	var stopCheckFuns []dsl.CheckFunc
	for _, node := range nodes {
		// Ensure that the node is no longer connected to the sequencer
		nodePeers := node.Peers()
		_, err := retry.Do(t.Ctx(), 5, &retry.ExponentialStrategy{Max: 10 * time.Second, Min: 1 * time.Second, MaxJitter: 250 * time.Millisecond}, func() (any, error) {
			for _, peer := range nodePeers.Peers {
				if peer.PeerID == seqPeerId {
					return nil, fmt.Errorf("expected node %s to be disconnected from sequencer %s", node.Escape().ID().Key(), sequencer.Escape().ID().Key())
				}
			}
			return nil, nil
		})
		t.Require().NoError(err)

		// Ensure that the other nodes are not advancing.
		// The local safe head may advance (for the next l1 block to be processed), but the unsafe head should not.
		stopCheckFuns = append(stopCheckFuns, node.NotAdvancedFn(types.LocalUnsafe, 50))
	}

	dsl.CheckAll(t, stopCheckFuns...)

	// Let's restart the sequencer node.
	t.Logf("Starting sequencer %s", sequencer.Escape().ID().Key())
	sequencer.Start()

	// Let's reconnect the sequencer to the nodes.
	t.Logf("Reconnecting sequencer %s to nodes", sequencer.Escape().ID().Key())
	for _, node := range nodes {
		t.Logf("Connecting sequencer %s to node %s", sequencer.Escape().ID().Key(), node.Escape().ID().Key())
		sequencer.ConnectPeer(&node)
	}

	// Let's ensure that the nodes are advancing.
	t.Logf("Waiting for nodes to advance")
	var postCheckFuns []dsl.CheckFunc
	for _, node := range nodes {
		postCheckFuns = append(postCheckFuns, node.AdvancedFn(types.LocalSafe, 10, 100), node.AdvancedFn(types.LocalUnsafe, 10, 100))
	}

	dsl.CheckAll(t, postCheckFuns...)
}
