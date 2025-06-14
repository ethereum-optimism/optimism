package rwel

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

type PollFinalizedRequestEvent struct {
}

func (ev PollFinalizedRequestEvent) String() string {
	return "poll-finalized-request"
}

func (eq *RWEL) onPollFinalizedRequest(ctx context.Context, ev PollFinalizedRequestEvent) {
	finalized, err := eq.engine.BlockRefByLabel(ctx, eth.Finalized)
	// Fall back to genesis, if the engine has not marked anything as "finalized" yet
	if errors.Is(err, ethereum.NotFound) {
		eq.log.Warn("Engine does not have a 'finalized' block yet, attempting to fall back to genesis now", "err", err)
		finalized = eth.BlockRef{
			Hash:       eq.cfg.Genesis.L2.Hash,
			Number:     eq.cfg.Genesis.L2.Number,
			ParentHash: common.Hash{},
			Time:       eq.cfg.Genesis.L2Time,
		}
		err = nil
	}
	if err != nil {
		eq.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
			Err: fmt.Errorf("failed to get finalized block: %w", err),
		})
		return
	}
	if eq.state.Finalized() != finalized {
		eq.state.SetFinalized(finalized)
	}
	// Even if no change, just emit the update event, to show we are done polling.
	eq.emitter.Emit(ctx, FinalizedUpdateEvent{
		Ref: finalized,
	})
}
