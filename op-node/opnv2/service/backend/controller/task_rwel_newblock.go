package controller

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/payloads"
	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/rwel"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

func (v *RWELState) maybeNewBlock() {
	if v.IncomingBlock.IsBusy() {
		return
	}

	// If not, then try get a payload
	pa, ok := v.state.Payloads(v.ChainID())
	if !ok {
		return
	}
	if v.localUnsafe.Number < pa.max.Number {
		nextNum := v.localUnsafe.Number + 1
		if v.syncing {
			// If we are syncing, then instead get the next payload after the sync-target.
			nextNum = v.syncTarget.Number + 1
		}
		v.IncomingBlock.Emit(v.rootCtx, payloads.PayloadRequestEvent{
			ChainID: v.ChainID(),
			Num:     nextNum,
		}, v.awaitReceiveBlock)
	}
}

func (v *RWELState) awaitReceiveBlock(ctx context.Context, ev event.Event) {
	switch x := ev.(type) {
	case payloads.PayloadResponseEvent:
		v.IncomingBlock.Emit(ctx, rwel.PayloadProcessEvent{
			DerivedFrom:  eth.L1BlockRef{},
			BuildStarted: v.state.Now(),
			Envelope:     x.Envelope,
		}, v.awaitProcessBlock)
	case Err:
		// TODO log, backoff
	}
}

func (v *RWELState) awaitProcessBlock(ctx context.Context, ev event.Event) {
	switch x := ev.(type) {
	case rwel.PayloadSuccessEvent:
		v.IncomingBlock.Emit(ctx, rwel.PromoteLocalUnsafeEvent{
			Ref: x.Envelope.BlockRef(),
		}, v.awaitPromotedBlock)
	case rwel.PayloadInvalidEvent:
		// TODO
	}
}

func (v *RWELState) awaitPromotedBlock(ctx context.Context, ev event.Event) {
	switch x := ev.(type) {
	case rwel.LocalUnsafeUpdateEvent:
		v.localUnsafe = x.Ref
	default:
	}
}
