package logs

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

const (
	searchCheckpointFrequency    = 256
	eventFlagHasExecutingMessage = byte(1)
)

var (
	// ErrLogOutOfOrder happens when you try to add a log to the DB,
	// but it does not actually fit onto the latest data (by being too old or new).
	ErrLogOutOfOrder = errors.New("log out of order")
	// ErrDataCorruption happens when the underlying DB has some I/O issue
	ErrDataCorruption = errors.New("data corruption")
	// ErrSkipped happens when we try to retrieve data that is not available (pruned)
	// It may also happen if we erroneously skip data, that was not considered a conflict, if the DB is corrupted.
	ErrSkipped = errors.New("skipped data")
	// ErrFuture happens when data is just not yet available
	ErrFuture = errors.New("future data")
	// ErrConflict happens when we know for sure that there is different canonical data
	ErrConflict = errors.New("conflicting data")
)

type Metrics interface {
	RecordDBEntryCount(count int64)
	RecordDBSearchEntriesRead(count int64)
}

type EntryStore interface {
	Size() int64
	LastEntryIdx() entrydb.EntryIdx
	Read(idx entrydb.EntryIdx) (entrydb.Entry, error)
	Append(entries ...entrydb.Entry) error
	Truncate(idx entrydb.EntryIdx) error
	Close() error
}

// DB implements an append only database for log data and cross-chain dependencies.
//
// To keep the append-only format, reduce data size, and support reorg detection and registering of executing-messages:
//
// Use a fixed 24 bytes per entry.
//
// Data is an append-only log, that can be binary searched for any necessary event data.
type DB struct {
	log    log.Logger
	m      Metrics
	store  EntryStore
	rwLock sync.RWMutex

	lastEntryContext logContext
}

func NewFromFile(logger log.Logger, m Metrics, path string, trimToLastSealed bool) (*DB, error) {
	store, err := entrydb.NewEntryDB(logger, path)
	if err != nil {
		return nil, fmt.Errorf("failed to open DB: %w", err)
	}
	return NewFromEntryStore(logger, m, store, trimToLastSealed)
}

func NewFromEntryStore(logger log.Logger, m Metrics, store EntryStore, trimToLastSealed bool) (*DB, error) {
	db := &DB{
		log:   logger,
		m:     m,
		store: store,
	}
	if err := db.init(trimToLastSealed); err != nil {
		return nil, fmt.Errorf("failed to init database: %w", err)
	}
	return db, nil
}

func (db *DB) lastEntryIdx() entrydb.EntryIdx {
	return db.store.LastEntryIdx()
}

func (db *DB) init(trimToLastSealed bool) error {
	defer db.updateEntryCountMetric() // Always update the entry count metric after init completes
	if trimToLastSealed {
		if err := db.trimToLastSealed(); err != nil {
			return fmt.Errorf("failed to trim invalid trailing entries: %w", err)
		}
	}
	if db.lastEntryIdx() < 0 {
		// Database is empty.
		// Make a state that is ready to apply the genesis block on top of as first entry.
		// This will infer into a checkpoint (half of the block seal here)
		// and is then followed up with canonical-hash entry of genesis.
		db.lastEntryContext = logContext{
			nextEntryIndex: 0,
			blockHash:      common.Hash{},
			blockNum:       0,
			timestamp:      0,
			logsSince:      0,
			logHash:        common.Hash{},
			execMsg:        nil,
			out:            nil,
		}
		return nil
	}
	// start at the last checkpoint,
	// and then apply any remaining changes on top, to hydrate the state.
	lastCheckpoint := (db.lastEntryIdx() / searchCheckpointFrequency) * searchCheckpointFrequency
	i := db.newIterator(lastCheckpoint)
	i.current.need.Add(entrydb.FlagCanonicalHash)
	if err := i.End(); err != nil {
		return fmt.Errorf("failed to init from remaining trailing data: %w", err)
	}
	db.lastEntryContext = i.current
	return nil
}

func (db *DB) trimToLastSealed() error {
	i := db.lastEntryIdx()
	for ; i >= 0; i-- {
		entry, err := db.store.Read(i)
		if err != nil {
			return fmt.Errorf("failed to read %v to check for trailing entries: %w", i, err)
		}
		if entry.Type() == entrydb.TypeCanonicalHash {
			// only an executing hash, indicating a sealed block, is a valid point for restart
			break
		}
	}
	if i < db.lastEntryIdx() {
		db.log.Warn("Truncating unexpected trailing entries", "prev", db.lastEntryIdx(), "new", i)
		// trim such that the last entry is the canonical-hash we identified
		return db.store.Truncate(i)
	}
	return nil
}

func (db *DB) updateEntryCountMetric() {
	db.m.RecordDBEntryCount(db.store.Size())
}

