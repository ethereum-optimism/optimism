package controller

import (
	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/rwel"
)

func (s *RWELState) maybeELSync() {
	if s.ELSync.IsBusy() {
		return
	}
	db, ok := s.state.ChainDB(s.ChainID())
	if !ok {
		return
	}
	// for any RWEL, if DB local-unsafe head > RWEL, then trigger RWEL to try to sync
	if db.localUnsafe.Number > s.localUnsafe.Number {
		// If syncing, and the sync target is beyond the DB, then keep that target
		if s.syncing && s.syncTarget.Number > db.localUnsafe.Number {
			return
		}
		// If not syncing, or targeting something older/nothing, then update the target.
		s.ELSync.Emit(s.rootCtx, rwel.TriggerSyncEvent{Target: db.localUnsafe.ID()}, nil)
	}
}
