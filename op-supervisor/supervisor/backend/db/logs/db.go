package logs

import (
	"errors"
	"fmt"
	"io"
	"math"
	"sync"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/types"
	"github.com/ethereum/go-ethereum/log"
)

const (
	searchCheckpointFrequency = 256

	eventFlagIncrementLogIdx     = byte(1)
	eventFlagHasExecutingMessage = byte(1) << 1
)

const (
	typeSearchCheckpoint byte = iota
	typeCanonicalHash
	typeInitiatingEvent
	typeExecutingLink
	typeExecutingCheck
)

var (
	ErrLogOutOfOrder  = errors.New("log out of order")
	ErrDataCorruption = errors.New("data corruption")
	ErrNotFound       = errors.New("not found")
)

type Metrics interface {
	RecordDBEntryCount(count int64)
	RecordDBSearchEntriesRead(count int64)
}

type logContext struct {
	blockNum uint64
	logIdx   uint32
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
//
// Rules:
// if entry_index % 256 == 0: must be type 0. For easy binary search.
// type 1 always adjacent to type 0
// type 2 "diff" values are offsets from type 0 values (always within 256 entries range)
// type 3 always after type 2
// type 4 always after type 3
//
// Types (<type> = 1 byte):
// type 0: "search checkpoint" <type><uint64 block number: 8 bytes><uint32 event index offset: 4 bytes><uint64 timestamp: 8 bytes> = 20 bytes
// type 1: "canonical hash" <type><parent blockhash truncated: 20 bytes> = 21 bytes
// type 2: "initiating event" <type><blocknum diff: 1 byte><event flags: 1 byte><event-hash: 20 bytes> = 23 bytes
// type 3: "executing link" <type><chain: 4 bytes><blocknum: 8 bytes><event index: 3 bytes><uint64 timestamp: 8 bytes> = 24 bytes
// type 4: "executing check" <type><event-hash: 20 bytes> = 21 bytes
// other types: future compat. E.g. for linking to L1, registering block-headers as a kind of initiating-event, tracking safe-head progression, etc.
//
// Right-pad each entry that is not 24 bytes.
//
// event-flags: each bit represents a boolean value, currently only two are defined
// * event-flags & 0x01 - true if the log index should increment. Should only be false when the event is immediately after a search checkpoint and canonical hash
// * event-flags & 0x02 - true if the initiating event has an executing link that should follow. Allows detecting when the executing link failed to write.
// event-hash: H(origin, timestamp, payloadhash); enough to check identifier matches & payload matches.
type DB struct {
	log    log.Logger
	m      Metrics
	store  EntryStore
	rwLock sync.RWMutex

	lastEntryContext logContext
}

func NewFromFile(logger log.Logger, m Metrics, path string) (*DB, error) {
	store, err := entrydb.NewEntryDB(logger, path)
	if err != nil {
		return nil, fmt.Errorf("failed to open DB: %w", err)
	}
	return NewFromEntryStore(logger, m, store)
}

func NewFromEntryStore(logger log.Logger, m Metrics, store EntryStore) (*DB, error) {
	db := &DB{
		log:   logger,
		m:     m,
		store: store,
	}
	if err := db.init(); err != nil {
		return nil, fmt.Errorf("failed to init database: %w", err)
	}
	return db, nil
}

func (db *DB) lastEntryIdx() entrydb.EntryIdx {
	return db.store.LastEntryIdx()
}

func (db *DB) init() error {
	defer db.updateEntryCountMetric() // Always update the entry count metric after init completes
	if err := db.trimInvalidTrailingEntries(); err != nil {
		return fmt.Errorf("failed to trim invalid trailing entries: %w", err)
	}
	if db.lastEntryIdx() < 0 {
		// Database is empty so no context to load
		return nil
	}

	lastCheckpoint := (db.lastEntryIdx() / searchCheckpointFrequency) * searchCheckpointFrequency
	i, err := db.newIterator(lastCheckpoint)
	if err != nil {
		return fmt.Errorf("failed to create iterator at last search checkpoint: %w", err)
	}
	// Read all entries until the end of the file
	for {
		_, _, _, err := i.NextLog()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return fmt.Errorf("failed to init from existing entries: %w", err)
		}
	}
	db.lastEntryContext = i.current
	return nil
}

func (db *DB) trimInvalidTrailingEntries() error {
	i := db.lastEntryIdx()
	for ; i >= 0; i-- {
		entry, err := db.store.Read(i)
		if err != nil {
			return fmt.Errorf("failed to read %v to check for trailing entries: %w", i, err)
		}
		if entry[0] == typeExecutingCheck {
			// executing check is a valid final entry
			break
		}
		if entry[0] == typeInitiatingEvent {
			evt, err := newInitiatingEventFromEntry(entry)
			if err != nil {
				// Entry is invalid, keep walking backwards
				continue
			}
			if !evt.hasExecMsg {
				// init event with no exec msg is a valid final entry
				break
			}
		}
	}
	if i < db.lastEntryIdx() {
		db.log.Warn("Truncating unexpected trailing entries", "prev", db.lastEntryIdx(), "new", i)
		return db.store.Truncate(i)
	}
	return nil
}

