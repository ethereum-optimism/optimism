package entrydb

import (
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	EntrySize = 24
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
	data         dataAccess
	lastEntryIdx int64
}

func NewEntryDB(path string) (*EntryDB, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o666)
	if err != nil {
		return nil, fmt.Errorf("failed to open database at %v: %w", path, err)
	}
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat database at %v: %w", path, err)
	}
	lastEntryIdx := info.Size()/EntrySize - 1
	return &EntryDB{
		data:         file,
		lastEntryIdx: lastEntryIdx,
	}, nil
}

func (e *EntryDB) Size() int64 {
	return e.lastEntryIdx + 1
}

func (e *EntryDB) Read(idx int64) (Entry, error) {
	var out Entry
	read, err := e.data.ReadAt(out[:], idx*EntrySize)
	// Ignore io.EOF if we read the entire last entry as ReadAt may return io.EOF or nil when it reads the last byte
	if err != nil && !(errors.Is(err, io.EOF) && read == EntrySize) {
		return Entry{}, fmt.Errorf("failed to read entry %v: %w", idx, err)
	}
	return out, nil
}

func (e *EntryDB) Append(entries ...Entry) error {
	for _, entry := range entries {
		if _, err := e.data.Write(entry[:]); err != nil {
			// TODO(optimism#10857): When a write fails, need to revert any in memory changes and truncate back to the
			// pre-write state. Likely need to batch writes for multiple entries into a single write akin to transactions
			// to avoid leaving hanging entries without the entry that should follow them.
			return err
		}
		e.lastEntryIdx++
	}
	return nil
}

func (e *EntryDB) Truncate(idx int64) error {
	if err := e.data.Truncate((idx + 1) * EntrySize); err != nil {
		return fmt.Errorf("failed to truncate to entry %v: %w", idx, err)
	}
	// Update the lastEntryIdx cache and then use db.init() to find the log context for the new latest log entry
	e.lastEntryIdx = idx
	return nil
}

func (e *EntryDB) Close() error {
	return e.data.Close()
}
