package malleable

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"time"

	ds "github.com/ipfs/go-datastore"
	sync "github.com/ipfs/go-datastore/sync"
	libp2p "github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	connmgr "github.com/libp2p/go-libp2p/core/connmgr"
	crypto "github.com/libp2p/go-libp2p/core/crypto"
	host "github.com/libp2p/go-libp2p/core/host"
	peer "github.com/libp2p/go-libp2p/core/peer"
	insecure "github.com/libp2p/go-libp2p/core/sec/insecure"
	pstoreds "github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoreds"
	cmgr "github.com/libp2p/go-libp2p/p2p/net/connmgr"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
	ma "github.com/multiformats/go-multiaddr"
	madns "github.com/multiformats/go-multiaddr-dns"

	eth "github.com/ethereum-optimism/optimism/op-node/eth"
	p2p "github.com/ethereum-optimism/optimism/op-node/p2p"
)

const (
	maxGossipSize          = 10 * (1 << 20)
	maxOutboundQueue       = 256
	maxValidateQueue       = 256
	globalValidateThrottle = 512
	gossipHeartbeat        = 500 * time.Millisecond
	seenMessagesTTL        = 130 * gossipHeartbeat
	floodPublish           = true
	peersGrace             = time.Second * 10
	peersLo                = 1
	peersHi                = 10
)

// NewGossipSub configures a new pubsub instance with the specified parameters.
// PubSub uses a GossipSubRouter as it's router under the hood.
func NewGossipSub(h host.Host) (*pubsub.PubSub, error) {
	denyList, err := pubsub.NewTimeCachedBlacklist(30 * time.Second)
	if err != nil {
		return nil, err
	}
	params := p2p.BuildGlobalGossipParams(nil)
	gossipOpts := []pubsub.Option{
		pubsub.WithMaxMessageSize(maxGossipSize),
		pubsub.WithNoAuthor(),
		pubsub.WithMessageSignaturePolicy(pubsub.StrictNoSign),
		pubsub.WithValidateQueueSize(maxValidateQueue),
		pubsub.WithPeerOutboundQueueSize(maxOutboundQueue),
		pubsub.WithValidateThrottle(globalValidateThrottle),
		pubsub.WithSeenMessagesTTL(seenMessagesTTL),
		pubsub.WithPeerExchange(false),
		pubsub.WithBlacklist(denyList),
		pubsub.WithGossipSubParams(params),
		pubsub.WithFloodPublish(floodPublish),
	}
	return pubsub.NewGossipSub(context.Background(), h, gossipOpts...)
}

// getBlockTopicName returns the topic name for the given chain ID.
func getBlockTopicName(chainID *big.Int) string {
	return fmt.Sprintf("/optimism/%s/0/blocks", chainID.String())
}

// OnUnsafeL2Payload is called when a new L2 payload is received from the p2p network.
func OnUnsafeL2Payload(ctx context.Context, from peer.ID, payload *eth.ExecutionPayload) error {
	// TODO: allow this to be configurable by downstream users?
	return nil
}

func defaultConnManager() (connmgr.ConnManager, error) {
	return cmgr.NewConnManager(
		int(peersLo),
		int(peersHi),
		cmgr.WithGracePeriod(peersGrace),
		cmgr.WithSilencePeriod(time.Minute),
		cmgr.WithEmergencyTrim(true))
}

type minimalHost struct {
	host.Host
	connMgr connmgr.ConnManager
	quitC   chan struct{}
}

func (m *minimalHost) ConnectionGater() p2p.ConnectionGater {
	return nil
}

func (m *minimalHost) ConnectionManager() connmgr.ConnManager {
	return m.connMgr
}

func (m *minimalHost) Close() error {
	close(m.quitC)
	return m.Host.Close()
}

func DefaultHost(priv crypto.PrivKey) (host.Host, error) {
	pub := priv.GetPublic()
	pid, err := peer.IDFromPublicKey(pub)
	if err != nil {
		return nil, fmt.Errorf("failed to derive pubkey from network priv key: %w", err)
	}

	store := sync.MutexWrap(ds.NewMapDatastore())

	ps, err := pstoreds.NewPeerstore(context.Background(), store, pstoreds.DefaultOpts())
	if err != nil {
		return nil, fmt.Errorf("failed to open peerstore: %w", err)
	}

	if err := ps.AddPrivKey(pid, priv); err != nil {
		return nil, fmt.Errorf("failed to set up peerstore with priv key: %w", err)
	}
	if err := ps.AddPubKey(pid, pub); err != nil {
		return nil, fmt.Errorf("failed to set up peerstore with pub key: %w", err)
	}

	connMngr, err := defaultConnManager()
	if err != nil {
		return nil, fmt.Errorf("failed to open connection manager: %w", err)
	}

	// Bind to any available port on localhost
	listenIp := net.IP{127, 0, 0, 1}
	listenTCPPort := uint16(0)
	listenAddr, err := addrFromIPAndPort(listenIp, listenTCPPort)
	if err != nil {
		return nil, fmt.Errorf("failed to make listen addr: %w", err)
	}
	tcpTransport := libp2p.Transport(
		tcp.NewTCPTransport,
		tcp.WithConnectionTimeout(time.Minute*60))
	if err != nil {
		return nil, fmt.Errorf("failed to create TCP transport: %w", err)
	}

	timeoutDial := time.Second * 2
	userAgent := "optimism-testing"
	hostMux := []libp2p.Option{p2p.YamuxC(), p2p.MplexC()}
	opts := []libp2p.Option{
		libp2p.Identity(priv),
		libp2p.UserAgent(userAgent),
		tcpTransport,
		libp2p.WithDialTimeout(timeoutDial),
		libp2p.DisableRelay(),
		libp2p.ListenAddrs(listenAddr),
		libp2p.ConnectionManager(connMngr),
		libp2p.Peerstore(ps),
		libp2p.MultiaddrResolver(madns.DefaultResolver),
		libp2p.Ping(true),
		libp2p.EnableNATService(),
		libp2p.AutoNATServiceRateLimit(10, 5, time.Second*60),
	}
	opts = append(opts, hostMux...)
	opts = append(opts, libp2p.Security(insecure.ID, insecure.NewWithIdentity))
	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, err
	}

	out := &minimalHost{
		Host:    h,
		connMgr: connMngr,
		quitC:   make(chan struct{}),
	}

	return out, nil
}

func addrFromIPAndPort(ip net.IP, port uint16) (ma.Multiaddr, error) {
	ipScheme := "ip4"
	if ip4 := ip.To4(); ip4 == nil {
		ipScheme = "ip6"
	} else {
		ip = ip4
	}
	return ma.NewMultiaddr(fmt.Sprintf("/%s/%s/tcp/%d", ipScheme, ip.String(), port))
}
