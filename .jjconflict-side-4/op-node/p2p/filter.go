package p2p

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// FilterSelf implements GossipIn and filters blocks published by ourselves.
type FilterSelf struct {
	self  peer.ID
	inner GossipIn
}

var _ GossipIn = (*FilterSelf)(nil)

func NewFilterSelf(self peer.ID, inner GossipIn) *FilterSelf {
	return &FilterSelf{self: self, inner: inner}
}

func (f *FilterSelf) OnUnsafeL2Payload(ctx context.Context, from peer.ID, msg *eth.ExecutionPayloadEnvelope) error {
	if f.self == from {
		return nil // ignore blocks that we published ourselves
	}
	return f.inner.OnUnsafeL2Payload(ctx, from, msg)
}
