package p2p

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/hashicorp/go-multierror"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/host"
	p2pmetrics "github.com/libp2p/go-libp2p/core/metrics"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/p2p/gating"
	"github.com/ethereum-optimism/optimism/op-node/p2p/monitor"
	"github.com/ethereum-optimism/optimism/op-node/p2p/store"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// NodeP2P is a p2p node, which can be used to gossip messages.
type NodeP2P struct {
	host        host.Host                      // p2p host (optional, may be nil)
	gater       gating.BlockingConnectionGater // p2p gater, to ban/unban peers with, may be nil even with p2p enabled
	scorer      Scorer                         // writes score-updates to the peerstore and keeps metrics of score changes
	connMgr     connmgr.ConnManager            // p2p conn manager, to keep a reliable number of peers, may be nil even with p2p enabled
	peerMonitor *monitor.PeerMonitor           // peer monitor to disconnect bad peers, may be nil even with p2p enabled
	store       store.ExtendedPeerstore        // peerstore of host, with extra bindings for scoring and banning
	appScorer   ApplicationScorer
	log         log.Logger
	// the below components are all optional, and may be nil. They require the host to not be nil.
	dv5Local *enode.LocalNode // p2p discovery identity
	dv5Udp   *discover.UDPv5  // p2p discovery service
	gs       *pubsub.PubSub   // p2p gossip router
	gsOut    GossipOut        // p2p gossip application interface for publishing
	syncCl   *SyncClient
	syncSrv  *ReqRespServer
}

// NewNodeP2P creates a new p2p node, and returns a reference to it. If the p2p is disabled, it returns nil.
// If metrics are configured, a bandwidth monitor will be spawned in a goroutine.
func NewNodeP2P(resourcesCtx context.Context, rollupCfg *rollup.Config, log log.Logger, setup SetupP2P, gossipIn GossipIn, l2Chain L2Chain, runCfg GossipRuntimeConfig, metrics metrics.Metricer, elSyncEnabled bool) (*NodeP2P, error) {
	if setup == nil {
		return nil, errors.New("p2p node cannot be created without setup")
	}
	var n NodeP2P
	if err := n.init(resourcesCtx, rollupCfg, log, setup, gossipIn, l2Chain, runCfg, metrics, elSyncEnabled); err != nil {
		closeErr := n.Close()
		if closeErr != nil {
			log.Error("failed to close p2p after starting with err", "closeErr", closeErr, "err", err)
		}
		return nil, err
	}
	if n.host == nil {
		return nil, nil
	}
	return &n, nil
}

