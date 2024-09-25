package entrydb

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/ethereum/go-ethereum/log"
)

const (
	EntrySize = 34
)

type EntryIdx int64

type Entry [EntrySize]byte

func (entry Entry) Type() EntryType {
	return EntryType(entry[0])
}

type EntryTypeFlag uint8

const (
	FlagSearchCheckpoint EntryTypeFlag = 1 << TypeSearchCheckpoint
	FlagCanonicalHash    EntryTypeFlag = 1 << TypeCanonicalHash
	FlagInitiatingEvent  EntryTypeFlag = 1 << TypeInitiatingEvent
	FlagExecutingLink    EntryTypeFlag = 1 << TypeExecutingLink
	FlagExecutingCheck   EntryTypeFlag = 1 << TypeExecutingCheck
	FlagPadding          EntryTypeFlag = 1 << TypePadding
	// for additional padding
	FlagPadding2 EntryTypeFlag = FlagPadding << 1
)

func (ex EntryTypeFlag) Any(v EntryTypeFlag) bool {
	return ex&v != 0
}

func (ex *EntryTypeFlag) Add(v EntryTypeFlag) {
	*ex = *ex | v
}

func (ex *EntryTypeFlag) Remove(v EntryTypeFlag) {
	*ex = *ex &^ v
}

type EntryType uint8

const (
	TypeSearchCheckpoint EntryType = iota
	TypeCanonicalHash
	TypeInitiatingEvent
	TypeExecutingLink
	TypeExecutingCheck
	TypePadding
)

func (d EntryType) String() string {
	switch d {
	case TypeSearchCheckpoint:
		return "searchCheckpoint"
	case TypeCanonicalHash:
		return "canonicalHash"
	case TypeInitiatingEvent:
		return "initiatingEvent"
	case TypeExecutingLink:
		return "executingLink"
	case TypeExecutingCheck:
		return "executingCheck"
	case TypePadding:
		return "padding"
	default:
		return fmt.Sprintf("unknown-%d", uint8(d))
	}
}

// dataAccess defines a minimal API required to manipulate the actual stored data.
// It is a subset of the os.File API but could (theoretically) be satisfied by an in-memory implementation for testing.
type dataAccess interface {
	io.ReaderAt
	io.Writer
	io.Closer
	Truncate(size int64) error
}

type EntryDB struct {
	data         dataAccess
	lastEntryIdx EntryIdx

	cleanupFailedWrite bool
}

// NewEntryDB creates an EntryDB. A new file will be created if the specified path does not exist,
// but parent directories will not be created.
// If the file exists it will be used as the existing data.
// Returns ErrRecoveryRequired if the existing file is not a valid entry db. A EntryDB is still returned but all
// operations will return ErrRecoveryRequired until the Recover method is called.
func NewEntryDB(logger log.Logger, path string) (*EntryDB, error) {
	logger.Info("Opening entry database", "path", path)
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open database at %v: %w", path, err)
	}
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat database at %v: %w", path, err)
	}
	size := info.Size() / EntrySize
	db := &EntryDB{
		data:         file,
		lastEntryIdx: EntryIdx(size - 1),
	}
	if size*EntrySize != info.Size() {
		logger.Warn("File size is not a multiple of entry size. Truncating to last complete entry", "fileSize", size, "entrySize", EntrySize)
		if err := db.recover(); err != nil {
			return nil, fmt.Errorf("failed to recover database at %v: %w", path, err)
		}
	}
	return db, nil
}

func (e *EntryDB) Size() int64 {
	return int64(e.lastEntryIdx) + 1
}

func (e *EntryDB) LastEntryIdx() EntryIdx {
	return e.lastEntryIdx
}

// Read an entry from the database by index. Returns io.EOF iff idx is after the last entry.
func (e *EntryDB) Read(idx EntryIdx) (Entry, error) {
	if idx > e.lastEntryIdx {
		return Entry{}, io.EOF
	}
	var out Entry
	read, err := e.data.ReadAt(out[:], int64(idx)*EntrySize)
	// Ignore io.EOF if we read the entire last entry as ReadAt may return io.EOF or nil when it reads the last byte
	if err != nil && !(errors.Is(err, io.EOF) && read == EntrySize) {
		return Entry{}, fmt.Errorf("failed to read entry %v: %w", idx, err)
	}
	return out, nil
}

// Append entries to the database.
// The entries are combined in memory and passed to a single Write invocation.
// If the write fails, it will attempt to truncate any partially written data.
// Subsequent writes to this instance will fail until partially written data is truncated.
func (e *EntryDB) Append(entries ...Entry) error {
	if e.cleanupFailedWrite {
		// Try to rollback partially written data from a previous Append
		if truncateErr := e.Truncate(e.lastEntryIdx); truncateErr != nil {
			return fmt.Errorf("failed to recover from previous write error: %w", truncateErr)
		}
	}
	data := make([]byte, 0, len(entries)*EntrySize)
	for _, entry := range entries {
		data = append(data, entry[:]...)
	}
	if n, err := e.data.Write(data); err != nil {
		if n == 0 {
			// Didn't write any data, so no recovery required
			return err
		}
		// Try to rollback the partially written data
		if truncateErr := e.Truncate(e.lastEntryIdx); truncateErr != nil {
			// Failed to rollback, set a flag to attempt the clean up on the next write
			e.cleanupFailedWrite = true
			return errors.Join(err, fmt.Errorf("failed to remove partially written data: %w", truncateErr))
		}
		// Successfully rolled back the changes, still report the failed write
		return err
	}
	e.lastEntryIdx += EntryIdx(len(entries))
	return nil
}

// Truncate the database so that the last retained entry is idx. Any entries after idx are deleted.
func (e *EntryDB) Truncate(idx EntryIdx) error {
	if err := e.data.Truncate((int64(idx) + 1) * EntrySize); err != nil {
		return fmt.Errorf("failed to truncate to entry %v: %w", idx, err)
	}
	// Update the lastEntryIdx cache
	e.lastEntryIdx = idx
	e.cleanupFailedWrite = false
	return nil
}

// recover an invalid database by truncating back to the last complete event.
func (e *EntryDB) recover() error {
	if err := e.data.Truncate((e.Size()) * EntrySize); err != nil {
		return fmt.Errorf("failed to truncate trailing partial entries: %w", err)
	}
	return nil
}

func (e *EntryDB) Close() error {
	return e.data.Close()
}
