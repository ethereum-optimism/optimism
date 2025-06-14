package controller

import (
	"errors"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/superevents"
)

func (s *RWELState) maybeIndex() {
	db, ok := s.state.ChainDB(s.ChainID())
	if !ok {
		return
	}
	if db.chainIndexingWork.IsBusy() {
		return
	}
	if s.localUnsafe == (eth.BlockRef{}) {
		return // nothing to index yet
	}
	if s.syncing {
		return // should not pull data that is incomplete
	}

	// For any RWEL|REL, if DB local-unsafe head < RWEL|REL, then index it into DB.
	// Don't try to index data from an EL node that is actively doing EL sync.

	now := s.state.Now()
	if db.localUnsafe.Number < s.localUnsafe.Number &&
		!db.chainIndexingWork.IsBackedOff(now) {
		// Let's make the chain processor run.
		// No tasks possible because of old assumptions in chain indexer.
		db.chainIndexingWork.DoBackoff(errors.New("legacy indexing"), now)
		db.chainIndexingWork.Emit(s.rootCtx, superevents.ChainProcessEvent{
			ChainID: db.chainID,
		}, nil)
	}
}
