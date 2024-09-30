package fromda

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
)

type DB struct {
	log    log.Logger
	inner  *entrydb.DB[EntryType, Key, *state, driver]
	rwLock sync.RWMutex
}

func (db *DB) AddDerived(derivedFrom eth.BlockID, derived eth.BlockRef) error {
	db.rwLock.Lock()
	defer db.rwLock.Unlock()

	if err := db.inner.HeadState.AddDerived(derivedFrom, derived); err != nil {
		return fmt.Errorf("failed to add derived block derivedFrom: %s, derived: %s, err: %w", derivedFrom, derived, err)
	}
	db.log.Trace("Added derived block", "derivedFrom", derivedFrom, "derived", derived)
	return db.inner.Flush()
}

func (db *DB) SealDerivedFrom(derivedFrom eth.BlockRef) error {
	db.rwLock.Lock()
	defer db.rwLock.Unlock()

	if err := db.inner.HeadState.SealDerivedFrom(derivedFrom); err != nil {
		return fmt.Errorf("failed to seal derived from: %w", err)
	}
	db.log.Trace("Sealed derived-from", "derivedFrom", derivedFrom)
	return db.inner.Flush()
}

func (db *DB) Rewind(derivedFrom uint64) error {
	return db.inner.Rewind(Key{DerivedFrom: derivedFrom, Derived: 0})
}

// LatestDerivedFrom returns the last known primary key (the L1 block)
func (db *DB) LatestDerivedFrom() (ref eth.BlockID, ok bool) {
	db.rwLock.Lock()
	defer db.rwLock.Unlock()
	state := db.inner.HeadState
	if state.Incomplete() {
		return eth.BlockID{}, false
	}
	return state.derived, true
}

// LatestDerived returns the last known value (the L2 block that was derived)
func (db *DB) LatestDerived() (ref eth.BlockID, ok bool) {
	db.rwLock.Lock()
	defer db.rwLock.Unlock()
	state := db.inner.HeadState
	if state.Incomplete() {
		return eth.BlockID{}, false
	}
	return state.derived, true
}

// LastDerivedAt returns the last L2 block derived from the given L1 block
func (db *DB) LastDerivedAt(derivedFrom eth.BlockID) (eth.BlockID, error) {
	db.rwLock.Lock()
	defer db.rwLock.Unlock()
	iter, err := db.inner.NewIteratorFor(func(key Key) bool {
		return key.DerivedFrom < derivedFrom.Number
	})
	if err != nil {
		return eth.BlockID{}, err
	}
	if errors.Is(err, entrydb.ErrStop) {
		err = nil
	}
	if err != nil {
		return eth.BlockID{}, err
	}
	state := iter.State()
	if state.Incomplete() {
		return eth.BlockID{}, entrydb.ErrDataCorruption
	}
	if state.derivedFrom != derivedFrom { // did not reach derived From yet
		return eth.BlockID{}, entrydb.ErrFuture
	}
	return state.derived, nil
}

// TODO do we want to expose an iterator interface?
//type Iterator interface {
//	TraverseConditional(fn func(*state) error) error
//}
//
//func (db *DB) IteratorStartingFor() (Iterator, error) {
//	return db.inner.NewIteratorFor()
//}

// DerivedFrom determines where a L2 block was derived from.
func (db *DB) DerivedFrom(derived eth.BlockID) (eth.BlockID, error) {
	// search to the last point before the data
	iter, err := db.inner.NewIteratorFor(func(key Key) bool {
		return key.Derived < derived.Number
	})
	if err != nil {
		return eth.BlockID{}, err
	}
	// go forward and read the data
	err = iter.TraverseConditional(func(state *state) error {
		v, ok := state.Derived()
		if !ok {
			return nil
		}
		if v.Number > derived.Number {
			return entrydb.ErrStop
		}
		return nil
	})
	if errors.Is(err, entrydb.ErrStop) {
		err = nil
	}
	if err != nil {
		return eth.BlockID{}, err
	}
	state := iter.State()
	if state.Incomplete() {
		return eth.BlockID{}, entrydb.ErrDataCorruption
	}
	if state.derived != derived {
		return eth.BlockID{}, entrydb.ErrConflict
	}
	return state.derivedFrom, nil
}
