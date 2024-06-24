package entrydb

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/ethereum/go-ethereum/log"
)

const (
	EntrySize = 24
)

var (
	ErrRecoveryRequired = errors.New("recovery required")
)

type Entry [EntrySize]byte

// dataAccess defines a minimal API required to manipulate the actual stored data.
// It is a subset of the os.File API but could (theoretically) be satisfied by an in-memory implementation for testing.
type dataAccess interface {
	io.ReaderAt
	io.Writer
	io.Closer
	Truncate(size int64) error
}

type EntryDB struct {
	data             dataAccess
	size             int64
	recoveryRequired bool
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
		data: file,
		size: size,
	}
	if size*EntrySize != info.Size() {
		db.recoveryRequired = true
		logger.Warn("File size (%v) is nut a multiple of entry size %v. Truncating to last complete entry", size, EntrySize)
		if err := db.Recover(); err != nil {
			return nil, fmt.Errorf("failed to recover database at %v: %w", path, err)
		}
	}
	return db, nil
}

func (e *EntryDB) RecoveryRequired() bool {
	return e.recoveryRequired
}

func (e *EntryDB) Size() int64 {
	return e.size
}

// Read an entry from the database by index. Returns io.EOF iff idx is after the last entry.
func (e *EntryDB) Read(idx int64) (Entry, error) {
	var out Entry
	if e.recoveryRequired {
		return out, ErrRecoveryRequired
	}
	read, err := e.data.ReadAt(out[:], idx*EntrySize)
	// Ignore io.EOF if we read the entire last entry as ReadAt may return io.EOF or nil when it reads the last byte
	if err != nil && !(errors.Is(err, io.EOF) && read == EntrySize) {
		return Entry{}, fmt.Errorf("failed to read entry %v: %w", idx, err)
	}
	return out, nil
}

// Append an entry to the database.
func (e *EntryDB) Append(entry Entry) error {
	if e.recoveryRequired {
		return ErrRecoveryRequired
	}
	if _, err := e.data.Write(entry[:]); err != nil {
		// TODO(optimism#10857): When a write fails, need to revert any in memory changes and truncate back to the
		// pre-write state. Likely need to batch writes for multiple entries into a single write akin to transactions
		// to avoid leaving hanging entries without the entry that should follow them.
		return err
	}
	e.size++
	return nil
}

// Truncate the database so that the last retained entry is idx. Any entries after idx are deleted.
func (e *EntryDB) Truncate(idx int64) error {
	if e.recoveryRequired {
		return ErrRecoveryRequired
	}
	if err := e.data.Truncate((idx + 1) * EntrySize); err != nil {
		return fmt.Errorf("failed to truncate to entry %v: %w", idx, err)
	}
	// Update the lastEntryIdx cache
	e.size = idx + 1
	return nil
}

// Recover an invalid database by truncating back to the last complete event.
func (e *EntryDB) Recover() error {
	if !e.recoveryRequired {
		return nil
	}
	if err := e.data.Truncate((e.size) * EntrySize); err != nil {
		return fmt.Errorf("failed to truncate trailing partial entries: %w", err)
	}
	e.recoveryRequired = false
	return nil
}

func (e *EntryDB) Close() error {
	return e.data.Close()
}
