package rwel

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

// BuildInvalidEvent is an internal engine event, to post-process upon invalid attributes.
// Not for temporary processing problems.
type BuildInvalidEvent struct {
	Attributes *derive.AttributesWithParent
	Err        error
}

func (ev BuildInvalidEvent) String() string {
	return "build-invalid"
}

// InvalidPayloadAttributesEvent is a signal to external derivers that the attributes were invalid.
type InvalidPayloadAttributesEvent struct {
	Attributes *derive.AttributesWithParent
	Err        error
}

func (ev InvalidPayloadAttributesEvent) String() string {
	return "invalid-payload-attributes"
}

func (eq *RWEL) onBuildInvalid(ctx context.Context, ev BuildInvalidEvent) {
	eq.log.Warn("could not process payload attributes", "err", ev.Err)

	// Deposit transaction execution errors are suppressed in the execution engine, but if the
	// block is somehow invalid, there is nothing we can do to recover & we should exit.
	if ev.Attributes.Attributes.IsDepositsOnly() {
		eq.log.Error("deposit only block was invalid", "parent", ev.Attributes.Parent, "err", ev.Err)
		eq.emitter.Emit(ctx, rollup.CriticalErrorEvent{
			Err: fmt.Errorf("failed to process block with only deposit transactions: %w", ev.Err),
		})
		return
	}

	if ev.Attributes.IsDerived() {
		req := derive.DepositsOnlyPayloadAttributesRequestEvent{
			Parent:      ev.Attributes.Parent.ID(),
			DerivedFrom: ev.Attributes.DerivedFrom,
		}
		eq.log.Warn("Payload building was invalid, requesting deposits-only attributes",
			"parent", req.Parent, "derived_from", req.DerivedFrom)
		eq.emitter.Emit(ctx, req)
		return
	}
	// drop the payload without inserting it into the engine

	// Signal that we deemed the attributes as unfit
	eq.emitter.Emit(ctx, InvalidPayloadAttributesEvent(ev))
}
