package rwel

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// PromoteFinalizedEvent signals that a block can be marked as finalized.
type PromoteFinalizedEvent struct {
	ID eth.BlockID
}

func (ev PromoteFinalizedEvent) String() string {
	return "promote-finalized"
}

func (e *RWEL) onPromoteFinalized(ctx context.Context, ev PromoteFinalizedEvent) {
	if ev.ID.Number < e.state.Finalized().Number {
		e.log.Error("Cannot rewind finality,",
			"attempted", ev.ID, "finalized", e.state.Finalized())
		return
	}
	if ev.ID.Number > e.state.CrossSafe().Number {
		e.log.Error("Block must be cross-safe before it can be finalized",
			"attempted", ev.ID, "safe", e.state.CrossSafe())
		return
	}
	ref, err := e.engine.BlockRefByHash(ctx, ev.ID.Hash)
	if err != nil {
		e.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
			Err: fmt.Errorf("cannot promote finalized block, failed to retrieve it: %w", err)})
		return
	}
	e.onForkchoiceUpdateRequest(ctx, ForkchoiceUpdateRequestEvent{
		LocalUnsafe: e.state.LocalUnsafe(),
		CrossSafe:   e.state.CrossSafe(),
		Finalized:   ref,
	})
}
