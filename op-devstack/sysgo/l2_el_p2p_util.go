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
// The initiator dials the acceptor and the peering is additionally registered
// as trusted on both nodes: with discovery disabled, an untrusted one-sided
// peering is unrecoverable once a session drops (the inbound side has no
// dialable address to re-dial). Trusted peers are never evicted; the re-dial
// after a drop comes from the initiator's static entry (geth) or from either
// side's trusted entry (reth).
func ConnectP2P(ctx context.Context, require *testreq.Assertions, initiator RpcCaller, acceptor RpcCaller) {
	var targetInfo p2p.NodeInfo
	require.NoError(acceptor.CallContext(ctx, &targetInfo, "admin_nodeInfo"), "get node info")
	targetNode, err := enode.ParseV4(targetInfo.Enode)
	require.NoError(err, "failed to parse target node")
	expectedID := targetNode.ID().String()

	var initiatorInfo p2p.NodeInfo
	require.NoError(initiator.CallContext(ctx, &initiatorInfo, "admin_nodeInfo"), "get initiator node info")

	// Dial first, then mark trusted: on reth, admin_addPeer overwrites the peer
	// kind with Static, so calling it after admin_addTrustedPeer would downgrade
	// the peer from Trusted back to Static.
	var peerAdded bool
	require.NoError(initiator.CallContext(ctx, &peerAdded, "admin_addPeer", targetInfo.Enode), "add peer")
	require.True(peerAdded, "should have added peer successfully")

	addTrustedPeer := func(node RpcCaller, enodeURL string, side string) {
		var added bool
		require.NoError(node.CallContext(ctx, &added, "admin_addTrustedPeer", enodeURL), "add trusted peer on "+side)
		require.True(added, "should have added trusted peer on "+side)
	}
	addTrustedPeer(initiator, targetInfo.Enode, "initiator")
	addTrustedPeer(acceptor, initiatorInfo.Enode, "acceptor")

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

	var initiatorInfo p2p.NodeInfo
	require.NoError(initiator.CallContext(ctx, &initiatorInfo, "admin_nodeInfo"), "get initiator node info")

	// Drop the trusted status set up by ConnectP2P first: reth's admin_removePeer
	// is a no-op on a trusted peer. Trusted removal alone is not enough either,
	// as it only downgrades the entry to Basic while keeping its dialable
	// address, so the peer must also be removed on both sides or the pair
	// silently reconnects shortly after.
	removeTrustedPeer := func(node RpcCaller, enodeURL string, side string) {
		var removed bool
		require.NoError(node.CallContext(ctx, &removed, "admin_removeTrustedPeer", enodeURL), "remove trusted peer on "+side)
		require.True(removed, "should have removed trusted peer on "+side)
	}
	removeTrustedPeer(initiator, targetInfo.Enode, "initiator")
	removeTrustedPeer(acceptor, initiatorInfo.Enode, "acceptor")

	removePeer := func(node RpcCaller, enr string, side string) {
		var removed bool
		require.NoError(node.CallContext(ctx, &removed, "admin_removePeer", enr), "remove peer on "+side)
		require.True(removed, "should have removed peer on "+side)
	}
	removePeer(initiator, targetInfo.ENR, "initiator")
	removePeer(acceptor, initiatorInfo.ENR, "acceptor")

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
