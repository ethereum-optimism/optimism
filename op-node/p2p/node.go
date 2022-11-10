package p2p

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/hashicorp/go-multierror"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/host"
	p2pmetrics "github.com/libp2p/go-libp2p/core/metrics"
	"github.com/libp2p/go-libp2p/p2p/protocol/identify"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/ethereum-optimism/optimism/op-node/metrics"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

type NodeP2P struct {
	host    host.Host           // p2p host (optional, may be nil)
	gater   ConnectionGater     // p2p gater, to ban/unban peers with, may be nil even with p2p enabled
	connMgr connmgr.ConnManager // p2p conn manager, to keep a reliable number of peers, may be nil even with p2p enabled
	// the below components are all optional, and may be nil. They require the host to not be nil.
	dv5Local *enode.LocalNode // p2p discovery identity
	dv5Udp   *discover.UDPv5  // p2p discovery service
	gs       *pubsub.PubSub   // p2p gossip router
	gsOut    GossipOut        // p2p gossip application interface for publishing
}

func NewNodeP2P(resourcesCtx context.Context, rollupCfg *rollup.Config, log log.Logger, setup SetupP2P, gossipIn GossipIn, metrics metrics.Metricer) (*NodeP2P, error) {
	if setup == nil {
		return nil, errors.New("p2p node cannot be created without setup")
	}
	var n NodeP2P
	if err := n.init(resourcesCtx, rollupCfg, log, setup, gossipIn, metrics); err != nil {
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

func (n *NodeP2P) init(resourcesCtx context.Context, rollupCfg *rollup.Config, log log.Logger, setup SetupP2P, gossipIn GossipIn, metrics metrics.Metricer) error {
	bwc := p2pmetrics.NewBandwidthCounter()

	var err error
	// nil if disabled.
	n.host, err = setup.Host(log, bwc)
	if err != nil {
		if n.dv5Udp != nil {
			n.dv5Udp.Close()
		}
		return fmt.Errorf("failed to start p2p host: %w", err)
	}

	if n.host != nil {
		// Enable extra features, if any. During testing we don't setup the most advanced host all the time.
		if extra, ok := n.host.(ExtraHostFeatures); ok {
			n.gater = extra.ConnectionGater()
			n.connMgr = extra.ConnectionManager()
		}
		// notify of any new connections/streams/etc.
		n.host.Network().Notify(NewNetworkNotifier(log, metrics))
		// unregister identify-push handler. Only identifying on dial is fine, and more robust against spam
		n.host.RemoveStreamHandler(identify.IDDelta)
		n.gs, err = NewGossipSub(resourcesCtx, n.host, rollupCfg, metrics)
		if err != nil {
			return fmt.Errorf("failed to start gossipsub router: %w", err)
		}

		n.gsOut, err = JoinGossip(resourcesCtx, n.host.ID(), n.gs, log, rollupCfg, gossipIn)
		if err != nil {
			return fmt.Errorf("failed to join blocks gossip topic: %w", err)
		}
		log.Info("started p2p host", "addrs", n.host.Addrs(), "peerID", n.host.ID().Pretty())

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
	}
	return nil
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

func (n *NodeP2P) ConnectionGater() ConnectionGater {
	return n.gater
}

func (n *NodeP2P) ConnectionManager() connmgr.ConnManager {
	return n.connMgr
}

func (n *NodeP2P) Close() error {
	var result *multierror.Error
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
