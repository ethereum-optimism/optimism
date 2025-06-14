package rwel

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// PromoteCrossSafeEvent signals that a block can be promoted to cross-safe.
type PromoteCrossSafeEvent struct {
	ID eth.BlockID
}

func (ev PromoteCrossSafeEvent) String() string {
	return "promote-cross-safe"
}

func (e *RWEL) onPromoteCrossSafe(ctx context.Context, ev PromoteCrossSafeEvent) {
	ref, err := e.engine.BlockRefByHash(ctx, ev.ID.Hash)
	if err != nil {
		e.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
			Err: fmt.Errorf("cannot promote cross-safe block, failed to retrieve it: %w", err)})
		return
	}
	e.onForkchoiceUpdateRequest(ctx, ForkchoiceUpdateRequestEvent{
		LocalUnsafe: e.state.LocalUnsafe(),
		CrossSafe:   ref,
		Finalized:   e.state.Finalized(),
	})
}
