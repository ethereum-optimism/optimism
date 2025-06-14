package p2p

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/host"
	p2pmetrics "github.com/libp2p/go-libp2p/core/metrics"
	"github.com/libp2p/go-libp2p/core/peer"
	manet "github.com/multiformats/go-multiaddr/net"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-node/p2p/gating"
	"github.com/ethereum-optimism/optimism/op-node/p2p/monitor"
	"github.com/ethereum-optimism/optimism/op-node/p2p/store"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type P2PMetricer interface {
	p2p.HostMetrics
	p2p.GossipMetricer
	p2p.ScoreMetrics
	p2p.NotificationsMetricer
	RecordBandwidth(ctx context.Context, bwc *p2pmetrics.BandwidthCounter)
}

// NodeP2P is a p2p node, which can be used to gossip messages.
type NodeP2P struct {
	host        host.Host                      // p2p host (optional, may be nil)
	gater       gating.BlockingConnectionGater // p2p gater, to ban/unban peers with, may be nil even with p2p enabled
	scorer      p2p.Scorer                     // writes score-updates to the peerstore and keeps metrics of score changes
	connMgr     connmgr.ConnManager            // p2p conn manager, to keep a reliable number of peers, may be nil even with p2p enabled
	peerMonitor *monitor.PeerMonitor           // peer monitor to disconnect bad peers, may be nil even with p2p enabled
	store       store.ExtendedPeerstore        // peerstore of host, with extra bindings for scoring and banning
	appScorer   p2p.ApplicationScorer
	log         log.Logger
	// the below components are all optional, and may be nil. They require the host to not be nil.
	dv5Local *enode.LocalNode // p2p discovery identity
	dv5Udp   *discover.UDPv5  // p2p discovery service
	gs       *pubsub.PubSub   // p2p gossip router
	gsOut    p2p.GossipOut    // p2p gossip application interface for publishing
}

// NewNodeP2P creates a new p2p node, and returns a reference to it. If the p2p is disabled, it returns nil.
// If metrics are configured, a bandwidth monitor will be spawned in a goroutine.
func NewNodeP2P(
	resourcesCtx context.Context,
	log log.Logger,
	setup p2p.SetupP2P,
	gossipIn p2p.GossipIn,
	chainCfg p2p.GossipChainConfig, // TODO need multichain cfg
	metrics P2PMetricer,
) (*NodeP2P, error) {
	if setup == nil {
		return nil, errors.New("p2p node cannot be created without setup")
	}
	if setup.Disabled() {
		return nil, errors.New("SetupP2P.Disabled is true")
	}
	var n NodeP2P
	if err := n.init(resourcesCtx, log, setup, gossipIn, chainCfg, metrics); err != nil {
		closeErr := n.Close()
		if closeErr != nil {
			log.Error("failed to close p2p after starting with err", "closeErr", closeErr, "err", err)
		}
		return nil, err
	}
	if n.host == nil {
		// See prior comment about n.host optionality:
		// TODO: host is not optional, NodeP2P as a whole is.
		panic("host is not optional if p2p is enabled")
	}
	return &n, nil
}

func (n *NodeP2P) init(
	resourcesCtx context.Context,
	log log.Logger,
	setup p2p.SetupP2P,
	gossipIn p2p.GossipIn,
	chainCfg p2p.GossipChainConfig,
	metrics P2PMetricer,
) error {
	bwc := p2pmetrics.NewBandwidthCounter()

	n.log = log

	var err error
	// nil if disabled.
	n.host, err = setup.Host(log, bwc, metrics)
	if err != nil {
		if n.dv5Udp != nil {
			n.dv5Udp.Close()
		}
		return fmt.Errorf("failed to start p2p host: %w", err)
	}

	// Don't ingest blocks that we publish ourselves
	gossipIn = p2p.NewFilterSelf(n.host.ID(), gossipIn)

	// Enable extra features, if any. During testing we don't setup the most advanced host all the time.
	if extra, ok := n.host.(p2p.ExtraHostFeatures); ok {
		n.gater = extra.ConnectionGater()
		n.connMgr = extra.ConnectionManager()
	}
	eps, ok := n.host.Peerstore().(store.ExtendedPeerstore)
	if !ok {
		return fmt.Errorf("cannot init without extended peerstore: %w", err)
	}
	n.store = eps
	scoreParams := setup.PeerScoringParams()

	if scoreParams != nil {
		n.appScorer = p2p.NewPeerApplicationScorer(resourcesCtx, log, clock.SystemClock, &scoreParams.ApplicationScoring, eps, n.host.Network().Peers)
	} else {
		n.appScorer = &p2p.NoopApplicationScorer{}
	}
	n.scorer = p2p.NewScorer(eps, metrics, n.appScorer, log)
	// notify of any new connections/streams/etc.
	n.host.Network().Notify(p2p.NewNetworkNotifier(log, metrics))
	// note: the IDDelta functionality was removed from libP2P, and no longer needs to be explicitly disabled.

	n.gs, err = p2p.NewGossipSub(resourcesCtx, n.host, chainCfg, setup, n.scorer, metrics, log)
	if err != nil {
		return fmt.Errorf("failed to start gossipsub router: %w", err)
	}

	n.gsOut, err = p2p.JoinGossip(n.host.ID(), n.gs, log, chainCfg, gossipIn)
	if err != nil {
		return fmt.Errorf("failed to join blocks gossip topic: %w", err)
	}
	log.Info("started p2p host", "addrs", n.host.Addrs(), "peerID", n.host.ID().String())

	tcpPort, err := p2p.FindActiveTCPPort(n.host)
	if err != nil {
		log.Warn("failed to find what TCP port p2p is binded to", "err", err)
	}

	// Advertise the first of any list of chainIDs to identify we are part of the dependency set.
	// Later on we may want to transition to a new OpStackENRData version.
	localNodeMods := func(localNode *enode.LocalNode) {
		chainIDU64, _ := chainCfg.ChainIDs()[0].Uint64()
		dat := p2p.OpStackENRData{
			ChainID: chainIDU64,
			Version: 0,
		}
		localNode.Set(&dat)
	}

	// All nil if disabled.
	n.dv5Local, n.dv5Udp, err = setup.Discovery(log.New("p2p", "discv5"), localNodeMods, tcpPort)
	if err != nil {
		return fmt.Errorf("failed to start discv5: %w", err)
	}

	if metrics != nil {
		go metrics.RecordBandwidth(resourcesCtx, bwc)
	}

	if setup.BanPeers() {
		n.peerMonitor = monitor.NewPeerMonitor(resourcesCtx, log, clock.SystemClock, n, setup.BanThreshold(), setup.BanDuration())
		n.peerMonitor.Start()
	}
	n.appScorer.Start()
	return nil
}

