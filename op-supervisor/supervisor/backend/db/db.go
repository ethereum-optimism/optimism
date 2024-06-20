package db

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"slices"
	"sync"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

const (
	entrySize                 = 24
	searchCheckpointFrequency = 256
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

type truncatedHash []byte

// dataAccess defines a minimal API required to manipulate the actual stored data.
// It is a subset of the os.File API but could (theoretically) be satisfied by an in-memory implementation for testing.
type dataAccess interface {
	io.ReaderAt
	io.Writer
	io.Closer
	Truncate(size int64) error
}

type Metrics interface {
	RecordEntryCount(count int64)
	RecordSearchEntriesRead(count int64)
}

type checkpointData struct {
	blockNum  uint64
	logIdx    uint32
	timestamp uint64
}

type state struct {
	blockNum  uint64
	blockHash truncatedHash
	timestamp uint64
	logIdx    uint32
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
// type 2: "initiating event" <type><blocknum diff: 1 byte><log idx diff: 1 byte><event-hash: 20 bytes> = 23 bytes
// type 3: "executing link" <type><chain: 4 bytes><blocknum: 8 bytes><event index: 3 bytes><uint64 timestamp: 8 bytes> = 24 bytes
// type 4: "executing check" <type><event-hash: 20 bytes> = 21 bytes
// other types: future compat. E.g. for linking to L1, registering block-headers as a kind of initiating-event, tracking safe-head progression, etc.
//
// Right-pad each entry that is not 24 bytes.
//
// event-hash: H(origin, timestamp, payloadhash); enough to check identifier matches & payload matches.
type DB struct {
	log    log.Logger
	m      Metrics
	data   dataAccess
	rwLock sync.RWMutex

	lastEntryIdx   int64
	lastEntryState state
}

func NewFromFile(logger log.Logger, m Metrics, path string) (*DB, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o666)
	if err != nil {
		return nil, fmt.Errorf("failed to open database at %v: %w", path, err)
	}
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat database at %v: %w", path, err)
	}
	lastEntryIdx := info.Size()/entrySize - 1
	db := &DB{
		log:          logger,
		m:            m,
		data:         file,
		lastEntryIdx: lastEntryIdx,
	}
	if err := db.init(); err != nil {
		return nil, fmt.Errorf("failed to init database: %w", err)
	}
	return db, nil
}

func (db *DB) init() error {
	if db.lastEntryIdx < 0 {
		// Database is empty so nothing to init
		return nil
	}
	lastCheckpoint := (db.lastEntryIdx / searchCheckpointFrequency) * searchCheckpointFrequency
	checkpoint, err := db.readSearchCheckpoint(lastCheckpoint)
	if err != nil {
		return fmt.Errorf("failed to read last search checkpoint: %w", err)
	}
	//blockHash, err := db.readCanonicalHash(lastCheckpoint + 1)
	//if err != nil {
	//	return fmt.Errorf("failed to read last canonical hash: %w", err)
	//}
	// TODO: This is broken - we need to consider any log events after the previous checkpoint which may increment blocks
	db.lastEntryState = state{
		blockNum: checkpoint.blockNum,
		//blockHash: blockHash, // TODO: This should be set - need a test for it.
		timestamp: checkpoint.timestamp,
		logIdx:    checkpoint.logIdx,
	}
	db.updateEntryCountMetric()
	return nil
}

func (db *DB) updateEntryCountMetric() {
	db.m.RecordEntryCount(db.lastEntryIdx + 1)
}

// Contains return true iff the specified logHash is recorded in the specified blockNum and logIdx.
// logIdx is the index of the log in the array of all logs the block.
func (db *DB) Contains(blockNum uint64, logIdx uint32, logHash common.Hash) (bool, error) {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()
	db.log.Trace("Checking for log", "blockNum", blockNum, "logIdx", logIdx, "hash", truncateHash(logHash))
	entryIdx, err := db.searchCheckpoint(blockNum, logIdx)
	if errors.Is(err, io.EOF) {
		// Did not find a checkpoint to start reading from so the log cannot be present.
		return false, nil
	} else if err != nil {
		return false, err
	}

	current, err := db.readSearchCheckpoint(entryIdx)
	if err != nil {
		return false, fmt.Errorf("failed to read search checkpoint entry %v: %w", entryIdx, err)
	}
	db.log.Trace("Starting search", "entry", entryIdx, "blockNum", current.blockNum, "logIdx", current.logIdx)
	entriesRead := int64(0)
	defer func() {
		db.m.RecordSearchEntriesRead(entriesRead)
	}()
	for i := entryIdx + 2; i <= db.lastEntryIdx; i++ {
		entry, err := db.readEntry(i)
		if err != nil {
			return false, fmt.Errorf("failed to read entry %v: %w", i, err)
		}
		entriesRead++
		switch entry[0] {
		case typeSearchCheckpoint:
			current = db.parseSearchCheckpoint(entry)
		case typeCanonicalHash:
			// Skip
		case typeInitiatingEvent:
			evtBlockNum, evtLogIdx, evtHash := db.parseInitiatingEvent(current, entry)
			if evtBlockNum == blockNum && evtLogIdx == logIdx {
				db.log.Trace("Found initiatingEvent", "blockNum", evtBlockNum, "logIdx", evtLogIdx, "hash", evtHash)
				// Found the requested block and log index, check if the hash matches
				return slices.Equal(evtHash, truncateHash(logHash)), nil
			}
			if evtBlockNum > blockNum || (evtBlockNum == blockNum && evtLogIdx > logIdx) {
				// Progressed past the requested log without finding it.
				return false, nil
			}
			current.blockNum = evtBlockNum
			current.logIdx = evtLogIdx
		case typeExecutingCheck:
		// TODO: Handle this properly
		case typeExecutingLink:
		// TODO: Handle this properly
		default:
			return false, fmt.Errorf("unknown entry type %v", entry[0])
		}
	}

	return false, nil
}

