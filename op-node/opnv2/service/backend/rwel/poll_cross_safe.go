package rwel

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type PollCrossSafeRequestEvent struct {
}

func (ev PollCrossSafeRequestEvent) String() string {
	return "poll-cross-safe-request"
}

func (eq *RWEL) onPollCrossSafeRequest(ctx context.Context, ev PollCrossSafeRequestEvent) {
	crossSafe, err := eq.engine.BlockRefByLabel(ctx, eth.Safe)
	// Fall back to finalized, if the engine has not marked anything as "safe" yet
	if errors.Is(err, ethereum.NotFound) {
		eq.log.Warn("Engine does not have a 'safe' block yet, attempting to fall back to finalized now", "err", err)
		crossSafe = eq.state.Finalized()
	}
	if err != nil {
		eq.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
			Err: fmt.Errorf("failed to get cross-safe block: %w", err),
		})
		return
	}
	if eq.state.CrossSafe() != crossSafe {
		// suggest we need to poll the finalized head too, since we observed an inconsistency
		eq.emitter.Emit(ctx, PollFinalizedRequestEvent{})
		eq.state.SetCrossSafe(crossSafe)
	}
	// Even if no change, just emit the update event, to show we are done polling.
	eq.emitter.Emit(ctx, CrossSafeUpdateEvent{
		CrossSafe: crossSafe,
	})
}
