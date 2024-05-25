package geth

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/stretchr/testify/require"
)

// ConnectP2P creates a p2p peer connection between node1 and node2.
func ConnectP2P(t *testing.T, node1 *ethclient.Client, node2 *ethclient.Client) {
	var targetInfo p2p.NodeInfo
	require.NoError(t, node2.Client().Call(&targetInfo, "admin_nodeInfo"), "get node info")

	var peerAdded bool
	require.NoError(t, node1.Client().Call(&peerAdded, "admin_addPeer", targetInfo.Enode), "add peer")
	require.True(t, peerAdded, "should have added peer successfully")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := wait.For(ctx, time.Second, func() (bool, error) {
		var peerCount hexutil.Uint64
		if err := node1.Client().Call(&peerCount, "net_peerCount"); err != nil {
			return false, err
		}
		t.Logf("Peer count %v", uint64(peerCount))
		return peerCount >= hexutil.Uint64(1), nil
	})
	require.NoError(t, err, "wait for a peer to be connected")
}

func WithP2P() func(ethCfg *ethconfig.Config, nodeCfg *node.Config) error {
	return func(ethCfg *ethconfig.Config, nodeCfg *node.Config) error {
		ethCfg.RollupDisableTxPoolGossip = false
		nodeCfg.P2P = p2p.Config{
			NoDiscovery: true,
			ListenAddr:  "127.0.0.1:0",
			MaxPeers:    10,
		}
		return nil
	}
}
