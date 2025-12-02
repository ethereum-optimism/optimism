package gating

import (
	"errors"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/p2p/store"
	"github.com/ethereum-optimism/optimism/op-service/clock"
)

type UnbanMetrics interface {
	RecordPeerUnban()
	RecordIPUnban()
}

//go:generate mockery --name ExpiryStore --output mocks/ --with-expecter=true
type ExpiryStore interface {
	store.IPBanStore
	store.PeerBanStore
}

// ExpiryConnectionGater enhances a BlockingConnectionGater by implementing ban-expiration
type ExpiryConnectionGater struct {
	BlockingConnectionGater
	store ExpiryStore
	log   log.Logger
	clock clock.Clock
	m     UnbanMetrics
}

func AddBanExpiry(gater BlockingConnectionGater, store ExpiryStore, log log.Logger, clock clock.Clock, m UnbanMetrics) *ExpiryConnectionGater {
	return &ExpiryConnectionGater{
		BlockingConnectionGater: gater,
		store:                   store,
		log:                     log,
		clock:                   clock,
		m:                       m,
	}
}

func (g *ExpiryConnectionGater) UnblockPeer(p peer.ID) error {
	if err := g.BlockingConnectionGater.UnblockPeer(p); err != nil {
		log.Warn("failed to unblock peer from underlying gater", "method", "UnblockPeer", "peer_id", p, "err", err)
		return err
	}
	if err := g.store.SetPeerBanExpiration(p, time.Time{}); err != nil {
		log.Warn("failed to unblock peer from expiry gater", "method", "UnblockPeer", "peer_id", p, "err", err)
		return err
	}
	g.m.RecordPeerUnban()
	return nil
}

func (g *ExpiryConnectionGater) peerBanExpiryCheck(p peer.ID) (allow bool) {
	// if the peer is blocked, check if it's time to unblock
	expiry, err := g.store.GetPeerBanExpiration(p)
	if errors.Is(err, store.ErrUnknownBan) {
		return true // peer is allowed if it has not been banned
	}
	if err != nil {
		g.log.Warn("failed to load peer-ban expiry time", "method", "peerBanExpiryCheck", "peer_id", p, "err", err)
		return false
	}
	if g.clock.Now().Before(expiry) {
		return false
	}
	g.log.Info("peer-ban expired, unbanning peer", "peer_id", p, "expiry", expiry)
	if err := g.store.SetPeerBanExpiration(p, time.Time{}); err != nil {
		g.log.Warn("failed to unban peer", "method", "peerBanExpiryCheck", "peer_id", p, "err", err)
		return false // if we ignored the error, then the inner connection-gater would drop them
	}
	g.m.RecordPeerUnban()
	return true
}

func (g *ExpiryConnectionGater) addrBanExpiryCheck(ma multiaddr.Multiaddr) (allow bool) {
	ip, err := manet.ToIP(ma)
	if err != nil {
		g.log.Error("tried to check multi-addr with bad IP", "method", "addrBanExpiryCheck", "addr", ma)
		return false
	}
	// if just the IP is blocked, check if it's time to unblock
	expiry, err := g.store.GetIPBanExpiration(ip)
	if errors.Is(err, store.ErrUnknownBan) {
		return true // IP is allowed if it has not been banned
	}
	if err != nil {
		g.log.Warn("failed to load IP-ban expiry time", "method", "addrBanExpiryCheck", "ip", ip, "err", err)
		return false
	}
	if g.clock.Now().Before(expiry) {
		return false
	}
	g.log.Info("IP-ban expired, unbanning IP", "method", "addrBanExpiryCheck", "ip", ip, "expiry", expiry)
	if err := g.store.SetIPBanExpiration(ip, time.Time{}); err != nil {
		g.log.Warn("failed to unban IP", "method", "addrBanExpiryCheck", "ip", ip, "err", err)
		return false // if we ignored the error, then the inner connection-gater would drop them
	}
	g.m.RecordIPUnban()
	return true
}

func (g *ExpiryConnectionGater) InterceptPeerDial(p peer.ID) (allow bool) {
	if !g.BlockingConnectionGater.InterceptPeerDial(p) {
		return false
	}
	peerBan := g.peerBanExpiryCheck(p)
	if !peerBan {
		log.Warn("peer is temporarily banned", "method", "InterceptPeerDial", "peer_id", p)
	}
	return peerBan
}

func (g *ExpiryConnectionGater) InterceptAddrDial(id peer.ID, ma multiaddr.Multiaddr) (allow bool) {
	if !g.BlockingConnectionGater.InterceptAddrDial(id, ma) {
		return false
	}
	peerBan := g.peerBanExpiryCheck(id)
	if !peerBan {
		log.Warn("peer id is temporarily banned", "method", "InterceptAddrDial", "peer_id", id, "multi_addr", ma)
		return false
	}
	addrBan := g.addrBanExpiryCheck(ma)
	if !addrBan {
		log.Warn("peer address is temporarily banned", "method", "InterceptAddrDial", "peer_id", id, "multi_addr", ma)
		return false
	}
	return true
}

func (g *ExpiryConnectionGater) InterceptAccept(mas network.ConnMultiaddrs) (allow bool) {
	if !g.BlockingConnectionGater.InterceptAccept(mas) {
		return false
	}
	addrBan := g.addrBanExpiryCheck(mas.RemoteMultiaddr())
	if !addrBan {
		log.Warn("peer address is temporarily banned", "method", "InterceptAccept", "multi_addr", mas.RemoteMultiaddr())
	}
	return addrBan
}

func (g *ExpiryConnectionGater) InterceptSecured(direction network.Direction, id peer.ID, mas network.ConnMultiaddrs) (allow bool) {
	// Outbound dials are always accepted: the dial intercepts handle it before the connection is made.
	if direction == network.DirOutbound {
		return true
	}
	if !g.BlockingConnectionGater.InterceptSecured(direction, id, mas) {
		return false
	}
	// InterceptSecured is called after InterceptAccept, we already checked the addrs.
	// This leaves just the peer-ID expiry to check on inbound connections.
	peerBan := g.peerBanExpiryCheck(id)
	if !peerBan {
		log.Warn("peer id is temporarily banned", "method", "InterceptSecured", "peer_id", id, "multi_addr", mas.RemoteMultiaddr())
	}
	return peerBan
}
