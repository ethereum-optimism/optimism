package p2p

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	//nolint:all
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoreds"

	libp2p "github.com/libp2p/go-libp2p"
	mplex "github.com/libp2p/go-libp2p-mplex"
	"github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/metrics"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/core/sec/insecure"
	basichost "github.com/libp2p/go-libp2p/p2p/host/basic"
	"github.com/libp2p/go-libp2p/p2p/muxer/yamux"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	tls "github.com/libp2p/go-libp2p/p2p/security/tls"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	ma "github.com/multiformats/go-multiaddr"
	madns "github.com/multiformats/go-multiaddr-dns"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/p2p/gating"
	"github.com/ethereum-optimism/optimism/op-node/p2p/store"
	"github.com/ethereum-optimism/optimism/op-service/clock"
)

const (
	staticPeerTag = "static"
	// Maximum backoff time for reconnecting to static peers
	maxStaticPeerBackoff = 5 * time.Minute
	// Initial backoff time for reconnecting to static peers
	initialStaticPeerBackoff = 5 * time.Second
)

// peerRetryState tracks the retry state for a static peer
// It implements an exponential backoff mechanism to avoid
// excessive reconnection attempts to unavailable peers
type peerRetryState struct {
	nextRetry time.Time
	backoff   time.Duration
}

// resetBackoff resets the backoff for a peer
// This is called when a successful connection is established
func (p *peerRetryState) resetBackoff() {
	p.backoff = initialStaticPeerBackoff
	p.nextRetry = time.Time{}
}

// increaseBackoff increases the backoff for a peer using exponential backoff
// This is called when a connection attempt fails
func (p *peerRetryState) increaseBackoff() {
	p.backoff = time.Duration(float64(p.backoff) * 1.5)
	if p.backoff > maxStaticPeerBackoff {
		p.backoff = maxStaticPeerBackoff
	}
	p.nextRetry = time.Now().Add(p.backoff)
}

type HostNewStream interface {
	NewStream(ctx context.Context, p peer.ID, pids ...protocol.ID) (network.Stream, error)
}

type ExtraHostFeatures interface {
	host.Host
	ConnectionGater() gating.BlockingConnectionGater
	ConnectionManager() connmgr.ConnManager
	IsStatic(peerID peer.ID) bool
	SyncOnlyReqToStatic() bool
}

type extraHost struct {
	host.Host
	gater   gating.BlockingConnectionGater
	connMgr connmgr.ConnManager
	log     log.Logger

	staticPeers   []*peer.AddrInfo
	staticPeerIDs map[peer.ID]struct{}

	// Track retry state for static peers
	staticPeerRetries map[peer.ID]*peerRetryState
	retryMu           sync.Mutex

	pinging *PingService

	quitC chan struct{}

	syncOnlyReqToStatic bool
}

func (e *extraHost) ConnectionGater() gating.BlockingConnectionGater {
	return e.gater
}

func (e *extraHost) ConnectionManager() connmgr.ConnManager {
	return e.connMgr
}

func (e *extraHost) IsStatic(peerID peer.ID) bool {
	_, exists := e.staticPeerIDs[peerID]
	return exists
}

func (e *extraHost) SyncOnlyReqToStatic() bool {
	return e.syncOnlyReqToStatic
}

func (e *extraHost) Close() error {
	close(e.quitC)
	if e.pinging != nil {
		e.pinging.Close()
	}
	return e.Host.Close()
}

func (e *extraHost) initStaticPeers() {
	e.staticPeerRetries = make(map[peer.ID]*peerRetryState)

	for _, addr := range e.staticPeers {
		e.Peerstore().AddAddrs(addr.ID, addr.Addrs, time.Hour*24*7)
		// We protect the peer, so the connection manager doesn't decide to prune it.
		// We tag it with "static" so other protects/unprotects with different tags don't affect this protection.
		e.connMgr.Protect(addr.ID, staticPeerTag)

		// Initialize retry state for each static peer
		e.retryMu.Lock()
		e.staticPeerRetries[addr.ID] = &peerRetryState{
			backoff: initialStaticPeerBackoff,
		}
		e.retryMu.Unlock()

		// Try to dial the node in the background
		go func(addr *peer.AddrInfo) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
			defer cancel()
			if err := e.dialStaticPeer(ctx, addr); err != nil {
				e.log.Warn("error dialing static peer", "peer", addr.ID, "err", err)
				// Initialize backoff on first connection failure
				e.retryMu.Lock()
				if state, exists := e.staticPeerRetries[addr.ID]; exists {
					state.increaseBackoff()
				}
				e.retryMu.Unlock()
			}
		}(addr)
	}
}

