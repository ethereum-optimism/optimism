package gating

import (
	"errors"
	"net"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
)

type UnbanMetrics interface {
	RecordPeerUnban()
	RecordIPUnban()
}

var UnknownExpiry = errors.New("unknown ban expiry")

//go:generate mockery --name ExpiryStore --output mocks/ --with-expecter=true
type ExpiryStore interface {
	PeerBanExpiry(id peer.ID) (time.Time, error)
	IPBanExpiry(ip net.IP) (time.Time, error)
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
	expiry, err := g.store.PeerBanExpiry(p)
	if err != nil {
		if errors.Is(err, UnknownExpiry) {
			return false // peer is permanently banned if no expiry time is set.
		}
		g.log.Warn("failed to load peer-ban expiry time", "peer_id", p, "err", err)
		return false
	}
	if g.clock.Now().Before(expiry) {
		return false
	}
	g.log.Info("peer-ban expired, unbanning peer", "peer_id", p, "expiry", expiry)
	if err := g.BlockingConnectionGater.UnblockPeer(p); err != nil {
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
	// Check if it's a subnet-wide ban first. Subnet-bans do not expire.
	for _, ipnet := range g.BlockingConnectionGater.ListBlockedSubnets() {
		if ipnet.Contains(ip) {
			return false // peer is still in banned subnet
		}
	}
	// if just the IP is blocked, check if it's time to unblock
	expiry, err := g.store.IPBanExpiry(ip)
	if err != nil {
		if errors.Is(err, UnknownExpiry) {
			return false // IP is permanently banned if no expiry time is set.
		}
		g.log.Warn("failed to load IP-ban expiry time", "ip", ip, "err", err)
		return false
	}
	if g.clock.Now().Before(expiry) {
		return false
	}
	g.log.Info("IP-ban expired, unbanning IP", "ip", ip, "expiry", expiry)
	if err := g.BlockingConnectionGater.UnblockAddr(ip); err != nil {
		g.log.Warn("failed to unban IP", "ip", ip, "err", err)
		return false // if we ignored the error, then the inner connection-gater would drop them
	}
	g.m.RecordIPUnban()
	return true
}

func (g *ExpiryConnectionGater) InterceptPeerDial(p peer.ID) (allow bool) {
	// if not allowed, and not expired, then do not allow the dial
	return g.BlockingConnectionGater.InterceptPeerDial(p) || g.peerBanExpiryCheck(p)
}

func (g *ExpiryConnectionGater) InterceptAddrDial(id peer.ID, ma multiaddr.Multiaddr) (allow bool) {
	if !g.BlockingConnectionGater.InterceptAddrDial(id, ma) {
		// Check if it was intercepted because of a peer ban
		if !g.BlockingConnectionGater.InterceptPeerDial(id) {
			if !g.peerBanExpiryCheck(id) {
				return false // peer is still peer-banned
			}
			if g.BlockingConnectionGater.InterceptAddrDial(id, ma) { // allow dial if peer-ban was everything
				return true
			}
		}
		// intercepted because of addr ban still, check if it is expired
		if !g.addrBanExpiryCheck(ma) {
			return false // peer is still addr-banned
		}
	}
	return true
}

func (g *ExpiryConnectionGater) InterceptAccept(mas network.ConnMultiaddrs) (allow bool) {
	return g.BlockingConnectionGater.InterceptAccept(mas) || g.addrBanExpiryCheck(mas.RemoteMultiaddr())
}

func (g *ExpiryConnectionGater) InterceptSecured(direction network.Direction, id peer.ID, mas network.ConnMultiaddrs) (allow bool) {
	// Outbound dials are always accepted: the dial intercepts handle it before the connection is made.
	if direction == network.DirOutbound {
		return true
	}
	// InterceptSecured is called after InterceptAccept, we already checked the addrs.
	// This leaves just the peer-ID expiry to check on inbound connections.
	return g.BlockingConnectionGater.InterceptSecured(direction, id, mas) || g.peerBanExpiryCheck(id)
}
