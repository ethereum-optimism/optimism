package node

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/l2"
	"github.com/libp2p/go-libp2p-core/peer"
)

// Tracer configures the OpNode to share events
type Tracer interface {
	OnNewL1Head(ctx context.Context, sig eth.L1BlockRef)
	OnUnsafeL2Payload(ctx context.Context, from peer.ID, payload *l2.ExecutionPayload)
	OnPublishL2Payload(ctx context.Context, payload *l2.ExecutionPayload)
}

type noOpTracer struct{}

func (n noOpTracer) OnNewL1Head(ctx context.Context, sig eth.L1BlockRef) {}

func (n noOpTracer) OnUnsafeL2Payload(ctx context.Context, from peer.ID, payload *l2.ExecutionPayload) {
}

func (n noOpTracer) OnPublishL2Payload(ctx context.Context, payload *l2.ExecutionPayload) {}

var _ Tracer = (*noOpTracer)(nil)
