package controller

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/derive2"
	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/rwel"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/superevents"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// Complete derivation of pipelines with attributes
func (c *PipelineState) maybeProcessAttributes() {
	if c.deriveTask.IsBusy() {
		return
	}
	// check if it has ready attribs ready
	if c.attributes == nil {
		return
	}

	// Plan A: If there is a RWEL that can process the block, do that.
	if r, ok := First[*RWELState](c.state.IterRWELs(func(el *RWELState) bool {
		return c.attributes.Parent.Number == el.localUnsafe.Number
	})); ok {
		// process it
		ctx := rwel.WithID(c.rootCtx, r.ID())
		c.deriveTask.Emit(ctx, rwel.BuildStartEvent{
			Attributes: c.attributes,
		}, c.awaitBuildStarted)
		return
	}

	// Plan B: If there is a RWEL that can consolidate the block, do that.
	if r, ok := First[*RWELState](c.state.IterRWELs(func(el *RWELState) bool {
		return c.attributes.Parent.Number < el.localUnsafe.Number
	})); ok {
		// try to consolidate it
		// (if parent does not match, we'll get a temporary error back)
		ctx := rwel.WithID(c.rootCtx, r.ID())
		c.deriveTask.Emit(ctx, rwel.TryConsolidateAttributesEvent{
			Attributes: c.attributes,
		}, c.awaitConsolidate)
	}

	// Plan C: If there is a REL that consolidate the block, do that.
	// TODO
}

func (c *PipelineState) awaitConsolidate(ctx context.Context, ev event.Event) {
	switch x := ev.(type) {
	case rwel.ConfirmConsolidateAttributesEvent:
		c.confirming = x.Ref
		c.deriveTask.Emit(ctx, derive2.ConfirmAttributesEvent{
			Confirmed: x.Ref,
		}, c.awaitConfirmed)
	case rwel.FailConsolidateAttributesEvent:
		// TODO
	case rollup.EngineTemporaryErrorEvent:
		// TODO
	}
}

func (c *PipelineState) awaitBuildStarted(ctx context.Context, ev event.Event) {
	switch x := ev.(type) {
	case rwel.BuildStartedEvent:
		c.deriveTask.Emit(ctx, rwel.BuildSealEvent{
			Info:         x.Info,
			BuildStarted: x.BuildStarted,
			DerivedFrom:  x.DerivedFrom,
		}, c.awaitBuildSealed)
	case rwel.BuildInvalidEvent:

	}
}
func (c *PipelineState) awaitBuildSealed(ctx context.Context, ev event.Event) {
	switch x := ev.(type) {
	case rwel.BuildSealedEvent:
		c.confirming = x.Ref
		c.deriveTask.Emit(ctx, rwel.PayloadProcessEvent{
			DerivedFrom:  x.DerivedFrom,
			BuildStarted: x.BuildStarted,
			Envelope:     x.Envelope,
		}, c.awaitProcessed)
	case rwel.PayloadSealInvalidEvent:
	}
}

func (c *PipelineState) awaitProcessed(ctx context.Context, ev event.Event) {
	switch x := ev.(type) {
	case rwel.PayloadSuccessEvent:
		c.envelope = x.Envelope
		// TODO: maybe also fork to publish a block
		c.deriveTask.Emit(ctx, rwel.PromoteLocalUnsafeEvent{
			Ref: x.Envelope.BlockRef(),
		}, c.awaitCanonical)
	case rwel.PayloadInvalidEvent:
	}
}

func (c *PipelineState) awaitCanonical(ctx context.Context, ev event.Event) {
	switch x := ev.(type) {
	case rwel.LocalUnsafeUpdateEvent:
		if c.confirming.Hash != x.Ref.Hash {
			panic("unexpected block update")
		}
		c.deriveTask.Emit(ctx, derive2.ConfirmAttributesEvent{
			Confirmed: c.confirming,
		}, c.awaitConfirmed)
	}
}

// If attribs were confirmed, then update the local-safe pre-state
func (c *PipelineState) awaitConfirmed(ctx context.Context, ev event.Event) {
	switch ev.(type) {
	case derive.DeriverMoreEvent:
		id := rwel.IDFromContext(ctx)
		c.deriveTask.Emit(ctx, superevents.LocalDerivedEvent{
			ChainID: c.chainID,
			Derived: types.DerivedBlockRefPair{
				Source:  c.lastL1Source,
				Derived: c.confirming.BlockRef(),
			},
			NodeID: id.String(),
		}, c.awaitStored)
	}
}

func (c *PipelineState) awaitStored(ctx context.Context, ev event.Event) {
	switch ev.(type) {
	case Err:
		// TODO: if DB does not match
	}
}
