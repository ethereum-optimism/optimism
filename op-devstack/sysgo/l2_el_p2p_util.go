package sysgo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"slices"
	"strconv"
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
	targetInfo, expectedID := nodeInfo(ctx, require, acceptor)
	initiatorInfo, _ := nodeInfo(ctx, require, initiator)

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

	require.NoError(waitForPeerConnected(ctx, initiator, expectedID), "The peer was not connected")

	// The acceptor's trusted entry is added only once the session is up: reth
	// dials trusted peers immediately, and two crossed in-flight dials are both
	// torn down with AlreadyConnected (reth has no simultaneous-connect
	// tie-break). Against a live session the trusted add only upgrades the
	// session's peer kind.
	addTrustedPeer(acceptor, initiatorInfo.Enode, "acceptor")
}

// ConnectP2PBasic creates a p2p peer connection between initiator and acceptor without
// registering it as trusted on either side, so the session stays subject to reputation
// penalties, bans and eviction like a discovered peer's. The initiator keeps the acceptor as a
// static entry and re-dials it after the session drops; the acceptor keeps nothing. ConnectP2P
// is for a peering that must survive whatever the test does; this is for a test that observes
// peer management, which a trusted peering masks.
func ConnectP2PBasic(ctx context.Context, require *testreq.Assertions, initiator RpcCaller, acceptor RpcCaller) {
	targetInfo, expectedID := nodeInfo(ctx, require, acceptor)

	var peerAdded bool
	require.NoError(initiator.CallContext(ctx, &peerAdded, "admin_addPeer", targetInfo.Enode), "add peer")
	require.True(peerAdded, "should have added peer successfully")

	require.NoError(waitForPeerConnected(ctx, initiator, expectedID), "The peer was not connected")

	require.False(PeerInfo(ctx, require, initiator, acceptor).Trusted, "the initiator must not hold the acceptor as a trusted peer")
	require.False(PeerInfo(ctx, require, acceptor, initiator).Trusted, "the acceptor must not hold the initiator as a trusted peer")
}

// WaitP2PConnected waits until node lists remote among its connected peers, and fails when it
// does not within the peer wait budget.
func WaitP2PConnected(ctx context.Context, require *testreq.Assertions, node RpcCaller, remote RpcCaller) {
	require.NoError(waitForPeerConnected(ctx, node, nodeID(ctx, require, remote)), "The peer is not connected")
}

// ELPeerInfo is what an execution-layer node reports about one of its peers through admin_peers.
type ELPeerInfo struct {
	// Caps are the capabilities the peer advertised in its hello, such as "eth/69".
	Caps []string
	// EthVersion is the eth protocol version the session negotiated, 0 while the eth handshake
	// is still pending.
	EthVersion uint64
	ID         string
	Inbound    bool
	// LocalAddress and RemoteAddress are the TCP endpoints of the session. Every dial comes from
	// a fresh ephemeral port, so a session that was dropped and re-dialed shows a new
	// RemoteAddress on the accepting side and a new LocalAddress on the dialing side.
	LocalAddress  string
	RemoteAddress string
	Static        bool
	Trusted       bool
}

// PeerInfo waits until node lists remote with a negotiated eth session and returns what node
// reports about it. It fails when the session is not up within the peer wait budget.
func PeerInfo(ctx context.Context, require *testreq.Assertions, node RpcCaller, remote RpcCaller) ELPeerInfo {
	expectedID := nodeID(ctx, require, remote)
	found, err := waitForPeer(ctx, node, expectedID, peer.negotiated)
	require.NoError(err, "The peer %s has no negotiated eth session", expectedID)
	return found.info()
}

// ListPeers returns what node reports about every peer it is connected to, including those
// whose eth handshake is still pending.
func ListPeers(ctx context.Context, node RpcCaller) ([]ELPeerInfo, error) {
	callCtx, cancel := context.WithTimeout(ctx, peerCallTimeout)
	defer cancel()
	var peers []peer
	if err := node.CallContext(callCtx, &peers, "admin_peers"); err != nil {
		return nil, fmt.Errorf("listing peers: %w", err)
	}
	infos := make([]ELPeerInfo, 0, len(peers))
	for _, p := range peers {
		infos = append(infos, p.info())
	}
	return infos, nil
}

// HighestEthVersion returns the highest eth protocol version among caps, and an error when caps
// carries none.
func HighestEthVersion(caps []string) (uint64, error) {
	versions, err := ethVersions(caps)
	if err != nil {
		return 0, err
	}
	if len(versions) == 0 {
		return 0, fmt.Errorf("no eth capability in %v", caps)
	}
	return slices.Max(versions), nil
}

// HighestCommonEthVersion returns the highest eth protocol version both caps and otherCaps carry,
// which is the version two peers advertising them negotiate, and an error when they share none.
func HighestCommonEthVersion(caps, otherCaps []string) (uint64, error) {
	versions, err := ethVersions(caps)
	if err != nil {
		return 0, err
	}
	otherVersions, err := ethVersions(otherCaps)
	if err != nil {
		return 0, err
	}
	common := slices.DeleteFunc(versions, func(version uint64) bool { return !slices.Contains(otherVersions, version) })
	if len(common) == 0 {
		return 0, fmt.Errorf("no common eth capability between %v and %v", caps, otherCaps)
	}
	return slices.Max(common), nil
}

