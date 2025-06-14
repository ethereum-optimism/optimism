package controller

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/rwel"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/superevents"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func (v *ChainDBState) maybeReplaceBlock() {
	if v.Replacement.IsBusy() {
		return
	}
	if v.invalidated == (types.DerivedBlockRefPair{}) {
		return
	}

	r, ok := First[*RWELState](v.state.IterRWELs(
		LatestAtLeast(v.invalidated.Derived.Number)))
	if !ok {
		// TODO try consolidate invalidation with REL nodes instead
		return
	}

	// TODO: some engines might already have replaced it.
	// If they already have, we can send back an event to skip the re-processing.

	ctx := rwel.WithID(v.rootCtx, r.ID())
	v.Replacement.Emit(ctx,
		rwel.InvalidateBlockRequestEvent{
			Invalidated: types.BlockSealFromRef(v.invalidated.Derived),
		}, v.awaitReplacementAttributes)
}

func (v *ChainDBState) awaitReplacementAttributes(ctx context.Context, ev event.Event) {
	switch x := ev.(type) {
	case rwel.BuildReplacementBlockEvent:
		v.replacementComplete = &types.BlockReplacement{
			Invalidated: x.Invalidated.Hash,
		}
		// find an RWEL to process the replacement for us
		r, ok := First(v.state.IterRWELs(LatestAtLeast(
			v.replacementAttributes.Parent.Number)))
		if ok {
			// TODO lock the RWEL also
			ctx = rwel.WithID(ctx, r.ID())
			v.Replacement.Emit(ctx, rwel.BuildStartEvent{
				Attributes: v.replacementAttributes,
			}, v.awaitReplacementBuilt)
			return
		}
	}
}

func (v *ChainDBState) awaitReplacementBuilt(ctx context.Context, ev event.Event) {
	switch x := ev.(type) {
	case rwel.PayloadSuccessEvent:
		v.replacementComplete.Replacement = x.Envelope.BlockRef()
		v.Replacement.Emit(ctx, rwel.PromoteLocalUnsafeEvent{
			Ref: x.Envelope.BlockRef(),
		}, v.awaitReplacementCanonical)
	}
}

func (v *ChainDBState) awaitReplacementCanonical(ctx context.Context, ev event.Event) {
	switch x := ev.(type) {
	case rwel.LocalUnsafeUpdateEvent:
		_ = x
		v.Replacement.Emit(ctx, superevents.ReplaceBlockEvent{
			Replacement: *v.replacementComplete,
		}, nil)
	}
}
