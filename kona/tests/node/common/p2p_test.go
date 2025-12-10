package node

import (
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/libp2p/go-libp2p/core/peer"
	node_utils "github.com/op-rs/kona/node/utils"
	"github.com/stretchr/testify/require"
)

func checkProtocols(t devtest.T, peer *apis.PeerInfo) {
	nodeName := peer.PeerID.String()

	require.Contains(t, peer.Protocols, "/meshsub/1.0.0", fmt.Sprintf("%s is not using the meshsub protocol 1.0.0", nodeName))
	require.Contains(t, peer.Protocols, "/meshsub/1.1.0", fmt.Sprintf("%s is not using the meshsub protocol 1.1.0", nodeName))
	require.Contains(t, peer.Protocols, "/meshsub/1.2.0", fmt.Sprintf("%s is not using the meshsub protocol 1.2.0", nodeName))
	require.Contains(t, peer.Protocols, "/ipfs/id/1.0.0", fmt.Sprintf("%s is not using the id protocol 1.0.0", nodeName))
	require.Contains(t, peer.Protocols, "/ipfs/id/push/1.0.0", fmt.Sprintf("%s is not using the id push protocol 1.0.0", nodeName))
	require.Contains(t, peer.Protocols, "/floodsub/1.0.0", fmt.Sprintf("%s is not using the floodsub protocol 1.0.0", nodeName))

}

// Check that the node has enough connected peers and peers in the discovery table.
func checkPeerStats(t devtest.T, node *dsl.L2CLNode, minConnected uint, minBlocksTopic uint) {
	peerStats, err := node.Escape().P2PAPI().PeerStats(t.Ctx())
	nodeName := node.Escape().ID()

	require.NoError(t, err, "failed to get peer stats for %s", nodeName)

	require.GreaterOrEqual(t, peerStats.Connected, minConnected, fmt.Sprintf("%s has no connected peers", nodeName))
	require.GreaterOrEqual(t, peerStats.BlocksTopic, minBlocksTopic, fmt.Sprintf("%s has no peers in the blocks topic", nodeName))
	require.GreaterOrEqual(t, peerStats.BlocksTopicV2, minBlocksTopic, fmt.Sprintf("%s has no peers in the blocks topic v2", nodeName))
	require.GreaterOrEqual(t, peerStats.BlocksTopicV3, minBlocksTopic, fmt.Sprintf("%s has no peers in the blocks topic v3", nodeName))
	require.GreaterOrEqual(t, peerStats.BlocksTopicV4, minBlocksTopic, fmt.Sprintf("%s has no peers in the blocks topic v4", nodeName))
}

// Check that `node` is connected to the other node and exposes the expected protocols.
func arePeers(t devtest.T, node *dsl.L2CLNode, otherNodeId peer.ID) {
	nodePeers := node.Peers()

	found := false
	for _, peer := range nodePeers.Peers {
		if peer.PeerID == otherNodeId {
			// TODO(@theochap): this test is flaky, we should fix it.
			// require.Equal(t, network.Connected, peer.Connectedness, fmt.Sprintf("%s is not connected to the %s", node.Escape().ID(), otherNodeId))
			checkProtocols(t, peer)
			found = true
		}
	}
	require.True(t, found, fmt.Sprintf("%s is not in the %s's peers", otherNodeId, node.Escape().ID()))
}

func TestP2PMinimal(gt *testing.T) {
	t := devtest.ParallelT(gt)

	out := node_utils.NewMixedOpKona(t)

	nodes := out.L2CLNodes()
	firstNode := nodes[0]
	secondNode := nodes[1]

	opNodeId := firstNode.PeerInfo().PeerID
	konaNodeId := secondNode.PeerInfo().PeerID

	// Wait for a few blocks to be produced.
	dsl.CheckAll(t, secondNode.ReachedFn(types.LocalUnsafe, 40, 80), firstNode.ReachedFn(types.LocalUnsafe, 40, 80))

	// Check that the nodes are connected to each other.
	arePeers(t, &firstNode, konaNodeId)
	arePeers(t, &secondNode, opNodeId)

	// Check that the nodes have enough connected peers and peers in the discovery table.
	checkPeerStats(t, &firstNode, 1, 1)
	checkPeerStats(t, &secondNode, 1, 1)
}

// Check that, for every node in the network, all the peers are connected to the expected protocols and the same chainID.
func TestP2PProtocols(gt *testing.T) {
	t := devtest.ParallelT(gt)

	out := node_utils.NewMixedOpKona(t)

	nodes := out.L2CLNodes()

	for _, node := range nodes {
		for _, peer := range node.Peers().Peers {
			checkProtocols(t, peer)
		}
	}
}

func TestP2PChainID(gt *testing.T) {
	t := devtest.ParallelT(gt)

	out := node_utils.NewMixedOpKona(t)

	nodes := out.L2CLKonaNodes()

	t.Gate().NotEmpty(nodes, "no KONA nodes found")

	chainID := nodes[0].PeerInfo().ChainID

	for _, node := range nodes {
		nodeChainID, ok := node.Escape().ID().ChainID().Uint64()
		require.True(t, ok, "chainID is too large for a uint64")
		require.Equal(t, chainID, nodeChainID, fmt.Sprintf("%s has a different chainID", node.Escape().ID()))

		for _, peer := range node.Peers().Peers {
			// Sometimes peers don't have a chainID because they are not part of the discovery table while being connected to gossip.
			if peer.ChainID != 0 {
				require.Equal(t, chainID, peer.ChainID, fmt.Sprintf("%s has a different chainID", node.Escape().ID()))
			}
		}
	}
}

// Check that all the nodes in the network have enough connected peers and peers in the discovery table.
func TestNetworkConnectivity(gt *testing.T) {
	t := devtest.ParallelT(gt)

	out := node_utils.NewMixedOpKona(t)

	nodes := out.L2CLNodes()
	numNodes := len(nodes)

	for _, node := range nodes {
		checkPeerStats(t, &node, uint(numNodes)-1, uint(numNodes)/2)
	}
}
