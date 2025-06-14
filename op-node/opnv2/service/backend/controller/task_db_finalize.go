package controller

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/superevents"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func (s *ChainDBState) maybeDBFinalize() {
	if !s.FinalizeWork.IsBusy() {
		return
	}
	l1State := s.state.L1State()
	// if L1 finality changed, then apply to DB finalized heads
	if l1State.finalizedL1 != (eth.BlockRef{}) {
		// TODO track L1 finality changes so we don't repeat
		return
	}
	if s.crossSafe == (types.DerivedBlockSealPair{}) {
		return // nothing to finalize
	}
	// TODO check for error backoff

	// If we have a cross-safe block that is derived
	// from a L1 block that is finalized,
	// then let's signal the finalized L1 again to the DB
	if s.crossSafe.Source.Number >= l1State.finalizedL1.Number &&
		s.finalized.Source.ID() != l1State.finalizedL1.ID() {
		s.FinalizeWork.Emit(s.rootCtx, superevents.FinalizedL1RequestEvent{
			FinalizedL1: l1State.finalizedL1}, s.awaitFinalized)
	}
}

func (s *ChainDBState) awaitFinalized(ctx context.Context, ev event.Event) {
	// TODO handle success/error
}
