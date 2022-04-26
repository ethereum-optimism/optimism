package p2p

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-peerstore/pstoreds"
	lconf "github.com/libp2p/go-libp2p/config"
	basichost "github.com/libp2p/go-libp2p/p2p/host/basic"
	tcp "github.com/libp2p/go-tcp-transport"
	ma "github.com/multiformats/go-multiaddr"
	madns "github.com/multiformats/go-multiaddr-dns"
)

func (conf *Config) Host() (host.Host, error) {
	// we cast the ecdsa key type to the libp2p wrapper, to then use the libp2p pubkey and ID interfaces.
	var priv crypto.PrivKey = (*crypto.Secp256k1PrivateKey)(conf.Priv)
	pub := priv.GetPublic()
	pid, err := peer.IDFromPublicKey(pub)
	if err != nil {
		return nil, fmt.Errorf("failed to derive pubkey from network priv key: %v", err)
	}

	ps, err := pstoreds.NewPeerstore(context.Background(), conf.Store, pstoreds.DefaultOpts())
	if err != nil {
		return nil, fmt.Errorf("failed to open peerstore: %v", err)
	}

	if err := ps.AddPrivKey(pid, priv); err != nil {
		return nil, fmt.Errorf("failed to set up peerstore with priv key: %v", err)
	}
	if err := ps.AddPubKey(pid, pub); err != nil {
		return nil, fmt.Errorf("failed to set up peerstore with pub key: %v", err)
	}

	connGtr, err := conf.ConnGater(conf)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection gater: %v", err)
	}

	connMngr, err := conf.ConnMngr(conf)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection manager: %v", err)
	}

	listenAddr, err := addrFromIPAndPort(conf.ListenIP, conf.ListenTCPPort)
	if err != nil {
		return nil, fmt.Errorf("failed to make listen addr: %v", err)
	}
	tcpTransport, err := lconf.TransportConstructor(
		tcp.NewTCPTransport,
		tcp.WithConnectionTimeout(time.Minute*60)) // break unused connections
	if err != nil {
		return nil, fmt.Errorf("failed to create TCP transport: %v", err)
	}
	// TODO: technically we can also run the node on websocket and QUIC transports. Maybe in the future?

	var nat lconf.NATManagerC // disabled if nil
	if conf.NAT {
		nat = basichost.NewNATManager
	}

	p2pConf := &lconf.Config{
		// Explicitly set the user-agent, so we can differentiate from other Go libp2p users.
		UserAgent: conf.UserAgent,

		PeerKey:            priv,
		Transports:         []lconf.TptC{tcpTransport},
		Muxers:             conf.HostMux,
		SecurityTransports: conf.HostSecurity,
		Insecure:           conf.NoTransportSecurity,
		PSK:                nil, // TODO: expose private subnet option to CLI / testing
		DialTimeout:        conf.TimeoutDial,
		// No relay services, direct connections between peers only.
		RelayCustom:        false,
		Relay:              false,
		EnableRelayService: false,
		RelayServiceOpts:   nil,
		// host will start and listen to network directly after construction from config.
		ListenAddrs: []ma.Multiaddr{listenAddr},

		AddrsFactory:      nil,
		ConnectionGater:   connGtr,
		ConnManager:       connMngr,
		ResourceManager:   nil, // TODO use resource manager interface to manage resources per peer better.
		NATManager:        nat,
		Peerstore:         ps,
		Reporter:          conf.BandwidthMetrics, // may be nil if disabled
		MultiaddrResolver: madns.DefaultResolver,
		// Ping is a small built-in libp2p protocol that helps us check/debug latency between peers.
		DisablePing:     false,
		Routing:         nil,
		EnableAutoRelay: false, // don't act as auto relay service
		// Help peers with their NAT reachability status, but throttle to avoid too much work.
		AutoNATConfig: lconf.AutoNATConfig{
			ForceReachability:   nil,
			EnableService:       true,
			ThrottleGlobalLimit: 10,
			ThrottlePeerLimit:   5,
			ThrottleInterval:    time.Second * 60,
		},
		// no static-relays, a "sentry" type infra with static peers and redundancy seems better
		StaticRelayOpt: nil,

		// TODO: hole punching is new, need to review differences with NAT manager options
		EnableHolePunching:  false,
		HolePunchingOptions: nil,
	}
	return p2pConf.NewNode()
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