func (db *DB) updateEntryCountMetric() {
	db.m.RecordDBEntryCount(db.store.Size())
}

func (db *DB) LatestBlockNum() uint64 {
	return db.lastEntryContext.blockNum
}

// ClosestBlockInfo returns the block number and hash of the highest recorded block at or before blockNum.
// Since block data is only recorded in search checkpoints, this may return an earlier block even if log data is
// recorded for the requested block.
func (db *DB) ClosestBlockInfo(blockNum uint64) (uint64, types.TruncatedHash, error) {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()
	checkpointIdx, err := db.searchCheckpoint(blockNum, math.MaxUint32)
	if err != nil {
		return 0, types.TruncatedHash{}, fmt.Errorf("no checkpoint at or before block %v found: %w", blockNum, err)
	}
	checkpoint, err := db.readSearchCheckpoint(checkpointIdx)
	if err != nil {
		return 0, types.TruncatedHash{}, fmt.Errorf("failed to reach checkpoint: %w", err)
	}
	entry, err := db.readCanonicalHash(checkpointIdx + 1)
	if err != nil {
		return 0, types.TruncatedHash{}, fmt.Errorf("failed to read canonical hash: %w", err)
	}
	return checkpoint.blockNum, entry.hash, nil
}

// Contains return true iff the specified logHash is recorded in the specified blockNum and logIdx.
// logIdx is the index of the log in the array of all logs the block.
// This can be used to check the validity of cross-chain interop events.
func (db *DB) Contains(blockNum uint64, logIdx uint32, logHash types.TruncatedHash) (bool, error) {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()
	db.log.Trace("Checking for log", "blockNum", blockNum, "logIdx", logIdx, "hash", logHash)

	evtHash, _, err := db.findLogInfo(blockNum, logIdx)
	if errors.Is(err, ErrNotFound) {
		// Did not find a log at blockNum and logIdx
		return false, nil
	} else if err != nil {
		return false, err
	}
	db.log.Trace("Found initiatingEvent", "blockNum", blockNum, "logIdx", logIdx, "hash", evtHash)
	// Found the requested block and log index, check if the hash matches
	return evtHash == logHash, nil
}

// Executes checks if the log identified by the specific block number and log index, has an ExecutingMessage associated
// with it that needs to be checked as part of interop validation.
// logIdx is the index of the log in the array of all logs the block.
// Returns the ExecutingMessage if it exists, or ExecutingMessage{} if the log is found but has no ExecutingMessage.
// Returns ErrNotFound if the specified log does not exist in the database.
func (db *DB) Executes(blockNum uint64, logIdx uint32) (types.ExecutingMessage, error) {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()
	_, iter, err := db.findLogInfo(blockNum, logIdx)
	if err != nil {
		return types.ExecutingMessage{}, err
	}
	execMsg, err := iter.ExecMessage()
	if err != nil {
		return types.ExecutingMessage{}, fmt.Errorf("failed to read executing message: %w", err)
	}
	return execMsg, nil
}

func (db *DB) findLogInfo(blockNum uint64, logIdx uint32) (types.TruncatedHash, *iterator, error) {
	entryIdx, err := db.searchCheckpoint(blockNum, logIdx)
	if errors.Is(err, io.EOF) {
		// Did not find a checkpoint to start reading from so the log cannot be present.
		return types.TruncatedHash{}, nil, ErrNotFound
	} else if err != nil {
		return types.TruncatedHash{}, nil, err
	}

	i, err := db.newIterator(entryIdx)
	if err != nil {
		return types.TruncatedHash{}, nil, fmt.Errorf("failed to create iterator: %w", err)
	}
	db.log.Trace("Starting search", "entry", entryIdx, "blockNum", i.current.blockNum, "logIdx", i.current.logIdx)
	defer func() {
		db.m.RecordDBSearchEntriesRead(i.entriesRead)
	}()
	for {
		evtBlockNum, evtLogIdx, evtHash, err := i.NextLog()
		if errors.Is(err, io.EOF) {
			// Reached end of log without finding the event
			return types.TruncatedHash{}, nil, ErrNotFound
		} else if err != nil {
			return types.TruncatedHash{}, nil, fmt.Errorf("failed to read next log: %w", err)
		}
		if evtBlockNum == blockNum && evtLogIdx == logIdx {
			db.log.Trace("Found initiatingEvent", "blockNum", evtBlockNum, "logIdx", evtLogIdx, "hash", evtHash)
			return evtHash, i, nil
		}
		if evtBlockNum > blockNum || (evtBlockNum == blockNum && evtLogIdx > logIdx) {
			// Progressed past the requested log without finding it.
			return types.TruncatedHash{}, nil, ErrNotFound
		}
	}
}

