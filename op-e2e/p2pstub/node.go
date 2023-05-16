package p2pstub

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/log"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

type GossipTopic string

const (
	BlockTopic = GossipTopic("blocks")
)

type Config struct {
	P2P    p2p.SetupP2P
	Rollup rollup.Config
}

type P2PNode struct {
	log    log.Logger
	config *Config
	gs     *pubsub.PubSub
	host   host.Host
	topics map[GossipTopic]*pubsub.Topic
}

func NewP2pNode(logger log.Logger, config Config) (*P2PNode, error) {
	return &P2PNode{
		log:    logger,
		config: &config,
		topics: make(map[GossipTopic]*pubsub.Topic),
	}, nil
}

func (n *P2PNode) Start(ctx context.Context) error {
	if err := n.config.P2P.Check(); err != nil {
		return fmt.Errorf("invalid p2p config: %w", err)
	}
	h, err := n.config.P2P.Host(n.log, nil)
	if err != nil {
		return fmt.Errorf("create host: %w", err)
	}
	n.gs, err = p2p.NewGossipSub(ctx, h, nil, &n.config.Rollup, n.config.P2P, metrics.NoopMetrics, n.log)
	n.host = h
	return nil
}

func (n *P2PNode) Close() error {
	return n.host.Close()
}

func (n *P2PNode) WaitForPeerCount(expected int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return e2eutils.WaitFor(ctx, 1*time.Second, func() (bool, error) {
		return expected == len(n.host.Network().Peers()), nil
	})
}

func (n *P2PNode) JoinTopic(topic GossipTopic) error {
	if _, ok := n.topics[topic]; ok {
		return fmt.Errorf("already joined topic: %v", topic)
	}
	var topicName string
	switch topic {
	case BlockTopic:
		topicName = fmt.Sprintf("/optimism/%s/0/blocks", n.config.Rollup.L2ChainID.String())
	default:
		return fmt.Errorf("unknown topic: %v", topic)
	}
	t, err := n.gs.Join(topicName)
	if err != nil {
		return err
	}
	n.topics[topic] = t
	return nil
}

func (n *P2PNode) WaitForPeerCountOnTopic(topic GossipTopic, expected int) error {
	t, ok := n.topics[topic]
	if !ok {
		return fmt.Errorf("not joined to topic %v", topic)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return e2eutils.WaitFor(ctx, 1*time.Second, func() (bool, error) {
		return expected == len(t.ListPeers()), nil
	})
}

func (n *P2PNode) PublishGossip(ctx context.Context, topic GossipTopic, msg []byte) error {
	t, ok := n.topics[topic]
	if !ok {
		return fmt.Errorf("not joined to topic %v", topic)
	}
	return t.Publish(ctx, msg)
}

func (n *P2PNode) DisconnectPeer(peerId peer.ID) error {
	return n.host.Network().ClosePeer(peerId)
}

func (n *P2PNode) ConnectPeer(ctx context.Context, peerId peer.ID) error {
	_, err := n.host.Network().DialPeer(ctx, peerId)
	return err
}