func (n *NodeP2P) AltSyncEnabled() bool {
	return false
}

func (n *NodeP2P) RequestL2Range(ctx context.Context, start, end eth.L2BlockRef) error {
	return errors.New("req-resp is not supported")
}

func (n *NodeP2P) Host() host.Host {
	return n.host
}

func (n *NodeP2P) Dv5Local() *enode.LocalNode {
	return n.dv5Local
}

func (n *NodeP2P) Dv5Udp() *discover.UDPv5 {
	return n.dv5Udp
}

func (n *NodeP2P) GossipSub() *pubsub.PubSub {
	return n.gs
}

func (n *NodeP2P) GossipOut() p2p.GossipOut {
	return n.gsOut
}

func (n *NodeP2P) ConnectionGater() gating.BlockingConnectionGater {
	return n.gater
}

func (n *NodeP2P) ConnectionManager() connmgr.ConnManager {
	return n.connMgr
}

func (n *NodeP2P) Peers() []peer.ID {
	return n.host.Network().Peers()
}

func (n *NodeP2P) GetPeerScore(id peer.ID) (float64, error) {
	return n.store.GetPeerScore(id)
}

func (n *NodeP2P) IsStatic(id peer.ID) bool {
	return n.connMgr != nil && n.connMgr.IsProtected(id, p2p.StaticPeerTag)
}

func (n *NodeP2P) BanPeer(id peer.ID, expiration time.Time) error {
	if err := n.store.SetPeerBanExpiration(id, expiration); err != nil {
		return fmt.Errorf("failed to set peer ban expiry: %w", err)
	}
	if err := n.host.Network().ClosePeer(id); err != nil {
		return fmt.Errorf("failed to close peer connection: %w", err)
	}
	return nil
}

func (n *NodeP2P) BanIP(ip net.IP, expiration time.Time) error {
	if err := n.store.SetIPBanExpiration(ip, expiration); err != nil {
		return fmt.Errorf("failed to set IP ban expiry: %w", err)
	}
	// kick all peers that match this IP
	for _, conn := range n.host.Network().Conns() {
		addr := conn.RemoteMultiaddr()
		remoteIP, err := manet.ToIP(addr)
		if err != nil {
			continue
		}
		if remoteIP.Equal(ip) {
			if err := conn.Close(); err != nil {
				n.log.Error("failed to close connection to peer with banned IP", "peer", conn.RemotePeer(), "ip", ip)
			}
		}
	}
	return nil
}

func (n *NodeP2P) DiscoveryProcess(ctx context.Context, log log.Logger, nodeFilter p2p.NodeFilter, connectGoal uint) {
	p2p.DiscoveryProcess(ctx, log, n, nodeFilter, connectGoal)
}

func (n *NodeP2P) Close() error {
	var result error
	if n.peerMonitor != nil {
		n.peerMonitor.Stop()
	}
	if n.dv5Udp != nil {
		n.dv5Udp.Close()
	}
	if n.gsOut != nil {
		if err := n.gsOut.Close(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to close gossip cleanly: %w", err))
		}
	}
	if n.host != nil {
		if err := n.host.Close(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to close p2p host cleanly: %w", err))
		}
	}
	if n.appScorer != nil {
		n.appScorer.Stop()
	}
	return result
}