func (db *DB) newIterator(startCheckpointEntry entrydb.EntryIdx) (*iterator, error) {
	checkpoint, err := db.readSearchCheckpoint(startCheckpointEntry)
	if err != nil {
		return nil, fmt.Errorf("failed to read search checkpoint entry %v: %w", startCheckpointEntry, err)
	}
	startIdx := startCheckpointEntry + 2
	firstEntry, err := db.store.Read(startIdx)
	if errors.Is(err, io.EOF) {
		// There should always be an entry after a checkpoint and canonical hash so an EOF here is data corruption
		return nil, fmt.Errorf("%w: no entry after checkpoint and canonical hash at %v", ErrDataCorruption, startCheckpointEntry)
	} else if err != nil {
		return nil, fmt.Errorf("failed to read first entry to iterate %v: %w", startCheckpointEntry+2, err)
	}
	startLogCtx := logContext{
		blockNum: checkpoint.blockNum,
		logIdx:   checkpoint.logIdx,
	}
	// Handle starting from a checkpoint after initiating-event but before its executing-link or executing-check
	if firstEntry[0] == typeExecutingLink || firstEntry[0] == typeExecutingCheck {
		if firstEntry[0] == typeExecutingLink {
			// The start checkpoint was between the initiating event and the executing link
			// Step back to read the initiating event. The checkpoint block data will be for the initiating event
			startIdx = startCheckpointEntry - 1
		} else {
			// The start checkpoint was between the executing link and the executing check
			// Step back to read the initiating event. The checkpoint block data will be for the initiating event
			startIdx = startCheckpointEntry - 2
		}
		initEntry, err := db.store.Read(startIdx)
		if err != nil {
			return nil, fmt.Errorf("failed to read prior initiating event: %w", err)
		}
		initEvt, err := newInitiatingEventFromEntry(initEntry)
		if err != nil {
			return nil, fmt.Errorf("invalid initiating event at idx %v: %w", startIdx, err)
		}
		startLogCtx = initEvt.preContext(startLogCtx)
	}
	i := &iterator{
		db: db,
		// +2 to skip the initial search checkpoint and the canonical hash event after it
		nextEntryIdx: startIdx,
		current:      startLogCtx,
	}
	return i, nil
}

// searchCheckpoint performs a binary search of the searchCheckpoint entries to find the closest one at or before
// the requested log.
// Returns the index of the searchCheckpoint to begin reading from or an error
func (db *DB) searchCheckpoint(blockNum uint64, logIdx uint32) (entrydb.EntryIdx, error) {
	n := (db.lastEntryIdx() / searchCheckpointFrequency) + 1
	// Define x[-1] < target and x[n] >= target.
	// Invariant: x[i-1] < target, x[j] >= target.
	i, j := entrydb.EntryIdx(0), n
	for i < j {
		h := entrydb.EntryIdx(uint64(i+j) >> 1) // avoid overflow when computing h
		checkpoint, err := db.readSearchCheckpoint(h * searchCheckpointFrequency)
		if err != nil {
			return 0, fmt.Errorf("failed to read entry %v: %w", h, err)
		}
		// i ≤ h < j
		if checkpoint.blockNum < blockNum || (checkpoint.blockNum == blockNum && checkpoint.logIdx < logIdx) {
			i = h + 1 // preserves x[i-1] < target
		} else {
			j = h // preserves x[j] >= target
		}
	}
	if i < n {
		checkpoint, err := db.readSearchCheckpoint(i * searchCheckpointFrequency)
		if err != nil {
			return 0, fmt.Errorf("failed to read entry %v: %w", i, err)
		}
		if checkpoint.blockNum == blockNum && checkpoint.logIdx == logIdx {
			// Found entry at requested block number and log index
			return i * searchCheckpointFrequency, nil
		}
	}
	if i == 0 {
		// There are no checkpoints before the requested blocks
		return 0, io.EOF
	}
	// Not found, need to start reading from the entry prior
	return (i - 1) * searchCheckpointFrequency, nil
}

