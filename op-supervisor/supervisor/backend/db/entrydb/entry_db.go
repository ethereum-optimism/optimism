package entrydb

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/ethereum/go-ethereum/log"
)

type EntryStore[T EntryType, E Entry[T]] interface {
	Size() int64
	LastEntryIdx() EntryIdx
	Read(idx EntryIdx) (E, error)
	Append(entries ...E) error
	Truncate(idx EntryIdx) error
	// Sync forces any buffered data and metadata to disk. On Linux this calls
	// fsync(2). On macOS it does NOT issue F_FULLFSYNC, so the device write
	// cache is not flushed; developers running locally on macOS get weaker
	// durability than production Linux.
	Sync() error
	// PrepareForMutation drains any outstanding deferred cleanup so that the
	// store is in a known-good state before the caller mutates its own
	// in-memory invariants. If the drain itself fails, the cleanup remains
	// outstanding and the returned error must be propagated; the caller MUST
	// NOT proceed with the mutation.
	PrepareForMutation() error
	Close() error
}

type EntryIdx int64

type EntryType interface {
	String() string
	~uint8
}

type Entry[T EntryType] interface {
	Type() T
	comparable
}

// Binary is the binary interface to encode/decode/size entries.
// This should be a zero-cost abstraction, and is bundled as interface for the EntryDB
// to have generic access to this functionality without const-generics for array size in Go.
type Binary[T EntryType, E Entry[T]] interface {
	Append(dest []byte, e *E) []byte
	ReadAt(dest *E, r io.ReaderAt, at int64) (n int, err error)
	EntrySize() int
}

// DataAccess defines a minimal API required to manipulate the actual stored
// data. It is a subset of the os.File API but could (theoretically) be
// satisfied by an in-memory implementation for testing.
type DataAccess interface {
	io.ReaderAt
	io.Writer
	io.Closer
	Truncate(size int64) error
	// Sync forces any buffered data and metadata to disk. Mirrors *os.File.Sync.
	Sync() error
}

// pendingCleanup tracks an outstanding logical upper bound enforced because a
// prior Truncate (either a rollback after a partial Write or an explicit
// caller-issued Truncate) failed and left the on-disk file longer than the
// logical end of the database. While active, reads must be bounded by
// truncateTo and the next mutation drains by retrying the Truncate.
type pendingCleanup struct {
	active     bool
	truncateTo EntryIdx
}

type EntryDB[T EntryType, E Entry[T], B Binary[T, E]] struct {
	data         DataAccess
	lastEntryIdx EntryIdx

	b B

	pendingCleanup pendingCleanup
}

// NewEntryDB creates an EntryDB. A new file will be created if the specified path does not exist,
// but parent directories will not be created.
// If the file exists it will be used as the existing data.
// Returns ErrRecoveryRequired if the existing file is not a valid entry db. A EntryDB is still returned but all
// operations will return ErrRecoveryRequired until the Recover method is called.
func NewEntryDB[T EntryType, E Entry[T], B Binary[T, E]](logger log.Logger, path string) (*EntryDB[T, E, B], error) {
	logger.Info("Opening entry database", "path", path)
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open database at %v: %w", path, err)
	}
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat database at %v: %w", path, err)
	}
	var b B
	size := info.Size() / int64(b.EntrySize())
	db := &EntryDB[T, E, B]{
		data:         file,
		lastEntryIdx: EntryIdx(size - 1),
	}
	if size*int64(b.EntrySize()) != info.Size() {
		logger.Warn("File size is not a multiple of entry size. Truncating to last complete entry", "fileSize", size, "entrySize", b.EntrySize())
		if err := db.recover(); err != nil {
			return nil, fmt.Errorf("failed to recover database at %v: %w", path, err)
		}
	}
	return db, nil
}

// NewEntryDBFromDataAccess constructs an EntryDB around an existing
// DataAccess implementation. Intended for tests that wrap *os.File with
// fault-injection shims. lastEntryIdx must already reflect the current size
// of the underlying data, in entries.
func NewEntryDBFromDataAccess[T EntryType, E Entry[T], B Binary[T, E]](data DataAccess, lastEntryIdx EntryIdx) *EntryDB[T, E, B] {
	return &EntryDB[T, E, B]{
		data:         data,
		lastEntryIdx: lastEntryIdx,
	}
}

// effectiveLastEntryIdx returns the logical upper bound for reads. When
// pendingCleanup is active, the on-disk file is longer than the logical end of
// the database; readers must observe the smaller of the two.
func (e *EntryDB[T, E, B]) effectiveLastEntryIdx() EntryIdx {
	if e.pendingCleanup.active && e.pendingCleanup.truncateTo < e.lastEntryIdx {
		return e.pendingCleanup.truncateTo
	}
	return e.lastEntryIdx
}

func (e *EntryDB[T, E, B]) Size() int64 {
	return int64(e.effectiveLastEntryIdx()) + 1
}