func (db *DB) IteratorStartingAt(i entrydb.EntryIdx) (Iterator, error) {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()
	if i > db.lastEntryContext.nextEntryIndex {
		return nil, ErrFuture
	}
	// TODO(#12031): Workaround while we not have IteratorStartingAt(heads.HeadPointer):
	// scroll back from the index, to find block info.
	idx := i
	for ; idx >= 0; i-- {
		entry, err := db.store.Read(idx)
		if err != nil {
			if errors.Is(err, io.EOF) {
				continue // traverse to when we did have blocks
			}
			return nil, err
		}
		if entry.Type() == entrydb.TypeSearchCheckpoint {
			break
		}
		if idx == 0 {
			return nil, fmt.Errorf("empty DB, no block entry, cannot start at %d", i)
		}
	}
	iter := db.newIterator(idx)
	for iter.NextIndex() < i {
		if _, err := iter.next(); err != nil {
			return nil, errors.New("failed to process back up to the head pointer")
		}
	}
	return iter, nil
}

// FindSealedBlock finds the requested block, to check if it exists,
// returning the next index after it where things continue from.
// returns ErrFuture if the block is too new to be able to tell
// returns ErrDifferent if the known block does not match
func (db *DB) FindSealedBlock(block eth.BlockID) (nextEntry entrydb.EntryIdx, err error) {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()
	iter, err := db.newIteratorAt(block.Number, 0)
	if errors.Is(err, ErrFuture) {
		return 0, fmt.Errorf("block %d is not known yet: %w", block.Number, ErrFuture)
	} else if err != nil {
		return 0, fmt.Errorf("failed to find sealed block %d: %w", block.Number, err)
	}
	h, _, ok := iter.SealedBlock()
	if !ok {
		panic("expected block")
	}
	if block.Hash != h {
		return 0, fmt.Errorf("queried %s but got %s at number %d: %w", block.Hash, h, block.Number, ErrConflict)
	}
	return iter.NextIndex(), nil
}

// LatestSealedBlockNum returns the block number of the block that was last sealed,
// or ok=false if there is no sealed block (i.e. empty DB)
func (db *DB) LatestSealedBlockNum() (n uint64, ok bool) {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()
	if !db.lastEntryContext.hasCompleteBlock() {
		if db.lastEntryContext.blockNum == 0 {
			db.log.Debug("No DB contents yet")
		} else {
			db.log.Debug("New block is already in progress", "num", db.lastEntryContext.blockNum)
		}
	}
	return db.lastEntryContext.blockNum, true
}

// Get returns the hash of the log at the specified blockNum (of the sealed block)
// and logIdx (of the log after the block), or an error if the log is not found.
func (db *DB) Get(blockNum uint64, logIdx uint32) (common.Hash, error) {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()
	hash, _, err := db.findLogInfo(blockNum, logIdx)
	return hash, err
}

// Contains returns no error iff the specified logHash is recorded in the specified blockNum and logIdx.
// If the log is out of reach, then ErrFuture is returned.
// If the log is determined to conflict with the canonical chain, then ErrConflict is returned.
// logIdx is the index of the log in the array of all logs in the block.
// This can be used to check the validity of cross-chain interop events.
func (db *DB) Contains(blockNum uint64, logIdx uint32, logHash common.Hash) (entrydb.EntryIdx, error) {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()
	db.log.Trace("Checking for log", "blockNum", blockNum, "logIdx", logIdx, "hash", logHash)

	evtHash, iter, err := db.findLogInfo(blockNum, logIdx)
	if err != nil {
		return 0, err // may be ErrConflict if the block does not have as many logs
	}
	db.log.Trace("Found initiatingEvent", "blockNum", blockNum, "logIdx", logIdx, "hash", evtHash)
	// Found the requested block and log index, check if the hash matches
	if evtHash != logHash {
		return 0, fmt.Errorf("payload hash mismatch: expected %s, got %s", logHash, evtHash)
	}
	return iter.NextIndex(), nil
}

