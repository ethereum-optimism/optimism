package node

import (
	"sync"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum/go-ethereum/common"
	node_utils "github.com/op-rs/kona/node/utils"
	"github.com/stretchr/testify/require"
)

// Check that the node p2p RPC endpoints are working.
func TestP2PPeers(gt *testing.T) {
	t := devtest.SerialT(gt)

	out := node_utils.NewMixedOpKona(t)

	p2pPeersAndPeerStats(t, out)

	p2pSelfAndPeers(t, out)

	p2pBanPeer(t, out)
}

// Ensure that the `opp2p_peers` and `opp2p_self` RPC endpoints return the same information.
func p2pSelfAndPeers(t devtest.T, out *node_utils.MixedOpKonaPreset) {
	nodes := out.L2CLKonaNodes()
	var wg sync.WaitGroup
	for _, node := range nodes {
		wg.Add(1)
		go func(node *dsl.L2CLNode) {
			defer wg.Done()
			clRPC := node_utils.GetNodeRPCEndpoint(node)
			clName := node.Escape().ID().Key()

			// Gather the peers for the node.
			peers := &apis.PeerDump{}
			require.NoError(t, node_utils.SendRPCRequest(clRPC, "opp2p_peers", peers, true), "failed to send RPC request to node %s: %s", clName)

			// Check that every peer's info matches the node's info.
			for _, peer := range peers.Peers {
				// Find the node that is the peer. We loop over all the nodes in the network and try to match their peerID's to
				// the peerID we are looking for.
				for _, node := range nodes {
					// We get the peer's info.
					otherPeerInfo := &apis.PeerInfo{}
					otherCLRPC := node_utils.GetNodeRPCEndpoint(&node)
					otherCLName := node.Escape().ID().Key()
					require.NoError(t, node_utils.SendRPCRequest(otherCLRPC, "opp2p_self", otherPeerInfo), "failed to send RPC request to node %s: %s", clName)

					// These checks fail for the op-node. It seems that their p2p handler is flaky and doesn't always return the correct peer info.
					if otherPeerInfo.PeerID == peer.PeerID {
						require.Equal(t, otherPeerInfo.NodeID, peer.NodeID, "nodeID mismatch, %s", otherCLName)
						require.Equal(t, otherPeerInfo.ProtocolVersion, peer.ProtocolVersion, "protocolVersion mismatch, %s", otherCLName)

						// Sometimes the node is not part of the discovery table so we don't have an ENR.
						if peer.ENR != "" {
							require.Equal(t, otherPeerInfo.ENR, peer.ENR, "ENR mismatch, %s", otherCLName)
						}

						// Sometimes the node is not part of the discovery table so we don't have a valid chainID.
						if peer.ChainID != 0 {
							require.Equal(t, otherPeerInfo.ChainID, peer.ChainID, "chainID mismatch, %s", otherCLName)
						}

						for _, addr := range peer.Addresses {
							require.Contains(t, otherPeerInfo.Addresses, addr, "the peer's address should be in the node's known addresses, %s", otherCLName)
						}

						for _, protocol := range peer.Protocols {
							require.Contains(t, otherPeerInfo.Protocols, protocol, "protocol %s not found, %s", protocol, otherCLName)
						}

						require.Equal(t, otherPeerInfo.UserAgent, peer.UserAgent, "userAgent mismatch, %s", otherCLName)
					}
				}
			}
		}(&node)
	}
	wg.Wait()
}

// Check that the `opp2p_peers` and `opp2p_peerStats` RPC endpoints return coherent information.
func p2pPeersAndPeerStats(t devtest.T, out *node_utils.MixedOpKonaPreset) {
	nodes := out.L2CLNodes()
	var wg sync.WaitGroup
	for _, node := range nodes {
		wg.Add(1)
		go func(node *dsl.L2CLNode) {
			defer wg.Done()
			clRPC := node_utils.GetNodeRPCEndpoint(node)
			clName := node.Escape().ID().Key()

			peers := &apis.PeerDump{}
			require.NoError(t, node_utils.SendRPCRequest(clRPC, "opp2p_peers", peers, true), "failed to send RPC request to node %s: %s", clName)

			peerStats := &apis.PeerStats{}
			require.NoError(t, node_utils.SendRPCRequest(clRPC, "opp2p_peerStats", peerStats), "failed to send RPC request to node %s: %s", clName)

			require.Equal(t, peers.TotalConnected, peerStats.Connected, "totalConnected mismatch node %s", clName)
			require.Equal(t, len(peers.Peers), int(peers.TotalConnected), "peer count mismatch node %s", clName)
		}(&node)
	}
	wg.Wait()
}

