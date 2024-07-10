package engine

import (
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"

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

	// Count the number of deposits to see if the tx list is deposit only.
	depositCount := 0
	for _, tx := range ev.Attributes.Attributes.Transactions {
		if len(tx) > 0 && tx[0] == types.DepositTxType {
			depositCount += 1
		}
	}
	// Deposit transaction execution errors are suppressed in the execution engine, but if the
	// block is somehow invalid, there is nothing we can do to recover & we should exit.
	if len(ev.Attributes.Attributes.Transactions) == depositCount {
		eq.log.Error("deposit only block was invalid", "parent", ev.Attributes.Parent, "err", ev.Err)
		eq.emitter.Emit(rollup.CriticalErrorEvent{Err: fmt.Errorf("failed to process block with only deposit transactions: %w", ev.Err)})
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