func (e *extraHost) dialStaticPeer(ctx context.Context, addr *peer.AddrInfo) error {
	e.log.Info("dialing static peer", "peer", addr.ID, "addrs", addr.Addrs)
	if _, err := e.Network().DialPeer(ctx, addr.ID); err != nil {
		return err
	}

	// Reset backoff on successful connection
	e.retryMu.Lock()
	if state, exists := e.staticPeerRetries[addr.ID]; exists {
		state.resetBackoff()
	}
	e.retryMu.Unlock()

	return nil
}

func (e *extraHost) monitorStaticPeers() {
	tick := time.NewTicker(time.Minute)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
			var wg sync.WaitGroup

			e.log.Debug("polling static peers", "peers", len(e.staticPeers))
			for _, addr := range e.staticPeers {
				connectedness := e.Network().Connectedness(addr.ID)
				e.log.Trace("static peer connectedness", "peer", addr.ID, "connectedness", connectedness)

				if connectedness == network.Connected {
					// Reset backoff on successful connection
					e.retryMu.Lock()
					if state, exists := e.staticPeerRetries[addr.ID]; exists {
						state.resetBackoff()
					}
					e.retryMu.Unlock()
					continue
				}

				// Check if we should attempt to reconnect based on backoff
				e.retryMu.Lock()
				state, exists := e.staticPeerRetries[addr.ID]
				shouldRetry := !exists || time.Now().After(state.nextRetry)
				e.retryMu.Unlock()

				if !shouldRetry {
					e.log.Debug("skipping reconnect to static peer due to backoff",
						"peer", addr.ID,
						"next_retry", state.nextRetry,
						"backoff", state.backoff)
					continue
				}

				wg.Add(1)
				go func(addr *peer.AddrInfo) {
					defer wg.Done()
					e.log.Warn("static peer disconnected, reconnecting", "peer", addr.ID)
					if err := e.dialStaticPeer(ctx, addr); err != nil {
						e.log.Warn("error reconnecting to static peer", "peer", addr.ID, "err", err)

						// Increase backoff on failed connection attempt
						e.retryMu.Lock()
						if state, exists := e.staticPeerRetries[addr.ID]; exists {
							state.increaseBackoff()
							e.log.Debug("increased backoff for static peer",
								"peer", addr.ID,
								"next_retry", state.nextRetry,
								"backoff", state.backoff)
						}
						e.retryMu.Unlock()
					}
				}(addr)
			}

			wg.Wait()
			cancel()
		case <-e.quitC:
			return
		}
	}
}

var _ ExtraHostFeatures = (*extraHost)(nil)

