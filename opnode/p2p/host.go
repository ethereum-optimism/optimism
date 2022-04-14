package p2p

import (
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/metrics"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-peerstore/pstoreds"
	"github.com/libp2p/go-libp2p-swarm"
	basichost "github.com/libp2p/go-libp2p/p2p/host/basic"
	"github.com/libp2p/go-libp2p/p2p/net/conngater"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	madns "github.com/multiformats/go-multiaddr-dns"
	"time"
)

func (conf *Config) Host() (host.Host, error) {
	// We do some more effort than just calling libp2p.New() so we can:
	// - configure the "swarm" setup, to use a more light one during testing
	// - hook up new features as they come out
	// - swap/customize components more easily
	// - control the transport preferences / upgrades

	// we cast the ecdsa key type to the libp2p wrapper, to then use the libp2p pubkey and ID interfaces.
	var priv crypto.PrivKey = (*crypto.Secp256k1PrivateKey)(conf.Priv)
	pid, err := peer.IDFromPublicKey(priv.GetPublic())
	if err != nil {
		return nil, fmt.Errorf("failed to derive pubkey from network priv key: %v", err)
	}

	connManager, err := connmgr.NewConnManager(
		int(conf.PeersLo),
		int(conf.PeersHi),
		connmgr.WithGracePeriod(conf.PeersGrace),
		connmgr.WithSilencePeriod(time.Minute),
		connmgr.WithEmergencyTrim(true))
	if err != nil {
		return nil, fmt.Errorf("failed to setup connection manager: %v", err)
	}

	bandwidthMetrics := metrics.NewBandwidthCounter()

	peerstore, err := pstoreds.NewPeerstore(context.Background(), conf.Store, pstoreds.DefaultOpts())
	if err != nil {
		return nil, fmt.Errorf("failed to open peerstore: %v", err)
	}

	connGtr, err := conngater.NewBasicConnectionGater(conf.Store)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection gater: %v", err)
	}

	// TODO: option to swap in the testing.GenSwarm(t, opts...) output here.

	// TODO: we can add swarm.WithResourceManager() to manage resources per peer better.
	network, err := swarm.NewSwarm(pid, peerstore,
		swarm.WithMetrics(bandwidthMetrics),
		swarm.WithDialTimeout(conf.TimeoutDial),
		swarm.WithConnectionGater(connGtr))
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p network core: %v", err)
	}

	// TODO: combine muxers
	// TODO: combine secure transports
	// TODO: create transport upgrader
	// TODO: add tptu transport to network

	h, err := basichost.NewHost(network, &basichost.HostOpts{
		MultistreamMuxer:   nil,
		NegotiationTimeout: conf.TimeoutNegotiation,
		AddrsFactory:       nil,
		// We can change / disable the DNS resolving of names in multi-addrs if we want to. Default is fine.
		MultiaddrResolver: madns.DefaultResolver,
		// The default NAT manager just tracks mappings. Auto-nat / fixed IP etc. options are separate.
		NATManager:  basichost.NewNATManager,
		ConnManager: connManager,
		// Ping is a small built-in libp2p protocol that helps us check/debug latency between peers.
		EnablePing: true,
		// We don't enable relay for now, nodes should just rely on real NAT methods instead
		EnableRelayService: false,
		RelayServiceOpts:   nil,
		// Explicitly set the user-agent, so we can differentiate from other Go libp2p users.
		UserAgent: conf.UserAgent,
		// We don't strictly need these, but no harm in enabling
		DisableSignedPeerRecord: false,
		// TODO: hole punching is new, need to review differences with NAT manager options
		EnableHolePunching:  false,
		HolePunchingOptions: nil,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to setup new host: %v", err)
	}

	// TODO: do we want to immediately listen on the network?
	//h.Network().Listen()

	// TODO: maybe setup autonat, the new libp2p one (not old deprecated autonat)
	// https://github.com/libp2p/go-libp2p/tree/master/p2p/host/autonat

	return h, nil
}
