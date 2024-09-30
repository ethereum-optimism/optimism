package entrydb

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
)

type EntryStore[T EntryType] interface {
	Size() int64
	LastEntryIdx() EntryIdx
	Read(idx EntryIdx) (Entry[T], error)
	Append(entries ...Entry[T]) error
	Truncate(idx EntryIdx) error
	Close() error
}

type Metrics interface {
	RecordDBEntryCount(count int64)
	RecordDBSearchEntriesRead(count int64)
}

type IndexKey interface {
	comparable
	String() string
}

type IndexState[T EntryType, K IndexKey] interface {
	NextIndex() EntryIdx
	Key() (k K, ok bool)
	Incomplete() bool
	ApplyEntry(entry Entry[T]) error

	Out() []Entry[T]
	ClearOut()
}

type IndexDriver[T EntryType, K IndexKey, S IndexState[T, K]] interface {
	// Less compares the primary key. To allow binary search over the index.
	Less(a, b K) bool
	// Copy copies an index state. To allow state-snapshots without copy, for conditional iteration.
	Copy(src, dst S)
	// NewState creates an empty state, with the given index as next target input.
	NewState(nextIndex EntryIdx) S
	// KeyFromCheckpoint is called to turn an entry at a SearchCheckpointFrequency interval into a primary key.
	KeyFromCheckpoint(e Entry[T]) (K, error)
	// ValidEnd inspects if we can truncate the DB and leave the given entry as last entry.
	ValidEnd(e Entry[T]) bool
	// SearchCheckpointFrequency returns a constant, the interval of how far apart the guaranteed checkpoint entries are.
	SearchCheckpointFrequency() uint64
}

type DB[T EntryType, K IndexKey, S IndexState[T, K], D IndexDriver[T, K, S]] struct {
	log    log.Logger
	m      Metrics
	store  EntryStore[T]
	rwLock sync.RWMutex

	HeadState S

	driver D
}

func (db *DB[T, K, S, D]) LastEntryIdx() EntryIdx {
	return db.store.LastEntryIdx()
}

func (db *DB[T, K, S, D]) Init(trimToLastSealed bool) error {
	defer db.updateEntryCountMetric() // Always update the entry count metric after init completes
	if trimToLastSealed {
		if err := db.trimToLastSealed(); err != nil {
			return fmt.Errorf("failed to trim invalid trailing entries: %w", err)
		}
	}
	if db.LastEntryIdx() < 0 {
		// Database is empty.
		// Make a state that is ready to apply the genesis block on top of as first entry.
		// This will infer into a checkpoint (half of the block seal here)
		// and is then followed up with canonical-hash entry of genesis.
		db.HeadState = db.driver.NewState(0)
		return nil
	}
	// start at the last checkpoint,
	// and then apply any remaining changes on top, to hydrate the state.
	searchCheckpointFrequency := EntryIdx(db.driver.SearchCheckpointFrequency())
	lastCheckpoint := (db.LastEntryIdx() / searchCheckpointFrequency) * searchCheckpointFrequency
	i := db.newIterator(lastCheckpoint)
	if err := i.End(); err != nil {
		return fmt.Errorf("failed to init from remaining trailing data: %w", err)
	}
	db.HeadState = i.current
	return nil
}

func (db *DB[T, K, S, D]) trimToLastSealed() error {
	i := db.LastEntryIdx()
	for ; i >= 0; i-- {
		entry, err := db.store.Read(i)
		if err != nil {
			return fmt.Errorf("failed to read %v to check for trailing entries: %w", i, err)
		}
		if db.driver.ValidEnd(entry) {
			break
		}
	}
	if i < db.LastEntryIdx() {
		db.log.Warn("Truncating unexpected trailing entries", "prev", db.LastEntryIdx(), "new", i)
		// trim such that the last entry is the canonical-hash we identified
		return db.store.Truncate(i)
	}
	return nil
}

func (db *DB[T, K, S, D]) updateEntryCountMetric() {
	db.m.RecordDBEntryCount(db.store.Size())
}

// NewIteratorFor returns an iterator that will have traversed everything that was returned as true by the given lessFn.
// It may return an ErrSkipped if some data is known, but no data is known to be less than the requested key.
// It may return ErrFuture if no data is known at all.
func (db *DB[T, K, S, D]) NewIteratorFor(lessFn func(key K) bool) (Iterator[T, K, S], error) {
	return db.newIteratorFor(lessFn)
}

func (db *DB[T, K, S, D]) newIteratorExactlyAt(at K) (*iterator[T, K, S, D], error) {
	iter, err := db.newIteratorFor(func(key K) bool {
		return db.driver.Less(key, at) || key == at
	})
	if err != nil {
		return nil, err
	}
	k, ok := iter.State().Key()
	if !ok { // we should have stopped at complete data
		return nil, ErrDataCorruption
	}
	if k != at { // we found data less than the key, but not exactly equal to it
		return nil, ErrFuture
	}
	return iter, nil
}