func (conf *Config) Host(log log.Logger, reporter metrics.Reporter, metrics HostMetrics) (host.Host, error) {
	if conf.DisableP2P {
		return nil, nil
	}
	pub := conf.Priv.GetPublic()
	pid, err := peer.IDFromPublicKey(pub)
	if err != nil {
		return nil, fmt.Errorf("failed to derive pubkey from network priv key: %w", err)
	}

	basePs, err := pstoreds.NewPeerstore(context.Background(), conf.Store, pstoreds.DefaultOpts())
	if err != nil {
		return nil, fmt.Errorf("failed to open peerstore: %w", err)
	}

	peerScoreParams := conf.PeerScoringParams()
	var scoreRetention time.Duration
	if peerScoreParams != nil {
		// Use the same retention period as gossip will if available
		scoreRetention = peerScoreParams.PeerScoring.RetainScore
	} else {
		// Disable score GC if peer scoring is disabled
		scoreRetention = 0
	}
	ps, err := store.NewExtendedPeerstore(context.Background(), log, clock.SystemClock, basePs, conf.Store, scoreRetention)
	if err != nil {
		return nil, fmt.Errorf("failed to open extended peerstore: %w", err)
	}

	if err := ps.AddPrivKey(pid, conf.Priv); err != nil {
		return nil, fmt.Errorf("failed to set up peerstore with priv key: %w", err)
	}
	if err := ps.AddPubKey(pid, pub); err != nil {
		return nil, fmt.Errorf("failed to set up peerstore with pub key: %w", err)
	}

	var connGtr gating.BlockingConnectionGater
	connGtr, err = gating.NewBlockingConnectionGater(conf.Store)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection gater: %w", err)
	}
	connGtr = gating.AddBanExpiry(connGtr, ps, log, clock.SystemClock, metrics)
	connGtr = gating.AddMetering(connGtr, metrics)

	connMngr, err := DefaultConnManager(conf)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection manager: %w", err)
	}

	listenAddr, err := addrFromIPAndPort(conf.ListenIP, conf.ListenTCPPort)
	if err != nil {
		return nil, fmt.Errorf("failed to make listen addr: %w", err)
	}
	tcpTransport := libp2p.Transport(
		tcp.NewTCPTransport,
		tcp.WithConnectionTimeout(time.Minute*60)) // break unused connections
	// TODO: technically we can also run the node on websocket and QUIC transports. Maybe in the future?

	opts := []libp2p.Option{
		libp2p.Identity(conf.Priv),
		// Explicitly set the user-agent, so we can differentiate from other Go libp2p users.
		libp2p.UserAgent(conf.UserAgent),
		tcpTransport,
		libp2p.WithDialTimeout(conf.TimeoutDial),
		// No relay services, direct connections between peers only.
		libp2p.DisableRelay(),
		// host will start and listen to network directly after construction from config.
		libp2p.ListenAddrs(listenAddr),
		libp2p.ConnectionGater(connGtr),
		libp2p.ConnectionManager(connMngr),
		//libp2p.ResourceManager(nil), // TODO use resource manager interface to manage resources per peer better.
		libp2p.Peerstore(ps),
		libp2p.BandwidthReporter(reporter), // may be nil if disabled
		libp2p.MultiaddrResolver(madns.DefaultResolver),
		// Ping is a small built-in libp2p protocol that helps us check/debug latency between peers.
		libp2p.Ping(true),
	}
	if conf.NAT {
		// Help peers with their NAT reachability status, but throttle to avoid too much work.
		opts = append(opts,
			libp2p.NATManager(basichost.NewNATManager),
			libp2p.EnableNATService(),
			libp2p.AutoNATServiceRateLimit(10, 5, time.Second*60))
	}
	opts = append(opts, conf.HostMux...)
	if conf.NoTransportSecurity {
		opts = append(opts, libp2p.Security(insecure.ID, insecure.NewWithIdentity))
	} else {
		opts = append(opts, conf.HostSecurity...)
	}
	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, err
	}

	staticPeers := make([]*peer.AddrInfo, 0, len(conf.StaticPeers))
	staticPeerIDs := make(map[peer.ID]struct{})
	for _, peerAddr := range conf.StaticPeers {
		addr, err := peer.AddrInfoFromP2pAddr(peerAddr)
		if err != nil {
			return nil, fmt.Errorf("bad peer address: %w", err)
		}
		if addr.ID == h.ID() {
			log.Info("Static-peer list contains address of local peer, ignoring the address.", "peer_id", addr.ID, "addrs", addr.Addrs)
			continue
		}
		staticPeers = append(staticPeers, addr)
		staticPeerIDs[addr.ID] = struct{}{}
	}

	out := &extraHost{
		Host:                h,
		connMgr:             connMngr,
		log:                 log,
		staticPeers:         staticPeers,
		staticPeerIDs:       staticPeerIDs,
		quitC:               make(chan struct{}),
		syncOnlyReqToStatic: conf.SyncOnlyReqToStatic,
	}

	if conf.EnablePingService {
		out.pinging = NewPingService(
			log,
			func(ctx context.Context, peerID peer.ID) <-chan ping.Result {
				return ping.Ping(ctx, h, peerID)
			},
			h.Network().Peers,
		)
	}

	out.initStaticPeers()
	if len(conf.StaticPeers) > 0 {
		go out.monitorStaticPeers()
	}

	out.gater = connGtr
	return out, nil
}

// Creates a multi-addr to bind to. Does not contain a PeerID component (required for usage by external peers)
func addrFromIPAndPort(ip net.IP, port uint16) (ma.Multiaddr, error) {
	ipScheme := "ip4"
	if ip4 := ip.To4(); ip4 == nil {
		ipScheme = "ip6"
	} else {
		ip = ip4
	}
	return ma.NewMultiaddr(fmt.Sprintf("/%s/%s/tcp/%d", ipScheme, ip.String(), port))
}

func YamuxC() libp2p.Option {
	return libp2p.Muxer("/yamux/1.0.0", yamux.DefaultTransport)
}

func MplexC() libp2p.Option {
	return libp2p.Muxer("/mplex/6.7.0", mplex.DefaultTransport)
}

func NoiseC() libp2p.Option {
	return libp2p.Security(noise.ID, noise.New)
}

func TlsC() libp2p.Option {
	return libp2p.Security(tls.ID, tls.New)
}