// searchCheckpoint performs a binary search of the searchCheckpoint entries to find the closest one at or before
// the requested log.
// Returns the index of the searchCheckpoint to begin reading from or an error
func (db *DB) searchCheckpoint(blockNum uint64, logIdx uint32) (int64, error) {
	n := (db.lastEntryIdx / searchCheckpointFrequency) + 1
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

func (db *DB) AddLog(log common.Hash, block eth.BlockID, timestamp uint64, logIdx uint32) error {
	db.rwLock.Lock()
	defer db.rwLock.Unlock()
	postState := state{
		blockNum:  block.Number,
		blockHash: truncateHash(block.Hash),
		timestamp: timestamp,
		logIdx:    logIdx,
	}
	if block.Number == 0 {
		return fmt.Errorf("%w: should not have logs in block 0", ErrLogOutOfOrder)
	}
	if db.lastEntryState.blockNum > block.Number {
		return fmt.Errorf("%w: adding block %v, head block: %v", ErrLogOutOfOrder, block.Number, db.lastEntryState.blockNum)
	}
	if db.lastEntryState.blockNum == block.Number && db.lastEntryState.logIdx >= logIdx {
		return fmt.Errorf("%w: adding log %v in block %v, but already at log %v", ErrLogOutOfOrder, logIdx, block.Number, db.lastEntryState.logIdx)
	}
	if (db.lastEntryIdx+1)%searchCheckpointFrequency == 0 {
		if err := db.writeSearchCheckpoint(postState); err != nil {
			return fmt.Errorf("failed to write search checkpoint: %w", err)
		}
		db.lastEntryState = postState
	}

	if err := db.writeInitiatingEvent(postState, log); err != nil {
		return err
	}
	db.lastEntryState = postState
	db.updateEntryCountMetric()
	return nil
}

// writeSearchCheckpoint appends search checkpoint and canonical hash entry to the log
// type 0: "search checkpoint" <type><uint64 block number: 8 bytes><uint32 event index offset: 4 bytes><uint64 timestamp: 8 bytes> = 20 bytes
// type 1: "canonical hash" <type><parent blockhash truncated: 20 bytes> = 21 bytes
func (db *DB) writeSearchCheckpoint(currentState state) error {
	var entry [entrySize]byte
	entry[0] = typeSearchCheckpoint
	binary.LittleEndian.PutUint64(entry[1:9], currentState.blockNum)
	binary.LittleEndian.PutUint32(entry[9:13], currentState.logIdx)
	binary.LittleEndian.PutUint64(entry[13:21], currentState.timestamp)
	if err := db.writeEntry(entry); err != nil {
		return err
	}
	return db.writeCanonicalHash(currentState)
}

func (db *DB) readSearchCheckpoint(entryIdx int64) (checkpointData, error) {
	data, err := db.readEntry(entryIdx)
	if err != nil {
		return checkpointData{}, fmt.Errorf("failed to read entry %v: %w", entryIdx, err)
	}
	if data[0] != typeSearchCheckpoint {
		return checkpointData{}, fmt.Errorf("%w: expected search checkpoint at entry %v but was type %v", ErrDataCorruption, entryIdx, data[0])
	}
	return db.parseSearchCheckpoint(data), nil
}

func (db *DB) parseSearchCheckpoint(data [entrySize]byte) checkpointData {
	return checkpointData{
		blockNum:  binary.LittleEndian.Uint64(data[1:9]),
		logIdx:    binary.LittleEndian.Uint32(data[9:13]),
		timestamp: binary.LittleEndian.Uint64(data[13:21]),
	}
}

// writeCanonicalHash appends a canonical hash entry to the log
// type 1: "canonical hash" <type><parent blockhash truncated: 20 bytes> = 21 bytes
func (db *DB) writeCanonicalHash(currentState state) error {
	var entry [entrySize]byte
	entry[0] = typeCanonicalHash
	copy(entry[1:21], currentState.blockHash[:])
	return db.writeEntry(entry)
}

func (db *DB) readCanonicalHash(entryIdx int64) (truncatedHash, error) {
	data, err := db.readEntry(entryIdx)
	if err != nil {
		return truncatedHash{}, fmt.Errorf("failed to read entry %v: %w", entryIdx, err)
	}
	if data[0] != typeCanonicalHash {
		return truncatedHash{}, fmt.Errorf("%w: expected canonical hash at entry %v but was type %v", ErrDataCorruption, entryIdx, data[0])
	}
	return db.parseCanonicalHash(data), nil
}

func (db *DB) parseCanonicalHash(data [24]byte) truncatedHash {
	return data[1:21]
}

// writeInitiatingEvent appends an initiating event to the log
// type 2: "initiating event" <type><blocknum diff: 1 byte><event index diff: 1 byte><event-hash: 20 bytes> = 23 bytes
func (db *DB) writeInitiatingEvent(postState state, log common.Hash) error {
	var entry [entrySize]byte
	entry[0] = typeInitiatingEvent
	blockDiff := postState.blockNum - db.lastEntryState.blockNum
	if blockDiff > math.MaxUint8 {
		// TODO: Need to find a way to support this.
		return fmt.Errorf("too many block skipped between %v and %v", db.lastEntryState.blockNum, postState.blockNum)
	}
	entry[1] = byte(blockDiff)
	// TODO: Probably shouldn't allow skipping logs as that indicates we missed data
	currLogIdx := db.lastEntryState.logIdx
	if blockDiff > 0 {
		currLogIdx = 0
	}
	logDiff := postState.logIdx - currLogIdx
	if logDiff > math.MaxUint8 {
		return fmt.Errorf("too many logs skipped between %v and %v", db.lastEntryState.logIdx, postState.logIdx)
	}
	entry[2] = byte(logDiff)
	truncated := truncateHash(log)
	copy(entry[3:23], truncated[:])
	return db.writeEntry(entry)
}

func (db *DB) parseInitiatingEvent(checkpoint checkpointData, entry [entrySize]byte) (uint64, uint32, []byte) {
	blockNumDiff := entry[1]
	logIdxDiff := entry[2]
	blockNum := checkpoint.blockNum + uint64(blockNumDiff)
	logIdx := checkpoint.logIdx
	if blockNumDiff > 0 {
		logIdx = 0
	}
	logIdx = logIdx + uint32(logIdxDiff)
	eventHash := entry[3:23]
	return blockNum, logIdx, eventHash
}

func (db *DB) writeEntry(entry [entrySize]byte) error {
	// TODO: Automatically insert search checkpoint
	if _, err := db.data.Write(entry[:]); err != nil {
		// TODO: Truncate to the start of the entry?
		// TODO: Roll back any updates to memory state
		return err
	}
	db.lastEntryIdx++
	return nil
}

// Rewind the database to remove any blocks after headBlockNum
// The block at headBlockNum itself is not removed.
//func (db *DB) Rewind(headBlockNum uint64) error {
//	db.rwLock.Lock()
//	defer db.rwLock.Unlock()
//	if headBlockNum > db.lastBlockNum {
//		// Nothing to do
//		return nil
//	}
//	// Find the first index we should delete
//	idx, _, _, err := db.search(headBlockNum+1, 0)
//	if err != nil {
//		return fmt.Errorf("failed to find entry index for block %v: %w", headBlockNum, err)
//	}
//	// Truncate to contain exactly idx entries, since indices are 0 based, this deletes idx and everything after it
//	err = db.data.Truncate(idx * common.HashLength)
//	if err != nil {
//		return fmt.Errorf("failed to truncate to block %v: %w", headBlockNum, err)
//	}
//	// The first remaining entry is one before the first deleted entry
//	db.lastEntryIdx = idx - 1
//	db.lastBlockNum = headBlockNum
//	return nil
//}

func (db *DB) readEntry(idx int64) ([entrySize]byte, error) {
	var out [entrySize]byte
	read, err := db.data.ReadAt(out[:], idx*entrySize)
	// Ignore io.EOF if we read the entire last entry as ReadAt may return io.EOF or nil when it reads the last byte
	if err != nil && !(errors.Is(err, io.EOF) && read == entrySize) {
		return [entrySize]byte{}, fmt.Errorf("failed to read entry %v: %w", idx, err)
	}
	return out, nil
}

func truncateHash(hash common.Hash) truncatedHash {
	return hash[0:20]
}

func (db *DB) Close() error {
	return db.data.Close()
}