func (n *NodeP2P) init(resourcesCtx context.Context, rollupCfg *rollup.Config, log log.Logger, setup SetupP2P, gossipIn GossipIn, l2Chain L2Chain, runCfg GossipRuntimeConfig, metrics metrics.Metricer, elSyncEnabled bool) error {
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

	// TODO(CLI-4016): host is not optional, NodeP2P as a whole is. This if statement is wrong
	if n.host != nil {
		// Enable extra features, if any. During testing we don't setup the most advanced host all the time.
		if extra, ok := n.host.(ExtraHostFeatures); ok {
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
			n.appScorer = newPeerApplicationScorer(resourcesCtx, log, clock.SystemClock, &scoreParams.ApplicationScoring, eps, n.host.Network().Peers)
		} else {
			n.appScorer = &NoopApplicationScorer{}
		}
		// Activate the P2P req-resp sync if enabled by feature-flag.
		if setup.ReqRespSyncEnabled() && !elSyncEnabled {
			n.syncCl = NewSyncClient(log, rollupCfg, n.host.NewStream, gossipIn.OnUnsafeL2Payload, metrics, n.appScorer)
			n.host.Network().Notify(&network.NotifyBundle{
				ConnectedF: func(nw network.Network, conn network.Conn) {
					n.syncCl.AddPeer(conn.RemotePeer())
				},
				DisconnectedF: func(nw network.Network, conn network.Conn) {
					// only when no connection is available, we can remove the peer
					if nw.Connectedness(conn.RemotePeer()) == network.NotConnected {
						n.syncCl.RemovePeer(conn.RemotePeer())
					}
				},
			})
			n.syncCl.Start()
			// the host may already be connected to peers, add them all to the sync client
			for _, peerID := range n.host.Network().Peers() {
				n.syncCl.AddPeer(peerID)
			}
			if l2Chain != nil { // Only enable serving side of req-resp sync if we have a data-source, to make minimal P2P testing easy
				n.syncSrv = NewReqRespServer(rollupCfg, l2Chain, metrics)
				// register the sync protocol with libp2p host
				payloadByNumber := MakeStreamHandler(resourcesCtx, log.New("serve", "payloads_by_number"), n.syncSrv.HandleSyncRequest)
				n.host.SetStreamHandler(PayloadByNumberProtocolID(rollupCfg.L2ChainID), payloadByNumber)
			}
		}
		n.scorer = NewScorer(rollupCfg, eps, metrics, n.appScorer, log)
		// notify of any new connections/streams/etc.
		n.host.Network().Notify(NewNetworkNotifier(log, metrics))
		// note: the IDDelta functionality was removed from libP2P, and no longer needs to be explicitly disabled.
		n.gs, err = NewGossipSub(resourcesCtx, n.host, rollupCfg, setup, n.scorer, metrics, log)
		if err != nil {
			return fmt.Errorf("failed to start gossipsub router: %w", err)
		}
		n.gsOut, err = JoinGossip(n.host.ID(), n.gs, log, rollupCfg, runCfg, gossipIn)
		if err != nil {
			return fmt.Errorf("failed to join blocks gossip topic: %w", err)
		}
		log.Info("started p2p host", "addrs", n.host.Addrs(), "peerID", n.host.ID().String())

		tcpPort, err := FindActiveTCPPort(n.host)
		if err != nil {
			log.Warn("failed to find what TCP port p2p is binded to", "err", err)
		}

		// All nil if disabled.
		n.dv5Local, n.dv5Udp, err = setup.Discovery(log.New("p2p", "discv5"), rollupCfg, tcpPort)
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
		n.appScorer.start()
	}
	return nil
}

func (n *NodeP2P) AltSyncEnabled() bool {
	return n.syncCl != nil
}

func (n *NodeP2P) RequestL2Range(ctx context.Context, start, end eth.L2BlockRef) error {
	if !n.AltSyncEnabled() {
		return fmt.Errorf("cannot request range %s - %s, req-resp sync is not enabled", start, end)
	}
	return n.syncCl.RequestL2Range(ctx, start, end)
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

func (n *NodeP2P) GossipOut() GossipOut {
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
	return n.connMgr != nil && n.connMgr.IsProtected(id, staticPeerTag)
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

func (n *NodeP2P) Close() error {
	var result *multierror.Error
	if n.peerMonitor != nil {
		n.peerMonitor.Stop()
	}
	if n.dv5Udp != nil {
		n.dv5Udp.Close()
	}
	if n.gsOut != nil {
		if err := n.gsOut.Close(); err != nil {
			result = multierror.Append(result, fmt.Errorf("failed to close gossip cleanly: %w", err))
		}
	}
	if n.host != nil {
		if err := n.host.Close(); err != nil {
			result = multierror.Append(result, fmt.Errorf("failed to close p2p host cleanly: %w", err))
		}
		if n.syncCl != nil {
			if err := n.syncCl.Close(); err != nil {
				result = multierror.Append(result, fmt.Errorf("failed to close p2p sync client cleanly: %w", err))
			}
		}
	}
	if n.appScorer != nil {
		n.appScorer.stop()
	}
	return result.ErrorOrNil()
}

func FindActiveTCPPort(h host.Host) (uint16, error) {
	var tcpPort uint16
	for _, addr := range h.Addrs() {
		tcpPortStr, err := addr.ValueForProtocol(ma.P_TCP)
		if err != nil {
			continue
		}
		v, err := strconv.ParseUint(tcpPortStr, 10, 16)
		if err != nil {
			continue
		}
		tcpPort = uint16(v)
		break
	}
	return tcpPort, nil
}
