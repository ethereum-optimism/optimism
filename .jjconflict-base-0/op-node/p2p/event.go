package p2p

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type BlockReceiverMetrics interface {
	RecordReceivedUnsafePayload(payload *eth.ExecutionPayloadEnvelope)
}

type SyncDeriver interface {
	OnUnsafeL2Payload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope)
}

type Tracer interface {
	OnUnsafeL2Payload(ctx context.Context, from peer.ID, payload *eth.ExecutionPayloadEnvelope)
}

// BlockReceiver can be plugged into the P2P gossip stack,
// to receive payloads and call syncDeriver to toss unsafe payload
type BlockReceiver struct {
	log     log.Logger
	metrics BlockReceiverMetrics

	// syncDeriver embedded for triggering unsafe payload sync via p2p
	syncDeriver SyncDeriver
	// Tracer embedded for tracing unsafe payload
	tracer Tracer
}

var _ GossipIn = (*BlockReceiver)(nil)

func NewBlockReceiver(log log.Logger, metrics BlockReceiverMetrics, syncDeriver SyncDeriver, tracer Tracer) *BlockReceiver {
	return &BlockReceiver{
		log:         log,
		metrics:     metrics,
		syncDeriver: syncDeriver,
		tracer:      tracer,
	}
}

func (g *BlockReceiver) OnUnsafeL2Payload(ctx context.Context, from peer.ID, msg *eth.ExecutionPayloadEnvelope) error {
	g.log.Debug("Received signed execution payload from p2p",
		"id", msg.ExecutionPayload.ID(),
		"peer", from, "txs", len(msg.ExecutionPayload.Transactions))
	g.metrics.RecordReceivedUnsafePayload(msg)
	g.syncDeriver.OnUnsafeL2Payload(ctx, msg)
	if g.tracer != nil { // tracer is optional
		g.tracer.OnUnsafeL2Payload(ctx, from, msg)
	}
	return nil
}
