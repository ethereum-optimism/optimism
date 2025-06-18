package p2p

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

type ReceivedBlockEvent struct {
	ChainID  eth.ChainID
	From     peer.ID
	Envelope *eth.ExecutionPayloadEnvelope
	event.Ctx
}

func (ev ReceivedBlockEvent) String() string {
	return "received-block-event"
}

type BlockReceiverMetrics interface {
	RecordReceivedUnsafePayload(payload *eth.ExecutionPayloadEnvelope)
}

// BlockReceiver can be plugged into the P2P gossip stack,
// to receive payloads as ReceivedBlockEvent events.
type BlockReceiver struct {
	log     log.Logger
	emitter event.Emitter
	metrics BlockReceiverMetrics
}

var _ GossipIn = (*BlockReceiver)(nil)

func NewBlockReceiver(log log.Logger, em event.Emitter, metrics BlockReceiverMetrics) *BlockReceiver {
	return &BlockReceiver{
		log:     log,
		emitter: em,
		metrics: metrics,
	}
}

func (g *BlockReceiver) OnUnsafeL2Payload(ctx context.Context, from peer.ID, msg *eth.ExecutionPayloadEnvelope) error {
	g.log.Debug("Received signed execution payload from p2p",
		"id", msg.ExecutionPayload.ID(),
		"peer", from, "txs", len(msg.ExecutionPayload.Transactions))
	g.metrics.RecordReceivedUnsafePayload(msg)
	g.emitter.Emit(ReceivedBlockEvent{From: from, Envelope: msg, Ctx: event.WrapCtx(ctx)})
	return nil
}
