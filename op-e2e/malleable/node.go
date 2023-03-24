package malleable

import (
	"context"
	"fmt"
	"testing"

	log "github.com/ethereum/go-ethereum/log"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	connmgr "github.com/libp2p/go-libp2p/core/connmgr"
	host "github.com/libp2p/go-libp2p/core/host"
	p2pmetrics "github.com/libp2p/go-libp2p/core/metrics"
	peer "github.com/libp2p/go-libp2p/core/peer"

	eth "github.com/ethereum-optimism/optimism/op-node/eth"
	p2p "github.com/ethereum-optimism/optimism/op-node/p2p"
)

// MalleableNode is a slimmed down [opNode.NodeP2P] with configurable gossip.
// Discovery is not enabled for simplicity.
type MalleableNode struct {
	innerHost host.Host

	// p2p parameters
	gater   p2p.ConnectionGater
	connMgr connmgr.ConnManager

	// Gossip parameters
	gs    *pubsub.PubSub
	gsOut p2p.GossipOut
}

// NewMalleableNode creates a new [MalleableNode].
func NewMalleableNode(t *testing.T) (*MalleableNode, error) {
	var m MalleableNode
	if err := m.Init(t); err != nil {
		return nil, err
	}
	if err := m.Start(context.Background()); err != nil {
		return nil, err
	}
	return &m, nil
}

// Init builds a [MalleableNode].
func (m *MalleableNode) Init(t *testing.T) error {
	bwc := p2pmetrics.NewBandwidthCounter()

	snapLog := log.New()
	snapLog.SetHandler(log.DiscardHandler())

	conf, err := DefaultConfig()
	if err != nil {
		return err
	}

	// Build the inner host
	m.innerHost, err = m.BuildHost(conf, snapLog, bwc)
	if err != nil {
		return fmt.Errorf("failed to start p2p host: %w", err)
	}

	if err := m.SetupGossip(t, conf, snapLog); err != nil {
		return fmt.Errorf("failed to setup gossip: %w", err)
	}

	return nil
}

// Start starts the [MalleableNode].
func (m *MalleableNode) Start(ctx context.Context) error {

	// TODO: start the malleable node

	return nil
}

// OnUnsafeL1Payload implements the [p2p.GossipIn] interface.
func (m *MalleableNode) OnUnsafeL2Payload(ctx context.Context, from peer.ID, msg *eth.ExecutionPayload) error {

	// TODO: Implement this

	return nil
}