func (db *DB[T, K, S, D]) newIteratorFor(lessFn func(key K) bool) (*iterator[T, K, S, D], error) {
	// Find a checkpoint before (not at) the requested key,
	// so we can read the value data corresponding to the key into the iterator state.
	searchCheckpointIndex, err := db.searchCheckpoint(lessFn)
	if errors.Is(err, io.EOF) {
		// Did not find a checkpoint to start reading from so the log cannot be present.
		return nil, ErrFuture
	} else if err != nil {
		return nil, err
	}
	// The iterator did not consume the checkpoint yet, it's positioned right at it.
	// So we can call NextBlock() and get the checkpoint itself as first entry.
	iter := db.newIterator(searchCheckpointIndex)
	if err != nil {
		return nil, err
	}
	defer func() {
		db.m.RecordDBSearchEntriesRead(iter.entriesRead)
	}()
	err = iter.TraverseConditional(func(state S) error {
		at, ok := state.Key()
		if !ok {
			return errors.New("expected complete state")
		}
		if !lessFn(at) {
			return ErrStop
		}
		return nil
	})
	if err == nil {
		panic("expected any error, good or bad, on stop")
	}
	if errors.Is(err, ErrStop) {
		err = nil
	}
	if err != nil {
		return nil, err
	}
	return iter, nil
}

// newIterator creates an iterator at the given index.
// None of the iterator attributes will be ready for reads,
// but the entry at the given index will be first read when using the iterator.
func (db *DB[T, K, S, D]) newIterator(index EntryIdx) *iterator[T, K, S, D] {
	return &iterator[T, K, S, D]{
		db:      db,
		current: db.driver.NewState(index),
	}
}

// searchCheckpoint performs a binary search of the searchCheckpoint entries
// to find the closest one with an equal or lower derivedFrom block number and equal or lower derived block number.
// Returns the index of the searchCheckpoint to begin reading from or an error.
func (db *DB[T, K, S, D]) searchCheckpoint(lessFn func(key K) bool) (EntryIdx, error) {
	if db.HeadState.NextIndex() == 0 {
		return 0, ErrFuture // empty DB, everything is in the future
	}
	searchCheckpointFrequency := EntryIdx(db.driver.SearchCheckpointFrequency())
	n := (db.LastEntryIdx() / searchCheckpointFrequency) + 1
	// Define: x is the array of known checkpoints
	// Invariant: x[i] <= target, x[j] > target.
	i, j := EntryIdx(0), n
	for i+1 < j { // i is inclusive, j is exclusive.
		// Get the checkpoint exactly in-between,
		// bias towards a higher value if an even number of checkpoints.
		// E.g. i=3 and j=4 would not run, since i + 1 < j
		// E.g. i=3 and j=5 leaves checkpoints 3, 4, and we pick 4 as pivot
		// E.g. i=3 and j=6 leaves checkpoints 3, 4, 5, and we pick 4 as pivot
		//
		// The following holds: i ≤ h < j
		h := EntryIdx((uint64(i) + uint64(j)) >> 1)
		checkpoint, err := db.readSearchCheckpoint(h * searchCheckpointFrequency)
		if err != nil {
			return 0, fmt.Errorf("failed to read entry %v: %w", h, err)
		}
		if lessFn(checkpoint) {
			i = h
		} else {
			j = h
		}
	}
	if i+1 != j {
		panic("expected to have 1 checkpoint left")
	}
	result := i * searchCheckpointFrequency
	checkpoint, err := db.readSearchCheckpoint(result)
	if err != nil {
		return 0, fmt.Errorf("failed to read final search checkpoint result: %w", err)
	}
	if !lessFn(checkpoint) {
		return 0, fmt.Errorf("missing data, earliest search checkpoint is %s, but is not before target: %w", checkpoint, ErrSkipped)
	}
	return result, nil
}

// Rewind the database to remove any blocks after headBlockNum
// The block at headBlockNum itself is not removed.
func (db *DB[T, K, S, D]) Rewind(newHead K) error {
	db.rwLock.Lock()
	defer db.rwLock.Unlock()
	// Even if the last fully-processed block matches headBlockNum,
	// we might still have trailing log events to get rid of.
	iter, err := db.newIteratorExactlyAt(newHead)
	if err != nil {
		return err
	}
	// Truncate to contain idx+1 entries, since indices are 0 based,
	// this deletes everything after idx
	if err := db.store.Truncate(iter.NextIndex()); err != nil {
		return fmt.Errorf("failed to truncate to %s: %w", newHead, err)
	}
	// Use db.init() to find the state for the new latest entry
	if err := db.Init(true); err != nil {
		return fmt.Errorf("failed to find new last entry context: %w", err)
	}
	return nil
}

// debug util to log the last 10 entries of the chain
func (db *DB[T, K, S, D]) debugTip() {
	for x := 0; x < 10; x++ {
		index := db.LastEntryIdx() - EntryIdx(x)
		if index < 0 {
			continue
		}
		e, err := db.store.Read(index)
		if err == nil {
			db.log.Debug("tip", "index", index, "type", e.Type())
		}
	}
}

func (db *DB[T, K, S, D]) Flush() error {
	out := db.HeadState.Out()
	nextIndex := db.HeadState.NextIndex()
	for i, e := range out {
		db.log.Trace("appending entry", "type", e.Type(), "entry", hexutil.Bytes(e[:]),
			"next", int(nextIndex)-len(out)+i)
	}
	if err := db.store.Append(out...); err != nil {
		return fmt.Errorf("failed to append entries: %w", err)
	}
	db.HeadState.ClearOut()
	db.updateEntryCountMetric()
	return nil
}

func (db *DB[T, K, S, D]) readSearchCheckpoint(entryIdx EntryIdx) (K, error) {
	data, err := db.store.Read(entryIdx)
	if err != nil {
		var k K
		return k, fmt.Errorf("failed to read entry %v: %w", entryIdx, err)
	}
	return db.driver.KeyFromCheckpoint(data)
}

func (db *DB[T, K, S, D]) Close() error {
	return db.store.Close()
}
