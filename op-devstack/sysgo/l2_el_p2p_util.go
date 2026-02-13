package sysgo

import (
	"context"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/testreq"
)

type RpcCaller interface {
	CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error
}

// ConnectP2P creates a p2p peer connection between node1 and node2.
func ConnectP2P(ctx context.Context, require *testreq.Assertions, initiator RpcCaller, acceptor RpcCaller, trusted bool) {
	var targetInfo p2p.NodeInfo
	require.NoError(acceptor.CallContext(ctx, &targetInfo, "admin_nodeInfo"), "get node info")
	targetNode, err := enode.ParseV4(targetInfo.Enode)
	require.NoError(err, "failed to parse target node")
	expectedID := targetNode.ID().String()

	var initiatorInfo p2p.NodeInfo
	require.NoError(initiator.CallContext(ctx, &initiatorInfo, "admin_nodeInfo"), "get initiator node info")

	var peerAdded bool
	require.NoError(initiator.CallContext(ctx, &peerAdded, "admin_addPeer", targetInfo.Enode), "add peer")
	require.True(peerAdded, "should have added peer successfully")

	if trusted {
		var peerAddedTrusted bool
		require.NoError(initiator.CallContext(ctx, &peerAddedTrusted, "admin_addTrustedPeer", targetInfo.Enode), "add trusted peer")
		require.True(peerAddedTrusted, "should have added trusted peer successfully")
	}

	// Skip P2P connection verification if SKIP_P2P_CONNECTION_CHECK is set
	// FIXME(#18570): it seems we have some issues getting op-reth to connect to op-geth. This is a temporary workaround to ensure we can still run the
	// devstack tests.
	if os.Getenv("SKIP_P2P_CONNECTION_CHECK") != "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err = wait.For(ctx, time.Second, func() (bool, error) {
		var peers []peer
		if err := initiator.CallContext(ctx, &peers, "admin_peers"); err != nil {
			return false, err
		}
		return slices.ContainsFunc(peers, func(p peer) bool {
			peerID := strings.TrimPrefix(strings.ToLower(p.ID), "0x")
			return peerID == strings.ToLower(expectedID)
		}), nil
	})
	require.NoError(err, "The peer was not connected")
}

// DisconnectP2P disconnects a p2p peer connection between node1 and node2.
func DisconnectP2P(ctx context.Context, require *testreq.Assertions, initiator RpcCaller, acceptor RpcCaller) {
	var targetInfo p2p.NodeInfo
	require.NoError(acceptor.CallContext(ctx, &targetInfo, "admin_nodeInfo"), "get node info")
	targetNode, err := enode.ParseV4(targetInfo.Enode)
	require.NoError(err, "failed to parse target node")
	expectedID := targetNode.ID().String()

	var peerRemoved bool
	require.NoError(initiator.CallContext(ctx, &peerRemoved, "admin_removePeer", targetInfo.ENR), "add peer")
	require.True(peerRemoved, "should have removed peer successfully")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err = wait.For(ctx, time.Second, func() (bool, error) {
		var peers []peer
		if err := initiator.CallContext(ctx, &peers, "admin_peers"); err != nil {
			return false, err
		}
		return !slices.ContainsFunc(peers, func(p peer) bool {
			peerID := strings.TrimPrefix(strings.ToLower(p.ID), "0x")
			return peerID == strings.ToLower(expectedID)
		}), nil
	})
	require.NoError(err, "The peer was not removed")
}

type peer struct {
	ID string `json:"id"`
}