// LastEntryIdx returns the index of the last entry in the DB.
// This returns -1 if the DB is empty.
func (e *EntryDB[T, E, B]) LastEntryIdx() EntryIdx {
	return e.effectiveLastEntryIdx()
}

// Read an entry from the database by index. Returns io.EOF iff idx is after the last entry.
func (e *EntryDB[T, E, B]) Read(idx EntryIdx) (E, error) {
	var out E
	if idx > e.effectiveLastEntryIdx() {
		return out, io.EOF
	}
	read, err := e.b.ReadAt(&out, e.data, int64(idx)*int64(e.b.EntrySize()))
	// Ignore io.EOF if we read the entire last entry as ReadAt may return io.EOF or nil when it reads the last byte
	if err != nil && !(errors.Is(err, io.EOF) && read == e.b.EntrySize()) {
		return out, fmt.Errorf("failed to read entry %v: %w", idx, err)
	}
	return out, nil
}

// drainCleanup retries the pending Truncate. On success, pendingCleanup is
// cleared and lastEntryIdx is updated.
func (e *EntryDB[T, E, B]) drainCleanup() error {
	if !e.pendingCleanup.active {
		return nil
	}
	target := e.pendingCleanup.truncateTo
	if err := e.data.Truncate((int64(target) + 1) * int64(e.b.EntrySize())); err != nil {
		return fmt.Errorf("failed to drain pending cleanup truncate to %v: %w", target, err)
	}
	e.lastEntryIdx = target
	e.pendingCleanup = pendingCleanup{}
	return nil
}

// PrepareForMutation drains any deferred cleanup. Callers invoke this before
// mutating in-memory state they want kept consistent with disk: if the drain
// fails the store stays in its current (cleanup-active) state, the caller
// receives the error, and no in-memory state has been advanced.
func (e *EntryDB[T, E, B]) PrepareForMutation() error {
	return e.drainCleanup()
}

// Append entries to the database.
// The entries are combined in memory and passed to a single Write invocation.
// If the write fails, it will attempt to truncate any partially written data.
// If pending cleanup from a prior failure exists, it is drained first; on
// drain failure the same typed error is returned without touching state.
func (e *EntryDB[T, E, B]) Append(entries ...E) error {
	if e.pendingCleanup.active {
		if err := e.drainCleanup(); err != nil {
			return err
		}
	}
	data := make([]byte, 0, len(entries)*e.b.EntrySize())
	for i := range entries {
		data = e.b.Append(data, &entries[i])
	}
	if n, err := e.data.Write(data); err != nil {
		if n == 0 {
			// Didn't write any data, so no recovery required
			return err
		}
		// Stage a rollback of the partial write and let drainCleanup execute it.
		// On drain failure, pendingCleanup stays active and the next mutation retries.
		e.pendingCleanup = pendingCleanup{active: true, truncateTo: e.lastEntryIdx}
		if drainErr := e.drainCleanup(); drainErr != nil {
			return errors.Join(err, fmt.Errorf("failed to remove partially written data: %w", drainErr))
		}
		return err
	}
	e.lastEntryIdx += EntryIdx(len(entries))
	return nil
}

// Truncate the database so that the last retained entry is idx. Any entries after idx are deleted.
// On failure, pendingCleanup is left active so that the next mutation will retry the truncate while
// reads continue to observe the smaller logical bound.
func (e *EntryDB[T, E, B]) Truncate(idx EntryIdx) error {
	// If pendingCleanup is active, its truncateTo is the strictest known target;
	// Truncate can only shrink further, never relax it. Take the stricter of the
	// two, install it as the pending target, and let drainCleanup do the work
	// (file truncate + lastEntryIdx update + clear-on-success).
	if e.pendingCleanup.active && idx > e.pendingCleanup.truncateTo {
		idx = e.pendingCleanup.truncateTo
	}
	e.pendingCleanup = pendingCleanup{active: true, truncateTo: idx}
	return e.drainCleanup()
}

// Sync forces any buffered data and metadata to disk. On Linux this is fsync(2).
// On macOS this does NOT issue F_FULLFSYNC; the device write cache is not flushed.
func (e *EntryDB[T, E, B]) Sync() error {
	return e.data.Sync()
}

// recover an invalid database by truncating back to the last complete event,
// then fsyncs so the recovery truncate is durable.
func (e *EntryDB[T, E, B]) recover() error {
	if err := e.data.Truncate(e.Size() * int64(e.b.EntrySize())); err != nil {
		return fmt.Errorf("failed to truncate trailing partial entries: %w", err)
	}
	if err := e.data.Sync(); err != nil {
		return fmt.Errorf("failed to sync after recovery truncate: %w", err)
	}
	return nil
}

func (e *EntryDB[T, E, B]) Close() error {
	return e.data.Close()
}
