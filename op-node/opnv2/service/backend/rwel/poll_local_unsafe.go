package rwel

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type PollLocalUnsafeRequestEvent struct {
}

func (ev PollLocalUnsafeRequestEvent) String() string {
	return "poll-local-unsafe-request"
}

func (eq *RWEL) onPollLocalUnsafeRequest(ctx context.Context, ev PollLocalUnsafeRequestEvent) {
	latest, err := eq.engine.BlockRefByLabel(ctx, eth.Unsafe)
	if err != nil {
		eq.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
			Err: fmt.Errorf("failed to get latest block: %w", err),
		})
		return
	}
	if eq.state.LocalUnsafe() != latest {
		// suggest we need to poll the safe head too, since we observed an inconsistency
		eq.emitter.Emit(ctx, PollCrossSafeRequestEvent{})
		eq.state.SetLocalUnsafe(latest)
	}
	// Even if no change, just emit the update event, to show we are done polling.
	eq.emitter.Emit(ctx, LocalUnsafeUpdateEvent{
		Ref: eq.state.LocalUnsafe(),
	})
}
