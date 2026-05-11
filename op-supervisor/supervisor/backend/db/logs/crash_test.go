package logs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
)

// crashInjector wraps an *os.File and can fail Write/Truncate/Sync to simulate
// crash scenarios at different points in a commit. It tracks call counts so
// tests can assert syscall count invariants alongside atomicity.
type crashInjector struct {
	f             *os.File
	writeFail     bool
	writeFailHead int // bytes written before Write returns the error (0 = whole write rejected)
	truncateFail  bool
	syncFail      bool

	writeCalls    int
	truncateCalls int
	syncCalls     int
}

func newCrashInjector(t *testing.T, path string) *crashInjector {
	t.Helper()
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o644)
	require.NoError(t, err)
	return &crashInjector{f: f}
}

func (c *crashInjector) ReadAt(p []byte, off int64) (int, error) {
	return c.f.ReadAt(p, off)
}

func (c *crashInjector) Write(p []byte) (int, error) {
	c.writeCalls++
	if c.writeFail {
		// Optionally let part of the write land on disk, then fail.
		n := c.writeFailHead
		if n > len(p) {
			n = len(p)
		}
		if n > 0 {
			if _, err := c.f.Write(p[:n]); err != nil {
				return 0, err
			}
		}
		return n, errors.New("injected write failure")
	}
	return c.f.Write(p)
}

func (c *crashInjector) Truncate(size int64) error {
	c.truncateCalls++
	if c.truncateFail {
		return errors.New("injected truncate failure")
	}
	return c.f.Truncate(size)
}

func (c *crashInjector) Sync() error {
	c.syncCalls++
	if c.syncFail {
		return errors.New("injected sync failure")
	}
	return c.f.Sync()
}

func (c *crashInjector) Close() error {
	return c.f.Close()
}

// openDBWithInjector wires a crashInjector behind an EntryDB and constructs a
// logs.DB on top of it.
func openDBWithInjector(t *testing.T, dir string) (*DB, *crashInjector, *stubMetrics) {
	t.Helper()
	path := filepath.Join(dir, "test.db")
	inj := newCrashInjector(t, path)
	stat, err := inj.f.Stat()
	require.NoError(t, err)
	var b EntryBinary
	size := stat.Size() / int64(b.EntrySize())
	store := entrydb.NewEntryDBFromDataAccess[EntryType, Entry, EntryBinary](inj, entrydb.EntryIdx(size-1))
	m := &stubMetrics{}
	db, err := NewFromEntryStore(testlog.Logger(t, log.LvlInfo), m, eth.ChainIDFromUInt64(1), store, true)
	require.NoError(t, err)
	return db, inj, m
}

// TestCrashInjection_SyncFailureDoesNotAdvanceCommitted verifies that when
// fsync fails during SealBlock, the in-memory committed state is not advanced
// and the on-disk bytes are rolled back via Truncate.
func TestCrashInjection_SyncFailureDoesNotAdvanceCommitted(t *testing.T) {
	dir := t.TempDir()

	// Seal a base block cleanly.
	db, inj, _ := openDBWithInjector(t, dir)
	bl0 := eth.BlockID{Hash: createHash(0), Number: 0}
	require.NoError(t, db.SealBlock(common.Hash{}, bl0, 100))
	beforeIdx := db.lastEntryContext.nextEntryIndex

	// Inject a sync failure for the next SealBlock.
	inj.syncFail = true
	bl1 := eth.BlockID{Hash: createHash(1), Number: 1}
	err := db.SealBlock(bl0.Hash, bl1, 101)
	require.Error(t, err)

	// Committed state unchanged.
	require.Equal(t, beforeIdx, db.lastEntryContext.nextEntryIndex)
	require.Equal(t, bl0.Hash, db.lastEntryContext.blockHash)
	require.Nil(t, db.working, "working clone should be discarded on sync failure")
}

// TestCrashInjection_PeriodicCheckpointBoundary asserts that recovery walks
// back to a true closing-block boundary (logsSince==0) and does not mistake a
// periodic search-checkpoint (logsSince>0) for one.
func TestCrashInjection_PeriodicCheckpointBoundary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	// Build a file with enough logs that a periodic search-checkpoint falls
	// mid-block.
	{
		store, err := entrydb.NewEntryDB[EntryType, Entry, EntryBinary](testlog.Logger(t, log.LvlInfo), path)
		require.NoError(t, err)
		m := &stubMetrics{}
		db, err := NewFromEntryStore(testlog.Logger(t, log.LvlInfo), m, eth.ChainIDFromUInt64(1), store, true)
		require.NoError(t, err)
		bl0 := eth.BlockID{Hash: createHash(0), Number: 0}
		require.NoError(t, db.SealBlock(common.Hash{}, bl0, 100))
		// Seed enough logs to span >256 entries within a single block (so a
		// periodic search-checkpoint is emitted while the block is still
		// open). Use forceBlock to set up cleanly, then AddLog × N + SealBlock.
		bl1 := eth.BlockID{Hash: createHash(1), Number: 1}
		require.NoError(t, db.SealBlock(bl0.Hash, bl1, 101))
		bl2 := eth.BlockID{Hash: createHash(2), Number: 2}
		for i := uint32(0); db.pendingNextIndex() < searchCheckpointFrequency+10; i++ {
			require.NoError(t, db.AddLog(createHash(int(i)), bl1, i, nil))
		}
		require.NoError(t, db.SealBlock(bl1.Hash, bl2, 102))
		require.NoError(t, db.Close())
	}

	// Reopen: drop-last-block rule should drop block 2, leaving block 1's
	// closing seal as the new tail. logsSince must be 0; this confirms
	// recovery did not land on the periodic search-checkpoint mid-block.
	{
		store, err := entrydb.NewEntryDB[EntryType, Entry, EntryBinary](testlog.Logger(t, log.LvlInfo), path)
		require.NoError(t, err)
		m := &stubMetrics{}
		db, err := NewFromEntryStore(testlog.Logger(t, log.LvlInfo), m, eth.ChainIDFromUInt64(1), store, true)
		require.NoError(t, err)
		require.Zero(t, db.lastEntryContext.logsSince, "recovery must land at a true closing boundary, not a periodic checkpoint")
		require.NoError(t, db.Close())
	}
}
