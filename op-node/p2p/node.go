package p2p

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/hashicorp/go-multierror"
	"github.com/libp2p/go-libp2p-core/connmgr"
	"github.com/libp2p/go-libp2p-core/host"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/p2p/protocol/identify"
)

type NodeP2P struct {
	host     host.Host           // p2p host (optional, may be nil)
	gater    ConnectionGater     // p2p gater, to ban/unban peers with, may be nil even with p2p enabled
	connMgr  connmgr.ConnManager // p2p conn manager, to keep a reliable number of peers, may be nil even with p2p enabled
	dv5Local *enode.LocalNode    // p2p discovery identity (optional, may be nil)
	dv5Udp   *discover.UDPv5     // p2p discovery service (optional, may be nil)
	gs       *pubsub.PubSub      // p2p gossip router (optional, may be nil)
	gsOut    GossipOut           // p2p gossip application interface for publishing (optional, may be nil)
}

func NewNodeP2P(resourcesCtx context.Context, rollupCfg *rollup.Config, log log.Logger, setup SetupP2P, gossipIn GossipIn) (*NodeP2P, error) {
	if setup == nil {
		return nil, errors.New("p2p node cannot be created without setup")
	}
	var n NodeP2P
	if err := n.init(resourcesCtx, rollupCfg, log, setup, gossipIn); err != nil {
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

func (n *NodeP2P) init(resourcesCtx context.Context, rollupCfg *rollup.Config, log log.Logger, setup SetupP2P, gossipIn GossipIn) error {
	var err error
	// All nil if disabled.
	n.dv5Local, n.dv5Udp, err = setup.Discovery(log.New("p2p", "discv5"))
	if err != nil {
		return fmt.Errorf("failed to start discv5: %v", err)
	}

	// nil if disabled.
	n.host, err = setup.Host(log)
	if err != nil {
		if n.dv5Udp != nil {
			n.dv5Udp.Close()
		}
		return fmt.Errorf("failed to start p2p host: %v", err)
	}

	if n.host != nil {
		// Enable extra features, if any. During testing we don't setup the most advanced host all the time.
		if extra, ok := n.host.(ExtraHostFeatures); ok {
			n.gater = extra.ConnectionGater()
			n.connMgr = extra.ConnectionManager()
		}
		// notify of any new connections/streams/etc.
		n.host.Network().Notify(NewNetworkNotifier(log))
		// unregister identify-push handler. Only identifying on dial is fine, and more robust against spam
		n.host.RemoveStreamHandler(identify.IDDelta)
		n.gs, err = NewGossipSub(resourcesCtx, n.host, rollupCfg)
		if err != nil {
			return fmt.Errorf("failed to start gossipsub router: %v", err)
		}

		n.gsOut, err = JoinGossip(resourcesCtx, n.host.ID(), n.gs, log, rollupCfg, gossipIn)
		if err != nil {
			return fmt.Errorf("failed to join blocks gossip topic: %v", err)
		}
		log.Info("started p2p host", "addrs", n.host.Addrs(), "peerID", n.host.ID().Pretty())
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
			result = multierror.Append(result, fmt.Errorf("failed to close gossip cleanly: %v", err))
		}
	}
	if n.host != nil {
		if err := n.host.Close(); err != nil {
			result = multierror.Append(result, fmt.Errorf("failed to close p2p host cleanly: %v", err))
		}
	}
	return result.ErrorOrNil()
}
