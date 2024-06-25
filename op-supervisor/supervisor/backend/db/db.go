package db

import (
	"errors"
	"fmt"
	"io"
	"math"
	"sync"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

const (
	searchCheckpointFrequency = 256

	eventFlagIncrementLogIdx = byte(1)
	//eventFlagHasExecutingMessage = byte(1) << 1
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
)

type TruncatedHash [20]byte

type Metrics interface {
	RecordEntryCount(count int64)
	RecordSearchEntriesRead(count int64)
}

type logContext struct {
	blockNum uint64
	logIdx   uint32
}

type entryStore interface {
	Size() int64
	Read(idx int64) (entrydb.Entry, error)
	Append(entries ...entrydb.Entry) error
	Truncate(idx int64) error
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
	store  entryStore
	rwLock sync.RWMutex

	lastEntryContext logContext
}

func NewFromFile(logger log.Logger, m Metrics, path string) (*DB, error) {
	store, err := entrydb.NewEntryDB(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open DB: %w", err)
	}
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

func (db *DB) lastEntryIdx() int64 {
	return db.store.Size() - 1
}

func (db *DB) init() error {
	db.updateEntryCountMetric()
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

func (db *DB) updateEntryCountMetric() {
	db.m.RecordEntryCount(db.lastEntryIdx() + 1)
}

// ClosestBlockInfo returns the block number and hash of the highest recorded block at or before blockNum.
// Since block data is only recorded in search checkpoints, this may return an earlier block even if log data is
// recorded for the requested block.
func (db *DB) ClosestBlockInfo(blockNum uint64) (uint64, TruncatedHash, error) {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()
	checkpointIdx, err := db.searchCheckpoint(blockNum, math.MaxUint32)
	if err != nil {
		return 0, TruncatedHash{}, fmt.Errorf("no checkpoint at or before block %v found: %w", blockNum, err)
	}
	checkpoint, err := db.readSearchCheckpoint(checkpointIdx)
	if err != nil {
		return 0, TruncatedHash{}, fmt.Errorf("failed to reach checkpoint: %w", err)
	}
	entry, err := db.readCanonicalHash(checkpointIdx + 1)
	if err != nil {
		return 0, TruncatedHash{}, fmt.Errorf("failed to read canonical hash: %w", err)
	}
	return checkpoint.blockNum, entry.hash, nil
}

// Contains return true iff the specified logHash is recorded in the specified blockNum and logIdx.
// logIdx is the index of the log in the array of all logs the block.
func (db *DB) Contains(blockNum uint64, logIdx uint32, logHash TruncatedHash) (bool, error) {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()
	db.log.Trace("Checking for log", "blockNum", blockNum, "logIdx", logIdx, "hash", logHash)
	entryIdx, err := db.searchCheckpoint(blockNum, logIdx)
	if errors.Is(err, io.EOF) {
		// Did not find a checkpoint to start reading from so the log cannot be present.
		return false, nil
	} else if err != nil {
		return false, err
	}

	i, err := db.newIterator(entryIdx)
	if err != nil {
		return false, fmt.Errorf("failed to create iterator: %w", err)
	}
	db.log.Trace("Starting search", "entry", entryIdx, "blockNum", i.current.blockNum, "logIdx", i.current.logIdx)
	defer func() {
		db.m.RecordSearchEntriesRead(i.entriesRead)
	}()
	for {
		evtBlockNum, evtLogIdx, evtHash, err := i.NextLog()
		if errors.Is(err, io.EOF) {
			// Reached end of log without finding the event
			return false, nil
		} else if err != nil {
			return false, fmt.Errorf("failed to read next log: %w", err)
		}
		if evtBlockNum == blockNum && evtLogIdx == logIdx {
			db.log.Trace("Found initiatingEvent", "blockNum", evtBlockNum, "logIdx", evtLogIdx, "hash", evtHash)
			// Found the requested block and log index, check if the hash matches
			return evtHash == logHash, nil
		}
		if evtBlockNum > blockNum || (evtBlockNum == blockNum && evtLogIdx > logIdx) {
			// Progressed past the requested log without finding it.
			return false, nil
		}
	}
}

func (db *DB) newIterator(startCheckpointEntry int64) (*iterator, error) {
	// TODO(optimism#10857): Handle starting from a checkpoint after initiating-event but before its executing-link
	// Will need to read the entry prior to the checkpoint to get the initiating event info
	current, err := db.readSearchCheckpoint(startCheckpointEntry)
	if err != nil {
		return nil, fmt.Errorf("failed to read search checkpoint entry %v: %w", startCheckpointEntry, err)
	}
	i := &iterator{
		db: db,
		// +2 to skip the initial search checkpoint and the canonical hash event after it
		nextEntryIdx: startCheckpointEntry + 2,
		current: logContext{
			blockNum: current.blockNum,
			logIdx:   current.logIdx,
		},
	}
	return i, nil
}

// searchCheckpoint performs a binary search of the searchCheckpoint entries to find the closest one at or before
// the requested log.
// Returns the index of the searchCheckpoint to begin reading from or an error
func (db *DB) searchCheckpoint(blockNum uint64, logIdx uint32) (int64, error) {
	n := (db.lastEntryIdx() / searchCheckpointFrequency) + 1
	// Define x[-1] < target and x[n] >= target.
	// Invariant: x[i-1] < target, x[j] >= target.
	i, j := int64(0), n
	for i < j {
		h := int64(uint64(i+j) >> 1) // avoid overflow when computing h
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

func (db *DB) AddLog(logHash TruncatedHash, block eth.BlockID, timestamp uint64, logIdx uint32) error {
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
	if (db.lastEntryIdx()+1)%searchCheckpointFrequency == 0 {
		if err := db.writeSearchCheckpoint(block.Number, logIdx, timestamp, block.Hash); err != nil {
			return fmt.Errorf("failed to write search checkpoint: %w", err)
		}
		db.lastEntryContext = postState
	}

	if err := db.writeInitiatingEvent(postState, logHash); err != nil {
		return err
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

// writeSearchCheckpoint appends search checkpoint and canonical hash entry to the log
// type 0: "search checkpoint" <type><uint64 block number: 8 bytes><uint32 event index offset: 4 bytes><uint64 timestamp: 8 bytes> = 20 bytes
// type 1: "canonical hash" <type><parent blockhash truncated: 20 bytes> = 21 bytes
func (db *DB) writeSearchCheckpoint(blockNum uint64, logIdx uint32, timestamp uint64, blockHash common.Hash) error {
	entry := newSearchCheckpoint(blockNum, logIdx, timestamp).encode()
	if err := db.store.Append(entry); err != nil {
		return err
	}
	return db.writeCanonicalHash(blockHash)
}

func (db *DB) readSearchCheckpoint(entryIdx int64) (searchCheckpoint, error) {
	data, err := db.store.Read(entryIdx)
	if err != nil {
		return searchCheckpoint{}, fmt.Errorf("failed to read entry %v: %w", entryIdx, err)
	}
	return newSearchCheckpointFromEntry(data)
}

// writeCanonicalHash appends a canonical hash entry to the log
// type 1: "canonical hash" <type><parent blockhash truncated: 20 bytes> = 21 bytes
func (db *DB) writeCanonicalHash(blockHash common.Hash) error {
	return db.store.Append(newCanonicalHash(TruncateHash(blockHash)).encode())
}

func (db *DB) readCanonicalHash(entryIdx int64) (canonicalHash, error) {
	data, err := db.store.Read(entryIdx)
	if err != nil {
		return canonicalHash{}, fmt.Errorf("failed to read entry %v: %w", entryIdx, err)
	}
	if data[0] != typeCanonicalHash {
		return canonicalHash{}, fmt.Errorf("%w: expected canonical hash at entry %v but was type %v", ErrDataCorruption, entryIdx, data[0])
	}
	return newCanonicalHashFromEntry(data)
}

// writeInitiatingEvent appends an initiating event to the log
// type 2: "initiating event" <type><blocknum diff: 1 byte><event flags: 1 byte><event-hash: 20 bytes> = 23 bytes
func (db *DB) writeInitiatingEvent(postState logContext, logHash TruncatedHash) error {
	evt, err := newInitiatingEvent(db.lastEntryContext, postState.blockNum, postState.logIdx, logHash)
	if err != nil {
		return err
	}
	return db.store.Append(evt.encode())
}

func TruncateHash(hash common.Hash) TruncatedHash {
	var truncated TruncatedHash
	copy(truncated[:], hash[0:20])
	return truncated
}

func (db *DB) Close() error {
	return db.store.Close()
}
