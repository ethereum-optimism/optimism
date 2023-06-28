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

func (g *ExpiryConnectionGater) peerBanExpiryCheck(p peer.ID) (allow bool) {
	// if the peer is blocked, check if it's time to unblock
	expiry, err := g.store.GetPeerBanExpiration(p)
	if errors.Is(err, store.UnknownBanErr) {
		return true // peer is allowed if it has not been banned
	}
	if err != nil {
		g.log.Warn("failed to load peer-ban expiry time", "peer_id", p, "err", err)
		return false
	}
	if g.clock.Now().Before(expiry) {
		return false
	}
	g.log.Info("peer-ban expired, unbanning peer", "peer_id", p, "expiry", expiry)
	if err := g.store.SetPeerBanExpiration(p, time.Time{}); err != nil {
		g.log.Warn("failed to unban peer", "peer_id", p, "err", err)
		return false // if we ignored the error, then the inner connection-gater would drop them
	}
	g.m.RecordPeerUnban()
	return true
}

func (g *ExpiryConnectionGater) addrBanExpiryCheck(ma multiaddr.Multiaddr) (allow bool) {
	ip, err := manet.ToIP(ma)
	if err != nil {
		g.log.Error("tried to check multi-addr with bad IP", "addr", ma)
		return false
	}
	// if just the IP is blocked, check if it's time to unblock
	expiry, err := g.store.GetIPBanExpiration(ip)
	if errors.Is(err, store.UnknownBanErr) {
		return true // IP is allowed if it has not been banned
	}
	if err != nil {
		g.log.Warn("failed to load IP-ban expiry time", "ip", ip, "err", err)
		return false
	}
	if g.clock.Now().Before(expiry) {
		return false
	}
	g.log.Info("IP-ban expired, unbanning IP", "ip", ip, "expiry", expiry)
	if err := g.store.SetIPBanExpiration(ip, time.Time{}); err != nil {
		g.log.Warn("failed to unban IP", "ip", ip, "err", err)
		return false // if we ignored the error, then the inner connection-gater would drop them
	}
	g.m.RecordIPUnban()
	return true
}

func (g *ExpiryConnectionGater) InterceptPeerDial(p peer.ID) (allow bool) {
	if !g.BlockingConnectionGater.InterceptPeerDial(p) {
		return false
	}
	return g.peerBanExpiryCheck(p)
}

func (g *ExpiryConnectionGater) InterceptAddrDial(id peer.ID, ma multiaddr.Multiaddr) (allow bool) {
	if !g.BlockingConnectionGater.InterceptAddrDial(id, ma) {
		return false
	}
	return g.peerBanExpiryCheck(id) && g.addrBanExpiryCheck(ma)
}

func (g *ExpiryConnectionGater) InterceptAccept(mas network.ConnMultiaddrs) (allow bool) {
	if !g.BlockingConnectionGater.InterceptAccept(mas) {
		return false
	}
	return g.addrBanExpiryCheck(mas.RemoteMultiaddr())
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
	return g.peerBanExpiryCheck(id)
}
