package engine

import (
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

func (eq *EngDeriver) onBuildInvalid(ev BuildInvalidEvent) {
	eq.log.Warn("could not process payload attributes", "err", ev.Err)

	// Deposit transaction execution errors are suppressed in the execution engine, but if the
	// block is somehow invalid, there is nothing we can do to recover & we should exit.
	if ev.Attributes.Attributes.IsDepositsOnly() {
		eq.log.Error("deposit only block was invalid", "parent", ev.Attributes.Parent, "err", ev.Err)
		eq.emitter.Emit(rollup.CriticalErrorEvent{Err: fmt.Errorf("failed to process block with only deposit transactions: %w", ev.Err)})
		return
	}

	// TODO: not sure if we have to check IsDerived, can we land here outside of derivation?
	if eq.cfg.IsHolocene(ev.Attributes.DerivedFrom.Time) && ev.Attributes.IsDerived() {
		eq.log.Warn("Holocene active, retrying deposits-only attributes")
		retryingAttributes := ev.Attributes.WithDepositsOnly()

		// let external derivers know so they can adapt accordingly
		eq.emitter.Emit(derive.RetryingDepositsPayloadAttributesEvent{
			OriginalAttributes: ev.Attributes,
			RetryingAttributes: retryingAttributes,
			Err:                ev.Err,
		})

		// attempt retry internally
		eq.emitter.Emit(BuildStartEvent{retryingAttributes})
		return
	}

	// Revert the pending safe head to the safe head.
	eq.ec.SetPendingSafeL2Head(eq.ec.SafeL2Head())
	// suppress the error b/c we want to retry with the next batch from the batch queue
	// If there is no valid batch the node will eventually force a deposit only block. If
	// the deposit only block fails, this will return the critical error above.

	// Try to restore to previous known unsafe chain.
	eq.ec.SetBackupUnsafeL2Head(eq.ec.BackupUnsafeL2Head(), true)

	// drop the payload without inserting it into the engine

	// Signal that we deemed the attributes as unfit
	eq.emitter.Emit(InvalidPayloadAttributesEvent(ev))
}