func (db *DB) findLogInfo(blockNum uint64, logIdx uint32) (common.Hash, Iterator, error) {
	if blockNum == 0 {
		return common.Hash{}, nil, ErrConflict // no logs in block 0
	}
	// blockNum-1, such that we find a log that came after the parent num-1 was sealed.
	// logIdx, such that all entries before logIdx can be skipped, but logIdx itself is still readable.
	iter, err := db.newIteratorAt(blockNum-1, logIdx)
	if errors.Is(err, ErrFuture) {
		db.log.Trace("Could not find log yet", "blockNum", blockNum, "logIdx", logIdx)
		return common.Hash{}, nil, err
	} else if err != nil {
		db.log.Error("Failed searching for log", "blockNum", blockNum, "logIdx", logIdx)
		return common.Hash{}, nil, err
	}
	if err := iter.NextInitMsg(); err != nil {
		return common.Hash{}, nil, fmt.Errorf("failed to read initiating message %d, on top of block %d: %w", logIdx, blockNum, err)
	}
	if _, x, ok := iter.SealedBlock(); !ok {
		panic("expected block")
	} else if x < blockNum-1 {
		panic(fmt.Errorf("bug in newIteratorAt, expected to have found parent block %d but got %d", blockNum-1, x))
	} else if x > blockNum-1 {
		return common.Hash{}, nil, fmt.Errorf("log does not exist, found next block already: %w", ErrConflict)
	}
	logHash, x, ok := iter.InitMessage()
	if !ok {
		panic("expected init message")
	} else if x != logIdx {
		panic(fmt.Errorf("bug in newIteratorAt, expected to have found log %d but got %d", logIdx, x))
	}
	return logHash, iter, nil
}

// newIteratorAt returns an iterator ready after the given sealed block number,
// and positioned such that the next log-read on the iterator return the log with logIndex, if any.
// It may return an ErrNotFound if the block number is unknown,
// or if there are just not that many seen log events after the block as requested.
func (db *DB) newIteratorAt(blockNum uint64, logIndex uint32) (*iterator, error) {
	// find a checkpoint before or exactly when blockNum was sealed,
	// and have processed up to but not including [logIndex] number of logs (i.e. all prior logs, if any).
	searchCheckpointIndex, err := db.searchCheckpoint(blockNum, logIndex)
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
	iter.current.need.Add(entrydb.FlagCanonicalHash)
	defer func() {
		db.m.RecordDBSearchEntriesRead(iter.entriesRead)
	}()
	// First walk up to the block that we are sealed up to (incl.)
	for {
		if _, n, _ := iter.SealedBlock(); n == blockNum { // we may already have it exactly
			break
		}
		if err := iter.NextBlock(); errors.Is(err, ErrFuture) {
			db.log.Trace("ran out of data, could not find block", "nextIndex", iter.NextIndex(), "target", blockNum)
			return nil, ErrFuture
		} else if err != nil {
			db.log.Error("failed to read next block", "nextIndex", iter.NextIndex(), "target", blockNum, "err", err)
			return nil, err
		}
		h, num, ok := iter.SealedBlock()
		if !ok {
			panic("expected sealed block")
		}
		db.log.Trace("found sealed block", "num", num, "hash", h)
		if num < blockNum {
			continue
		}
		if num != blockNum { // block does not contain
			return nil, fmt.Errorf("looking for %d, but already at %d: %w", blockNum, num, ErrConflict)
		}
		break
	}
	// Now walk up to the number of seen logs that we want to have processed.
	// E.g. logIndex == 2, need to have processed index 0 and 1,
	// so two logs before quiting (and not 3 to then quit after).
	for iter.current.logsSince < logIndex {
		if err := iter.NextInitMsg(); err == io.EOF {
			return nil, ErrFuture
		} else if err != nil {
			return nil, err
		}
		_, num, ok := iter.SealedBlock()
		if !ok {
			panic("expected sealed block")
		}
		if num > blockNum {
			// we overshot, the block did not contain as many seen log events as requested
			return nil, ErrConflict
		}
		_, idx, ok := iter.InitMessage()
		if !ok {
			panic("expected initializing message")
		}
		if idx+1 < logIndex {
			continue
		}
		if idx+1 == logIndex {
			break // the NextInitMsg call will position the iterator at the re
		}
		return nil, fmt.Errorf("unexpected log-skip at block %d log %d", blockNum, idx)
	}
	return iter, nil
}

// newIterator creates an iterator at the given index.
// None of the iterator attributes will be ready for reads,
// but the entry at the given index will be first read when using the iterator.
func (db *DB) newIterator(index entrydb.EntryIdx) *iterator {
	return &iterator{
		db: db,
		current: logContext{
			nextEntryIndex: index,
		},
	}
}

