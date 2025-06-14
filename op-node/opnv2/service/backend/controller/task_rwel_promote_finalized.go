package controller

import "github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/rwel"

func (s *RWELState) maybePromoteFinalized() {
	if s.Forkchoice.IsBusy() {
		return
	}
	db, ok := s.state.ChainDB(s.ChainID())
	if !ok {
		return
	}
	if s.finalizedBackoff.IsBackedOff(s.state.Now()) {
		return
	}
	if db.finalized.Derived.Number > s.finalized.Number &&
		db.crossSafe.Derived.Number <= s.crossSafe.Number {
		// If EL is on a different chain then this will error, and backoff finality work
		s.Forkchoice.Emit(s.rootCtx, rwel.PromoteFinalizedEvent{
			ID: db.finalized.Derived.ID()}, nil)
		// TODO handle error
	}
}
