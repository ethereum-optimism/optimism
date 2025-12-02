package engine

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
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

func (e *EngineController) onBuildInvalid(ctx context.Context, ev BuildInvalidEvent) {
	e.log.Warn("could not process payload attributes", "err", ev.Err)

	// Deposit transaction execution errors are suppressed in the execution engine, but if the
	// block is somehow invalid, there is nothing we can do to recover & we should exit.
	if ev.Attributes.Attributes.IsDepositsOnly() {
		e.log.Error("deposit only block was invalid", "parent", ev.Attributes.Parent, "err", ev.Err)
		e.emitter.Emit(ctx, rollup.CriticalErrorEvent{
			Err: fmt.Errorf("failed to process block with only deposit transactions: %w", ev.Err),
		})
		return
	}

	if ev.Attributes.IsDerived() && e.rollupCfg.IsHolocene(ev.Attributes.DerivedFrom.Time) {
		e.emitDepositsOnlyPayloadAttributesRequest(ctx, ev.Attributes.Parent.ID(), ev.Attributes.DerivedFrom)
		return
	}

	// Revert the pending safe head to the safe head.
	e.SetPendingSafeL2Head(e.safeHead)
	// suppress the error b/c we want to retry with the next batch from the batch queue
	// If there is no valid batch the node will eventually force a deposit only block. If
	// the deposit only block fails, this will return the critical error above.

	// Try to restore to previous known unsafe chain.
	e.SetBackupUnsafeL2Head(e.backupUnsafeHead, true)

	// drop the payload without inserting it into the engine

	// Signal that we deemed the attributes as unfit
	e.emitter.Emit(ctx, InvalidPayloadAttributesEvent(ev))
}

func (e *EngineController) emitDepositsOnlyPayloadAttributesRequest(ctx context.Context, parent eth.BlockID, derivedFrom eth.L1BlockRef) {
	e.log.Warn("Holocene active, requesting deposits-only attributes", "parent", parent, "derived_from", derivedFrom)
	// request deposits-only version
	e.emitter.Emit(ctx, derive.DepositsOnlyPayloadAttributesRequestEvent{
		Parent:      parent,
		DerivedFrom: derivedFrom,
	})
}
