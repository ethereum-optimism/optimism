package rwel

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// PromoteLocalUnsafeEvent signals that the given block may now become a canonical local-unsafe block.
type PromoteLocalUnsafeEvent struct {
	Ref eth.BlockRef
}

func (ev PromoteLocalUnsafeEvent) String() string {
	return "promote-local-unsafe"
}

func (e *RWEL) onPromoteLocalUnsafe(ctx context.Context, ev PromoteLocalUnsafeEvent) {
	e.onForkchoiceUpdateRequest(ctx, ForkchoiceUpdateRequestEvent{
		LocalUnsafe: ev.Ref,
		CrossSafe:   e.state.CrossSafe(),
		Finalized:   e.state.Finalized(),
	})
}