func p2pBanPeer(t devtest.T, out *node_utils.MixedOpKonaPreset) {
	nodes := out.L2CLNodes()
	for _, node := range nodes {
		clRPC := node_utils.GetNodeRPCEndpoint(&node)
		clName := node.Escape().ID().Key()

		peers := &apis.PeerDump{}
		require.NoError(t, node_utils.SendRPCRequest(clRPC, "opp2p_peers", peers, true), "failed to send RPC request to node %s: %s", clName)

		connectedPeers := peers.TotalConnected

		// Try to ban a peer.
		// We pick the first peer that is connected.
		peerToBan := ""
		for _, peer := range peers.Peers {
			peerToBan = peer.PeerID.String()
			break
		}

		require.NotEmpty(t, peerToBan, "no connected peer found")

		require.NoError(t, node_utils.SendRPCRequest[any](clRPC, "opp2p_blockPeer", nil, peerToBan), "failed to send RPC request to node %s: %s", clName)

		// Check that the peer is banned.
		peersAfterBan := &apis.PeerDump{}
		require.NoError(t, node_utils.SendRPCRequest(clRPC, "opp2p_peers", peersAfterBan, true), "failed to send RPC request to node %s: %s", clName)

		require.Equal(t, connectedPeers, peersAfterBan.TotalConnected, "totalConnected mismatch node %s", clName)

		contains := false
		// Loop over all the banned peers and check that the peer is banned.
		for _, bannedPeer := range peersAfterBan.BannedPeers {
			if bannedPeer.String() == peerToBan {
				require.Equal(t, bannedPeer.String(), peerToBan, "peer %s not banned", peerToBan)
				contains = true
			}
		}

		require.True(t, contains, "peer %s not banned", peerToBan)

		// Try to unban the peer.
		require.NoError(t, node_utils.SendRPCRequest[any](clRPC, "opp2p_unblockPeer", nil, peerToBan), "failed to send RPC request to node %s: %s", clName)

		// Check that the peer is unbanned.
		peersAfterUnban := &apis.PeerDump{}
		require.NoError(t, node_utils.SendRPCRequest(clRPC, "opp2p_peers", peersAfterUnban, true), "failed to send RPC request to node %s: %s", clName)

		require.Equal(t, connectedPeers, peersAfterUnban.TotalConnected, "totalConnected mismatch node %s", clName)
		require.NotContains(t, peersAfterUnban.BannedPeers, peerToBan, "peer %s is banned", peerToBan)
	}
}

func rollupConfig(t devtest.T, node *dsl.L2CLNode) *rollup.Config {
	clRPC := node_utils.GetNodeRPCEndpoint(node)
	clName := node.Escape().ID().Key()

	rollupConfig := &rollup.Config{}
	require.NoError(t, node_utils.SendRPCRequest(clRPC, "optimism_rollupConfig", rollupConfig), "failed to send RPC request to node %s: %s", clName)

	return rollupConfig
}

func rollupConfigMatches(t devtest.T, configA *rollup.Config, configB *rollup.Config) {
	// ProtocolVersionsAddress is deprecated in kona-node while not yet removed from the op-node.
	configA.ProtocolVersionsAddress = common.Address{}
	configB.ProtocolVersionsAddress = common.Address{}

	require.Equal(t, configA, configB, "rollup config mismatch")
}

func TestRollupConfig(gt *testing.T) {
	t := devtest.ParallelT(gt)

	out := node_utils.NewMixedOpKona(t)

	rollupConfigs := make([]*rollup.Config, 0)

	for _, node := range out.L2CLNodes() {
		rollupConfigs = append(rollupConfigs, rollupConfig(t, &node))
	}

	// Check that the rollup configs are the same.
	for _, config := range rollupConfigs {
		rollupConfigMatches(t, rollupConfigs[0], config)
	}
}
