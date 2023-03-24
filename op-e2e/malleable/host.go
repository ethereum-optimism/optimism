package malleable

import (
	"context"
	"fmt"
	"net"
	"time"

	p2p "github.com/ethereum-optimism/optimism/op-node/p2p"
	log "github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p/core/connmgr"
	host "github.com/libp2p/go-libp2p/core/host"
	metrics "github.com/libp2p/go-libp2p/core/metrics"
	peer "github.com/libp2p/go-libp2p/core/peer"
	insecure "github.com/libp2p/go-libp2p/core/sec/insecure"
	pstoreds "github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoreds"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
	ma "github.com/multiformats/go-multiaddr"
	madns "github.com/multiformats/go-multiaddr-dns"
)

// hostWrapper is a wrapper around the [host.Host] interface
// with a [p2p.ConnectionGater] and [connmgr.ConnManager].
type hostWrapper struct {
	host.Host
	gater   p2p.ConnectionGater
	connMgr connmgr.ConnManager
}

// ConnectionGater returns the [hostWrapper]'s [p2p.ConnectionGater].
func (h *hostWrapper) ConnectionGater() p2p.ConnectionGater {
	return h.gater
}

// ConnectionManager returns the [hostWrapper]'s [connmgr.ConnManager].
func (h *hostWrapper) ConnectionManager() connmgr.ConnManager {
	return h.connMgr
}

// Close closes the [hostWrapper].
func (h *hostWrapper) Close() error {
	return h.Host.Close()
}

// CreateMultiaddr creates an [ma.Multiaddr] to bind to.
// Does not contain a PeerID component (required for usage by external peers).
func CreateMultiaddr(ip net.IP, port uint16) (ma.Multiaddr, error) {
	ipScheme := "ip4"
	if ip4 := ip.To4(); ip4 == nil {
		ipScheme = "ip6"
	} else {
		ip = ip4
	}
	return ma.NewMultiaddr(fmt.Sprintf("/%s/%s/tcp/%d", ipScheme, ip.String(), port))
}

// BuildHost creates a new [host.Host] for the [MalleableNode].
func (m *MalleableNode) BuildHost(
	conf *p2p.Config,
	log log.Logger,
	reporter metrics.Reporter,
) (host.Host, error) {
	pub := conf.Priv.GetPublic()
	pid, err := peer.IDFromPublicKey(pub)
	if err != nil {
		return nil, fmt.Errorf("failed to derive pubkey from network priv key: %w", err)
	}

	ps, err := pstoreds.NewPeerstore(context.Background(), conf.Store, pstoreds.DefaultOpts())
	if err != nil {
		return nil, fmt.Errorf("failed to open peerstore: %w", err)
	}

	if err := ps.AddPrivKey(pid, conf.Priv); err != nil {
		return nil, fmt.Errorf("failed to set up peerstore with priv key: %w", err)
	}
	if err := ps.AddPubKey(pid, pub); err != nil {
		return nil, fmt.Errorf("failed to set up peerstore with pub key: %w", err)
	}

	listenAddr, err := CreateMultiaddr(conf.ListenIP, conf.ListenTCPPort)
	if err != nil {
		return nil, fmt.Errorf("failed to make listen addr: %w", err)
	}
	tcpTransport := libp2p.Transport(
		tcp.NewTCPTransport,
		tcp.WithConnectionTimeout(time.Minute*60)) // break unused connections
	if err != nil {
		return nil, fmt.Errorf("failed to create TCP transport: %w", err)
	}

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
		libp2p.ConnectionGater(m.gater),
		libp2p.ConnectionManager(m.connMgr),
		//libp2p.ResourceManager(nil),
		// libp2p.NATManager(nat),
		libp2p.Peerstore(ps),
		libp2p.BandwidthReporter(reporter), // may be nil if disabled
		libp2p.MultiaddrResolver(madns.DefaultResolver),
		// Ping is a small built-in libp2p protocol that helps us check/debug latency between peers.
		libp2p.Ping(true),
		// Help peers with their NAT reachability status, but throttle to avoid too much work.
		libp2p.EnableNATService(),
		libp2p.AutoNATServiceRateLimit(10, 5, time.Second*60),
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

	out := &hostWrapper{
		Host:    h,
		connMgr: m.connMgr,
		gater:   m.gater,
	}

	return out, nil
}
