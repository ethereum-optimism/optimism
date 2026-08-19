package sysgo

import (
	"context"
	"errors"
	"fmt"
	"net"
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

	waitCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err = forPeerList(waitCtx, initiator, func(peers []peer) bool {
		return slices.ContainsFunc(peers, func(p peer) bool {
			peerID := strings.TrimPrefix(strings.ToLower(p.ID), "0x")
			return peerID == strings.ToLower(expectedID)
		})
	})
	require.NoError(err, "The peer was not connected")

	// The acceptor's trusted entry is added only once the session is up: reth
	// dials trusted peers immediately, and two crossed in-flight dials are both
	// torn down with AlreadyConnected (reth has no simultaneous-connect
	// tie-break). Against a live session the trusted add only upgrades the
	// session's peer kind.
	addTrustedPeer(acceptor, initiatorInfo.Enode, "acceptor")
}

// forPeerList polls admin_peers on node until cond holds for the returned peer
// list. Each poll is individually time-bounded so that a single stalled call —
// e.g. the RPC client re-dialing a connection the kernel never answers — cannot
// consume the entire polling budget (a fresh attempt dials from a new source
// port and typically succeeds). Only attempt timeouts are retried; any other
// error still fails fast.
func forPeerList(ctx context.Context, node RpcCaller, cond func([]peer) bool) error {
	return pollPeerList(ctx, node, time.Second, 5*time.Second, cond)
}

// pollPeerList is forPeerList with the poll interval and per-attempt timeout
// made explicit for tests.
func pollPeerList(ctx context.Context, node RpcCaller, interval, attemptTimeout time.Duration, cond func([]peer) bool) error {
	var lastTimeout error
	err := wait.For(ctx, interval, func() (bool, error) {
		attemptCtx, cancel := context.WithTimeout(ctx, attemptTimeout)
		defer cancel()
		var peers []peer
		if err := node.CallContext(attemptCtx, &peers, "admin_peers"); err != nil {
			// Retry only genuine attempt timeouts: the per-attempt deadline must
			// have expired while the outer budget is still live, and the error
			// itself must be a timeout — the RPC client can return an unrelated
			// error while the attempt deadline happens to be expired, and that
			// must still fail fast.
			if attemptCtx.Err() != nil && ctx.Err() == nil && isTimeout(err) {
				lastTimeout = err
				return false, nil
			}
			return false, err
		}
		return cond(peers), nil
	})
	// Surface the retained attempt timeout only when the outer budget itself
	// expired: a terminal error can be a DeadlineExceeded from inside the RPC
	// client while the outer context is still live, and must not have a stale
	// prior attempt timeout attached.
	if err != nil && lastTimeout != nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
		err = fmt.Errorf("%w (last admin_peers attempt error: %w)", err, lastTimeout)
	}
	return err
}

func isTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
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
	err = forPeerList(ctx, initiator, func(peers []peer) bool {
		return !slices.ContainsFunc(peers, func(p peer) bool {
			peerID := strings.TrimPrefix(strings.ToLower(p.ID), "0x")
			return peerID == strings.ToLower(expectedID)
		})
	})
	require.NoError(err, "The peer was not removed")
}

type peer struct {
	ID string `json:"id"`
}
