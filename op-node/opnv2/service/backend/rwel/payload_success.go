package rwel

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type PayloadSuccessEvent struct {
	// payload is promoted to pending-safe if non-zero
	DerivedFrom eth.L1BlockRef

	BuildStarted  time.Time
	InsertStarted time.Time

	Envelope *eth.ExecutionPayloadEnvelope
}

func (ev PayloadSuccessEvent) String() string {
	return "payload-success"
}

func (eq *RWEL) onPayloadSuccess(ctx context.Context, ev PayloadSuccessEvent) {
	// TODO: on invalidate-block replacements we need to update cross-safe at the same time

	eq.emitter.Emit(ctx, PromoteLocalUnsafeEvent{Ref: ev.Envelope.BlockRef()})
}