// ethVersions parses the eth protocol versions out of caps, ignoring other capabilities.
func ethVersions(caps []string) ([]uint64, error) {
	var versions []uint64
	for _, capability := range caps {
		versionText, ok := strings.CutPrefix(capability, "eth/")
		if !ok {
			continue
		}
		version, err := strconv.ParseUint(versionText, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing capability %q: %w", capability, err)
		}
		versions = append(versions, version)
	}
	return versions, nil
}

// nodeInfo returns the node's admin_nodeInfo and its identity as admin_peers lists it on its
// peers.
func nodeInfo(ctx context.Context, require *testreq.Assertions, node RpcCaller) (p2p.NodeInfo, string) {
	var info p2p.NodeInfo
	require.NoError(node.CallContext(ctx, &info, "admin_nodeInfo"), "get node info")
	parsed, err := enode.ParseV4(info.Enode)
	require.NoError(err, "failed to parse node enode")
	return info, parsed.ID().String()
}

func nodeID(ctx context.Context, require *testreq.Assertions, node RpcCaller) string {
	_, id := nodeInfo(ctx, require, node)
	return id
}

// waitForPeerConnected polls node's admin_peers until it lists expectedID.
func waitForPeerConnected(ctx context.Context, node RpcCaller, expectedID string) error {
	_, err := waitForPeer(ctx, node, expectedID, func(peer) bool { return true })
	return err
}

// waitForPeer polls node's admin_peers until it lists expectedID with an entry accept takes, and
// returns that entry.
func waitForPeer(ctx context.Context, node RpcCaller, expectedID string, accept func(peer) bool) (peer, error) {
	waitCtx, cancel := context.WithTimeout(ctx, peerWaitTimeout)
	defer cancel()
	var found peer
	err := forPeerList(waitCtx, node, func(peers []peer) bool {
		for _, p := range peers {
			if p.idMatches(expectedID) && accept(p) {
				found = p
				return true
			}
		}
		return false
	})
	return found, err
}

// forPeerList polls admin_peers on node until cond holds for the returned peer
// list. Each poll is individually time-bounded so that a single stalled call —
// e.g. the RPC client re-dialing a connection the kernel never answers — cannot
// consume the entire polling budget (a fresh attempt dials from a new source
// port and typically succeeds). Only attempt timeouts are retried; any other
// error still fails fast.
func forPeerList(ctx context.Context, node RpcCaller, cond func([]peer) bool) error {
	return pollPeerList(ctx, node, time.Second, peerCallTimeout, cond)
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
	targetInfo, expectedID := nodeInfo(ctx, require, acceptor)
	initiatorInfo, _ := nodeInfo(ctx, require, initiator)

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

	waitCtx, cancel := context.WithTimeout(ctx, peerWaitTimeout)
	defer cancel()
	err := forPeerList(waitCtx, initiator, func(peers []peer) bool {
		return !slices.ContainsFunc(peers, func(p peer) bool { return p.idMatches(expectedID) })
	})
	require.NoError(err, "The peer was not removed")
}

const (
	// peerCallTimeout bounds one admin call.
	peerCallTimeout = 5 * time.Second
	// peerWaitTimeout bounds a wait for a peer list to reach a state.
	peerWaitTimeout = 30 * time.Second
)

// peer is an admin_peers entry as both op-geth and op-reth report it.
type peer struct {
	Caps    []string `json:"caps"`
	ID      string   `json:"id"`
	Network struct {
		Inbound       bool   `json:"inbound"`
		LocalAddress  string `json:"localAddress"`
		RemoteAddress string `json:"remoteAddress"`
		Static        bool   `json:"static"`
		Trusted       bool   `json:"trusted"`
	} `json:"network"`
	Protocols struct {
		// Eth is an object with the negotiated version once the eth handshake is done; op-geth
		// reports a string such as "handshake" until then.
		Eth json.RawMessage `json:"eth"`
	} `json:"protocols"`
}

func (p peer) idMatches(expectedID string) bool {
	return strings.TrimPrefix(strings.ToLower(p.ID), "0x") == strings.ToLower(expectedID)
}

// ethVersion returns the negotiated eth version, 0 while the handshake is pending.
func (p peer) ethVersion() uint64 {
	var eth struct {
		Version uint64 `json:"version"`
	}
	if err := json.Unmarshal(p.Protocols.Eth, &eth); err != nil {
		return 0
	}
	return eth.Version
}

func (p peer) negotiated() bool {
	return p.ethVersion() != 0
}

func (p peer) info() ELPeerInfo {
	return ELPeerInfo{
		Caps:          p.Caps,
		EthVersion:    p.ethVersion(),
		ID:            p.ID,
		Inbound:       p.Network.Inbound,
		LocalAddress:  p.Network.LocalAddress,
		RemoteAddress: p.Network.RemoteAddress,
		Static:        p.Network.Static,
		Trusted:       p.Network.Trusted,
	}
}
