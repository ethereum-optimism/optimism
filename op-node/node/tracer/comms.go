package tracer

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-node/rollup/status"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

// Tracer configures the OpNode to share events
type Tracer interface {
	OnNewL1Head(ctx context.Context, sig eth.L1BlockRef)
	OnUnsafeL2Payload(ctx context.Context, from peer.ID, payload *eth.ExecutionPayloadEnvelope)
	OnPublishL2Payload(ctx context.Context, payload *eth.ExecutionPayloadEnvelope)
}

type TracePublishBlockEvent struct {
	Envelope *eth.ExecutionPayloadEnvelope
}

func (ev TracePublishBlockEvent) String() string {
	return "trace-publish-event"
}

// TracerDeriver hooks a Tracer up to the event system as deriver
type TracerDeriver struct {
	tracer Tracer
	ctx    context.Context
	cancel context.CancelFunc
}

var _ event.Deriver = (*TracerDeriver)(nil)
var _ event.Unattacher = (*TracerDeriver)(nil)

func NewTracerDeriver(tracer Tracer) *TracerDeriver {
	ctx, cancel := context.WithCancel(context.Background())
	return &TracerDeriver{
		tracer: tracer,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (t *TracerDeriver) OnEvent(ctx context.Context, ev event.Event) bool {
	switch x := ev.(type) {
	case status.L1UnsafeEvent:
		t.tracer.OnNewL1Head(t.ctx, x.L1Unsafe)
	case p2p.ReceivedBlockEvent:
		t.tracer.OnUnsafeL2Payload(t.ctx, x.From, x.Envelope)
	case TracePublishBlockEvent:
		t.tracer.OnPublishL2Payload(t.ctx, x.Envelope)
	default:
		return false
	}
	return true
}

func (t *TracerDeriver) Unattach() {
	t.cancel()
}
