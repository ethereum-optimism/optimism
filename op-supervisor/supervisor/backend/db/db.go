package db

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrBlockOutOfOrder = errors.New("block out of order")
)

type dataAccess interface {
	io.ReaderAt
	io.Writer
	io.Closer
	Truncate(size int64) error
}

type DB struct {
	data   dataAccess
	rwLock sync.RWMutex

	lastEntryIdx int64
	lastBlockNum uint64
}

func NewFromFile(path string) (*DB, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o666)
	if err != nil {
		return nil, fmt.Errorf("failed to open database at %v: %w", path, err)
	}
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat database at %v: %w", path, err)
	}
	lastEntryIdx := info.Size()/common.HashLength - 1
	db := &DB{
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
	entry, err := db.readEntry(db.lastEntryIdx)
	if err != nil {
		return fmt.Errorf("failed to read last entry: %w", err)
	}
	db.lastBlockNum = db.blockNum(entry)
	return nil
}

// Contains return true iff the specified logHash is recorded in the specified blockNum and logIdx.
// logIdx is the index of the log in the array of all logs the block.
func (db *DB) Contains(blockNum uint64, logIdx uint32, logHash common.Hash) (bool, error) {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()
	_, entry, found, err := db.search(blockNum, logIdx)
	if err != nil {
		return false, err
	}
	if !found {
		return false, nil
	}
	searchFor := db.combine(blockNum, logIdx, logHash)
	return entry == searchFor, nil
}

// search performs a binary search to find the entry at blockNum, logIdx.
// Returns the index that the entry would be found at if present, the matching entry (or common.Hash{})
// and a bool indicating whether an entry at the specified blockNum and logIdx was found.
// An error is returned if an error occurs reading from the data store.
func (db *DB) search(blockNum uint64, logIdx uint32) (int64, common.Hash, bool, error) {
	n := db.lastEntryIdx + 1
	// Define x[-1] < target and x[n] >= target.
	// Invariant: x[i-1] < target, x[j] >= target.
	i, j := int64(0), n
	for i < j {
		h := int64(uint64(i+j) >> 1) // avoid overflow when computing h
		entry, err := db.readEntry(h)
		if err != nil {
			return 0, common.Hash{}, false, fmt.Errorf("failed to read entry %v: %w", h, err)
		}
		entryBlock := db.blockNum(entry)
		// i ≤ h < j
		if entryBlock < blockNum || (entryBlock == blockNum && db.logIdx(entry) < logIdx) {
			i = h + 1 // preserves x[i-1] < target
		} else {
			j = h // preserves x[j] >= target
		}
	}
	if i < n {
		entry, err := db.readEntry(i)
		if err != nil {
			return 0, common.Hash{}, false, fmt.Errorf("failed to read entry %v: %w", i, err)
		}
		if db.blockNum(entry) == blockNum && db.logIdx(entry) == logIdx {
			// Found entry at requested block number and log index
			return i, entry, true, nil
		}
	}
	// Not found, only return where it would be inserted
	return i, common.Hash{}, false, nil
}

// Add a block to the database with the specified logHashes. The logs are recorded in the order they are specified
// and must be the full set of logs for the block.
func (db *DB) Add(blockNum uint64, logHashes []common.Hash) error {
	db.rwLock.Lock()
	defer db.rwLock.Unlock()
	if db.lastBlockNum >= blockNum {
		return fmt.Errorf("%w: adding %v, head: %v", ErrBlockOutOfOrder, blockNum, db.lastBlockNum)
	}
	for logIdx, logHash := range logHashes {
		entry := db.combine(blockNum, uint32(logIdx), logHash)
		if _, err := db.data.Write(entry[:]); err != nil {
			return fmt.Errorf("failed to write logs for block %v: %w", blockNum, err)
		}
	}
	db.lastBlockNum = blockNum
	db.lastEntryIdx += int64(len(logHashes))
	return nil
}

// Rewind the database to remove any blocks after headBlockNum
// The block at headBlockNum itself is not removed.
func (db *DB) Rewind(headBlockNum uint64) error {
	db.rwLock.Lock()
	defer db.rwLock.Unlock()
	if headBlockNum > db.lastBlockNum {
		// Nothing to do
		return nil
	}
	// Find the first index we should delete
	idx, _, _, err := db.search(headBlockNum+1, 0)
	if err != nil {
		return fmt.Errorf("failed to find entry index for block %v: %w", headBlockNum, err)
	}
	// Truncate to contain exactly idx entries, since indices are 0 based, this deletes idx and everything after it
	err = db.data.Truncate(idx * common.HashLength)
	if err != nil {
		return fmt.Errorf("failed to truncate to block %v: %w", headBlockNum, err)
	}
	// The first remaining entry is one before the first deleted entry
	db.lastEntryIdx = idx - 1
	db.lastBlockNum = headBlockNum
	return nil
}

func (db *DB) readEntry(idx int64) (common.Hash, error) {
	var out common.Hash
	read, err := db.data.ReadAt(out[:], idx*common.HashLength)
	// Ignore io.EOF if we read the entire last entry as ReadAt may return io.EOF or nil when it reads the last byte
	if err != nil && !(errors.Is(err, io.EOF) && read == common.HashLength) {
		return common.Hash{}, fmt.Errorf("failed to read entry %v: %w", idx, err)
	}
	return out, nil
}

func (db *DB) combine(blockNum uint64, logIdx uint32, hash common.Hash) common.Hash {
	var result common.Hash
	binary.LittleEndian.PutUint64(result[0:8], blockNum)
	binary.LittleEndian.PutUint32(result[8:12], logIdx)
	copy(result[12:], hash[12:])
	return result
}

func (db *DB) blockNum(entry common.Hash) uint64 {
	return binary.LittleEndian.Uint64(entry[:8])
}

func (db *DB) logIdx(entry common.Hash) uint32 {
	return binary.LittleEndian.Uint32(entry[8:12])
}

func (db *DB) Close() error {
	return db.data.Close()
}
