package op_e2e

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/node"
)

type FnTracer struct {
	OnNewL1HeadFn        func(ctx context.Context, sig eth.L1BlockRef)
	OnUnsafeL2PayloadFn  func(ctx context.Context, from peer.ID, payload *eth.ExecutionPayload)
	OnPublishL2PayloadFn func(ctx context.Context, payload *eth.ExecutionPayload)
}

func (n *FnTracer) OnNewL1Head(ctx context.Context, sig eth.L1BlockRef) {
	if n.OnNewL1HeadFn != nil {
		n.OnNewL1HeadFn(ctx, sig)
	}
}

func (n *FnTracer) OnUnsafeL2Payload(ctx context.Context, from peer.ID, payload *eth.ExecutionPayload) {
	if n.OnUnsafeL2PayloadFn != nil {
		n.OnUnsafeL2PayloadFn(ctx, from, payload)
	}
}

func (n *FnTracer) OnPublishL2Payload(ctx context.Context, payload *eth.ExecutionPayload) {
	if n.OnPublishL2PayloadFn != nil {
		n.OnPublishL2PayloadFn(ctx, payload)
	}
}

var _ node.Tracer = (*FnTracer)(nil)