func (db *DB) AddLog(logHash types.TruncatedHash, block eth.BlockID, timestamp uint64, logIdx uint32, execMsg *types.ExecutingMessage) error {
	db.rwLock.Lock()
	defer db.rwLock.Unlock()
	postState := logContext{
		blockNum: block.Number,
		logIdx:   logIdx,
	}
	if block.Number == 0 {
		return fmt.Errorf("%w: should not have logs in block 0", ErrLogOutOfOrder)
	}
	if db.lastEntryContext.blockNum > block.Number {
		return fmt.Errorf("%w: adding block %v, head block: %v", ErrLogOutOfOrder, block.Number, db.lastEntryContext.blockNum)
	}
	if db.lastEntryContext.blockNum == block.Number && db.lastEntryContext.logIdx+1 != logIdx {
		return fmt.Errorf("%w: adding log %v in block %v, but currently at log %v", ErrLogOutOfOrder, logIdx, block.Number, db.lastEntryContext.logIdx)
	}
	if db.lastEntryContext.blockNum < block.Number && logIdx != 0 {
		return fmt.Errorf("%w: adding log %v as first log in block %v", ErrLogOutOfOrder, logIdx, block.Number)
	}
	var entriesToAdd []entrydb.Entry
	newContext := db.lastEntryContext
	lastEntryIdx := db.lastEntryIdx()

	addEntry := func(entry entrydb.Entry) {
		entriesToAdd = append(entriesToAdd, entry)
		lastEntryIdx++
	}
	maybeAddCheckpoint := func() {
		if (lastEntryIdx+1)%searchCheckpointFrequency == 0 {
			addEntry(newSearchCheckpoint(block.Number, logIdx, timestamp).encode())
			addEntry(newCanonicalHash(types.TruncateHash(block.Hash)).encode())
			newContext = postState
		}
	}
	maybeAddCheckpoint()

	evt, err := newInitiatingEvent(newContext, postState.blockNum, postState.logIdx, logHash, execMsg != nil)
	if err != nil {
		return fmt.Errorf("failed to create initiating event: %w", err)
	}
	addEntry(evt.encode())

	if execMsg != nil {
		maybeAddCheckpoint()
		link, err := newExecutingLink(*execMsg)
		if err != nil {
			return fmt.Errorf("failed to create executing link: %w", err)
		}
		addEntry(link.encode())

		maybeAddCheckpoint()
		addEntry(newExecutingCheck(execMsg.Hash).encode())
	}
	if err := db.store.Append(entriesToAdd...); err != nil {
		return fmt.Errorf("failed to append entries: %w", err)
	}
	db.lastEntryContext = postState
	db.updateEntryCountMetric()
	return nil
}

// Rewind the database to remove any blocks after headBlockNum
// The block at headBlockNum itself is not removed.
func (db *DB) Rewind(headBlockNum uint64) error {
	db.rwLock.Lock()
	defer db.rwLock.Unlock()
	if headBlockNum >= db.lastEntryContext.blockNum {
		// Nothing to do
		return nil
	}
	// Find the last checkpoint before the block to remove
	idx, err := db.searchCheckpoint(headBlockNum+1, 0)
	if errors.Is(err, io.EOF) {
		// Requested a block prior to the first checkpoint
		// Delete everything without scanning forward
		idx = -1
	} else if err != nil {
		return fmt.Errorf("failed to find checkpoint prior to block %v: %w", headBlockNum, err)
	} else {
		// Scan forward from the checkpoint to find the first entry about a block after headBlockNum
		i, err := db.newIterator(idx)
		if err != nil {
			return fmt.Errorf("failed to create iterator when searching for rewind point: %w", err)
		}
		// If we don't find any useful logs after the checkpoint, we should delete the checkpoint itself
		// So move our delete marker back to include it as a starting point
		idx--
		for {
			blockNum, _, _, err := i.NextLog()
			if errors.Is(err, io.EOF) {
				// Reached end of file, we need to keep everything
				return nil
			} else if err != nil {
				return fmt.Errorf("failed to find rewind point: %w", err)
			}
			if blockNum > headBlockNum {
				// Found the first entry we don't need, so stop searching and delete everything after idx
				break
			}
			// Otherwise we need all of the entries the iterator just read
			idx = i.nextEntryIdx - 1
		}
	}
	// Truncate to contain idx+1 entries, since indices are 0 based, this deletes everything after idx
	if err := db.store.Truncate(idx); err != nil {
		return fmt.Errorf("failed to truncate to block %v: %w", headBlockNum, err)
	}
	// Use db.init() to find the log context for the new latest log entry
	if err := db.init(); err != nil {
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

func (db *DB) readCanonicalHash(entryIdx entrydb.EntryIdx) (canonicalHash, error) {
	data, err := db.store.Read(entryIdx)
	if err != nil {
		return canonicalHash{}, fmt.Errorf("failed to read entry %v: %w", entryIdx, err)
	}
	return newCanonicalHashFromEntry(data)
}

func (db *DB) Close() error {
	return db.store.Close()
}
