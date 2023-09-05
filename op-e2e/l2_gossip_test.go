package op_e2e

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/stretchr/testify/require"
)

func TestTxGossip(t *testing.T) {
	InitParallel(t)
	cfg := DefaultSystemConfig(t)
	gethOpts := []GethOption{
		func(ethCfg *ethconfig.Config, nodeCfg *node.Config) error {
			ethCfg.RollupDisableTxPoolGossip = false
			nodeCfg.P2P = p2p.Config{
				NoDiscovery: true,
				ListenAddr:  "127.0.0.1:0",
				MaxPeers:    10,
			}
			return nil
		},
	}
	cfg.GethOptions["sequencer"] = gethOpts
	cfg.GethOptions["verifier"] = gethOpts
	sys, err := cfg.Start(t)
	require.NoError(t, err, "Start system")

	seqClient := sys.Clients["sequencer"]
	verifClient := sys.Clients["verifier"]
	var seqInfo p2p.NodeInfo
	require.NoError(t, seqClient.Client().Call(&seqInfo, "admin_nodeInfo"), "get sequencer node info")
	var verifInfo p2p.NodeInfo
	require.NoError(t, verifClient.Client().Call(&verifInfo, "admin_nodeInfo"), "get verifier node info")

	var peerAdded bool
	require.NoError(t, verifClient.Client().Call(&peerAdded, "admin_addPeer", seqInfo.Enode), "add peer to verifier")
	require.True(t, peerAdded, "should have added peer to verifier successfully")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err = wait.For(ctx, time.Second, func() (bool, error) {
		var peerCount hexutil.Uint64
		if err := verifClient.Client().Call(&peerCount, "net_peerCount"); err != nil {
			return false, err
		}
		t.Logf("Peer count %v", uint64(peerCount))
		return peerCount >= hexutil.Uint64(1), nil
	})
	require.NoError(t, err, "wait for a peer to be connected")

	// Send a transaction to the verifier and it should be gossiped to the sequencer and included in a block.
	SendL2Tx(t, cfg, verifClient, cfg.Secrets.Alice, func(opts *TxOpts) {
		opts.ToAddr = &common.Address{0xaa}
		opts.Value = common.Big1
		opts.VerifyOnClients(seqClient, verifClient)
	})
}