// searchCheckpoint performs a binary search of the searchCheckpoint entries
// to find the closest one with an equal or lower block number and equal or lower amount of seen logs.
// Returns the index of the searchCheckpoint to begin reading from or an error.
func (db *DB) searchCheckpoint(sealedBlockNum uint64, logsSince uint32) (entrydb.EntryIdx, error) {
	n := (db.lastEntryIdx() / searchCheckpointFrequency) + 1
	// Define: x is the array of known checkpoints
	// Invariant: x[i] <= target, x[j] > target.
	i, j := entrydb.EntryIdx(0), n
	for i+1 < j { // i is inclusive, j is exclusive.
		// Get the checkpoint exactly in-between,
		// bias towards a higher value if an even number of checkpoints.
		// E.g. i=3 and j=4 would not run, since i + 1 < j
		// E.g. i=3 and j=5 leaves checkpoints 3, 4, and we pick 4 as pivot
		// E.g. i=3 and j=6 leaves checkpoints 3, 4, 5, and we pick 4 as pivot
		//
		// The following holds: i ≤ h < j
		h := entrydb.EntryIdx((uint64(i) + uint64(j)) >> 1)
		checkpoint, err := db.readSearchCheckpoint(h * searchCheckpointFrequency)
		if err != nil {
			return 0, fmt.Errorf("failed to read entry %v: %w", h, err)
		}
		if checkpoint.blockNum < sealedBlockNum ||
			(checkpoint.blockNum == sealedBlockNum && checkpoint.logsSince < logsSince) {
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
	if checkpoint.blockNum > sealedBlockNum ||
		(checkpoint.blockNum == sealedBlockNum && checkpoint.logsSince > logsSince) {
		return 0, fmt.Errorf("missing data, earliest search checkpoint is %d with %d logs, cannot find something before or at %d with %d logs: %w",
			checkpoint.blockNum, checkpoint.logsSince, sealedBlockNum, logsSince, ErrSkipped)
	}
	return result, nil
}

// debug util to log the last 10 entries of the chain
func (db *DB) debugTip() {
	for x := 0; x < 10; x++ {
		index := db.lastEntryIdx() - entrydb.EntryIdx(x)
		if index < 0 {
			continue
		}
		e, err := db.store.Read(index)
		if err == nil {
			db.log.Debug("tip", "index", index, "type", e.Type())
		}
	}
}

func (db *DB) flush() error {
	for i, e := range db.lastEntryContext.out {
		db.log.Trace("appending entry", "type", e.Type(), "entry", hexutil.Bytes(e[:]),
			"next", int(db.lastEntryContext.nextEntryIndex)-len(db.lastEntryContext.out)+i)
	}
	if err := db.store.Append(db.lastEntryContext.out...); err != nil {
		return fmt.Errorf("failed to append entries: %w", err)
	}
	db.lastEntryContext.out = db.lastEntryContext.out[:0]
	db.updateEntryCountMetric()
	return nil
}

func (db *DB) SealBlock(parentHash common.Hash, block eth.BlockID, timestamp uint64) error {
	db.rwLock.Lock()
	defer db.rwLock.Unlock()

	if err := db.lastEntryContext.SealBlock(parentHash, block, timestamp); err != nil {
		return fmt.Errorf("failed to seal block: %w", err)
	}
	db.log.Trace("Sealed block", "parent", parentHash, "block", block, "timestamp", timestamp)
	return db.flush()
}

func (db *DB) AddLog(logHash common.Hash, parentBlock eth.BlockID, logIdx uint32, execMsg *types.ExecutingMessage) error {
	db.rwLock.Lock()
	defer db.rwLock.Unlock()

	if err := db.lastEntryContext.ApplyLog(parentBlock, logIdx, logHash, execMsg); err != nil {
		return fmt.Errorf("failed to apply log: %w", err)
	}
	db.log.Trace("Applied log", "parentBlock", parentBlock, "logIndex", logIdx, "logHash", logHash, "executing", execMsg != nil)
	return db.flush()
}

// Rewind the database to remove any blocks after headBlockNum
// The block at headBlockNum itself is not removed.
func (db *DB) Rewind(newHeadBlockNum uint64) error {
	db.rwLock.Lock()
	defer db.rwLock.Unlock()
	// Even if the last fully-processed block matches headBlockNum,
	// we might still have trailing log events to get rid of.
	iter, err := db.newIteratorAt(newHeadBlockNum, 0)
	if err != nil {
		return err
	}
	// Truncate to contain idx+1 entries, since indices are 0 based,
	// this deletes everything after idx
	if err := db.store.Truncate(iter.NextIndex()); err != nil {
		return fmt.Errorf("failed to truncate to block %v: %w", newHeadBlockNum, err)
	}
	// Use db.init() to find the log context for the new latest log entry
	if err := db.init(true); err != nil {
		return fmt.Errorf("failed to find new last entry context: %w", err)
	}
	return nil
}

func (db *DB) readSearchCheckpoint(entryIdx entrydb.EntryIdx) (searchCheckpoint, error) {
	data, err := db.store.Read(entryIdx)
	if err != nil {
		return searchCheckpoint{}, fmt.Errorf("failed to read entry %v: %w", entryIdx, err)
	}
	return newSearchCheckpointFromEntry(data)
}

func (db *DB) Close() error {
	return db.store.Close()
}
