package interop

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestVerifiedDB_WriteAndRead(t *testing.T) {
	// Create a temporary directory for the test database
	dataDir := t.TempDir()

	// Open the database
	db, err := OpenVerifiedDB(dataDir)
	require.NoError(t, err)
	defer db.Close()

	// Initially, there should be no last timestamp
	lastTs, initialized := db.LastTimestamp()
	require.False(t, initialized)
	require.Equal(t, uint64(0), lastTs)

	// Create test data
	chainID1 := eth.ChainIDFromUInt64(10)
	chainID2 := eth.ChainIDFromUInt64(8453)

	result1 := VerifiedResult{
		Timestamp: 1000,
		L1Inclusion: eth.BlockID{
			Hash:   common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111"),
			Number: 100,
		},
		L2Heads: map[eth.ChainID]eth.BlockID{
			chainID1: {
				Hash:   common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222"),
				Number: 200,
			},
			chainID2: {
				Hash:   common.HexToHash("0x3333333333333333333333333333333333333333333333333333333333333333"),
				Number: 300,
			},
		},
	}

	// Write the first result
	err = db.Commit(result1)
	require.NoError(t, err)

	// Verify the last timestamp was updated
	lastTs, initialized = db.LastTimestamp()
	require.True(t, initialized)
	require.Equal(t, uint64(1000), lastTs)

	// Read it back
	retrieved, err := db.Get(1000)
	require.NoError(t, err)
	require.Equal(t, result1.Timestamp, retrieved.Timestamp)
	require.Equal(t, result1.L1Inclusion, retrieved.L1Inclusion)
	require.Equal(t, len(result1.L2Heads), len(retrieved.L2Heads))
	require.Equal(t, result1.L2Heads[chainID1], retrieved.L2Heads[chainID1])
	require.Equal(t, result1.L2Heads[chainID2], retrieved.L2Heads[chainID2])

	// Check Has returns true
	has, err := db.Has(1000)
	require.NoError(t, err)
	require.True(t, has)

	// Check Has returns false for non-existent timestamp
	has, err = db.Has(999)
	require.NoError(t, err)
	require.False(t, has)

	// Try to read non-existent timestamp
	_, err = db.Get(999)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestVerifiedDB_SequentialCommits(t *testing.T) {
	dataDir := t.TempDir()

	db, err := OpenVerifiedDB(dataDir)
	require.NoError(t, err)
	defer db.Close()

	chainID := eth.ChainIDFromUInt64(10)

	// Commit first timestamp
	err = db.Commit(VerifiedResult{
		Timestamp:   100,
		L1Inclusion: eth.BlockID{Hash: common.HexToHash("0x01"), Number: 1},
		L2Heads:     map[eth.ChainID]eth.BlockID{chainID: {Hash: common.HexToHash("0x02"), Number: 2}},
	})
	require.NoError(t, err)

	// Commit next sequential timestamp should succeed
	err = db.Commit(VerifiedResult{
		Timestamp:   101,
		L1Inclusion: eth.BlockID{Hash: common.HexToHash("0x03"), Number: 3},
		L2Heads:     map[eth.ChainID]eth.BlockID{chainID: {Hash: common.HexToHash("0x04"), Number: 4}},
	})
	require.NoError(t, err)

	// Try to commit non-sequential timestamp (gap)
	err = db.Commit(VerifiedResult{
		Timestamp:   105,
		L1Inclusion: eth.BlockID{Hash: common.HexToHash("0x05"), Number: 5},
		L2Heads:     map[eth.ChainID]eth.BlockID{chainID: {Hash: common.HexToHash("0x06"), Number: 6}},
	})
	require.ErrorIs(t, err, ErrNonSequential)

	// Try to commit already committed timestamp
	err = db.Commit(VerifiedResult{
		Timestamp:   100,
		L1Inclusion: eth.BlockID{Hash: common.HexToHash("0x07"), Number: 7},
		L2Heads:     map[eth.ChainID]eth.BlockID{chainID: {Hash: common.HexToHash("0x08"), Number: 8}},
	})
	require.ErrorIs(t, err, ErrAlreadyCommitted)

	// Verify the database state is correct
	lastTs, _ := db.LastTimestamp()
	require.Equal(t, uint64(101), lastTs)
}

func TestVerifiedDB_Persistence(t *testing.T) {
	dataDir := t.TempDir()
	chainID := eth.ChainIDFromUInt64(42161)

	// Open database and write some data
	db, err := OpenVerifiedDB(dataDir)
	require.NoError(t, err)

	err = db.Commit(VerifiedResult{
		Timestamp:   500,
		L1Inclusion: eth.BlockID{Hash: common.HexToHash("0xaaaa"), Number: 50},
		L2Heads:     map[eth.ChainID]eth.BlockID{chainID: {Hash: common.HexToHash("0xbbbb"), Number: 100}},
	})
	require.NoError(t, err)

	err = db.Commit(VerifiedResult{
		Timestamp:   501,
		L1Inclusion: eth.BlockID{Hash: common.HexToHash("0xcccc"), Number: 51},
		L2Heads:     map[eth.ChainID]eth.BlockID{chainID: {Hash: common.HexToHash("0xdddd"), Number: 101}},
	})
	require.NoError(t, err)

	db.Close()

	// Reopen database and verify data persisted
	db2, err := OpenVerifiedDB(dataDir)
	require.NoError(t, err)
	defer db2.Close()

	// Last timestamp should be restored
	lastTs, initialized := db2.LastTimestamp()
	require.True(t, initialized)
	require.Equal(t, uint64(501), lastTs)

	// Data should be readable
	result, err := db2.Get(500)
	require.NoError(t, err)
	require.Equal(t, uint64(500), result.Timestamp)
	require.Equal(t, common.HexToHash("0xaaaa"), result.L1Inclusion.Hash)

	result, err = db2.Get(501)
	require.NoError(t, err)
	require.Equal(t, uint64(501), result.Timestamp)

	// Next commit should continue from last timestamp
	err = db2.Commit(VerifiedResult{
		Timestamp:   502,
		L1Inclusion: eth.BlockID{Hash: common.HexToHash("0xeeee"), Number: 52},
		L2Heads:     map[eth.ChainID]eth.BlockID{chainID: {Hash: common.HexToHash("0xffff"), Number: 102}},
	})
	require.NoError(t, err)
}

func TestVerifiedDB_RewindTo(t *testing.T) {
	t.Parallel()

	t.Run("removes entries at and after timestamp", func(t *testing.T) {
		t.Parallel()
		dataDir := t.TempDir()

		db, err := OpenVerifiedDB(dataDir)
		require.NoError(t, err)
		defer db.Close()

		chainID := eth.ChainIDFromUInt64(10)

		// Commit several timestamps
		for ts := uint64(100); ts <= 105; ts++ {
			err = db.Commit(VerifiedResult{
				Timestamp:   ts,
				L1Inclusion: eth.BlockID{Hash: common.BytesToHash([]byte{byte(ts)}), Number: ts},
				L2Heads:     map[eth.ChainID]eth.BlockID{chainID: {Hash: common.BytesToHash([]byte{byte(ts + 100)}), Number: ts}},
			})
			require.NoError(t, err)
		}

		// Verify all exist
		lastTs, _ := db.LastTimestamp()
		require.Equal(t, uint64(105), lastTs)

		// Rewind to 103 (should remove 103, 104, 105)
		deleted, err := db.Rewind(103)
		require.NoError(t, err)
		require.True(t, deleted)

		// Verify 100, 101, 102 still exist
		for ts := uint64(100); ts <= 102; ts++ {
			has, err := db.Has(ts)
			require.NoError(t, err)
			require.True(t, has, "timestamp %d should still exist", ts)
		}

		// Verify 103, 104, 105 are gone
		for ts := uint64(103); ts <= 105; ts++ {
			has, err := db.Has(ts)
			require.NoError(t, err)
			require.False(t, has, "timestamp %d should be deleted", ts)
		}

		// Last timestamp should be updated to 102
		lastTs, _ = db.LastTimestamp()
		require.Equal(t, uint64(102), lastTs)
	})

	t.Run("returns false when no entries deleted", func(t *testing.T) {
		t.Parallel()
		dataDir := t.TempDir()

		db, err := OpenVerifiedDB(dataDir)
		require.NoError(t, err)
		defer db.Close()

		chainID := eth.ChainIDFromUInt64(10)

		// Commit up to timestamp 100
		for ts := uint64(98); ts <= 100; ts++ {
			err = db.Commit(VerifiedResult{
				Timestamp:   ts,
				L1Inclusion: eth.BlockID{Hash: common.BytesToHash([]byte{byte(ts)}), Number: ts},
				L2Heads:     map[eth.ChainID]eth.BlockID{chainID: {Hash: common.BytesToHash([]byte{byte(ts + 100)}), Number: ts}},
			})
			require.NoError(t, err)
		}

		// Rewind to 200 (nothing to delete)
		deleted, err := db.Rewind(200)
		require.NoError(t, err)
		require.False(t, deleted)

		// All entries should still exist
		lastTs, _ := db.LastTimestamp()
		require.Equal(t, uint64(100), lastTs)
	})

	t.Run("rewind all entries", func(t *testing.T) {
		t.Parallel()
		dataDir := t.TempDir()

		db, err := OpenVerifiedDB(dataDir)
		require.NoError(t, err)
		defer db.Close()

		chainID := eth.ChainIDFromUInt64(10)

		// Commit a few entries
		for ts := uint64(100); ts <= 102; ts++ {
			err = db.Commit(VerifiedResult{
				Timestamp:   ts,
				L1Inclusion: eth.BlockID{Hash: common.BytesToHash([]byte{byte(ts)}), Number: ts},
				L2Heads:     map[eth.ChainID]eth.BlockID{chainID: {Hash: common.BytesToHash([]byte{byte(ts + 100)}), Number: ts}},
			})
			require.NoError(t, err)
		}

		// Rewind to 0 (delete all)
		deleted, err := db.Rewind(0)
		require.NoError(t, err)
		require.True(t, deleted)

		// No entries should exist
		for ts := uint64(100); ts <= 102; ts++ {
			has, err := db.Has(ts)
			require.NoError(t, err)
			require.False(t, has)
		}

		// Last timestamp should be reset to uninitialized
		_, initialized := db.LastTimestamp()
		require.False(t, initialized)
	})

	t.Run("allows sequential commits after rewind", func(t *testing.T) {
		t.Parallel()
		dataDir := t.TempDir()

		db, err := OpenVerifiedDB(dataDir)
		require.NoError(t, err)
		defer db.Close()

		chainID := eth.ChainIDFromUInt64(10)

		// Commit 100-105
		for ts := uint64(100); ts <= 105; ts++ {
			err = db.Commit(VerifiedResult{
				Timestamp:   ts,
				L1Inclusion: eth.BlockID{Hash: common.BytesToHash([]byte{byte(ts)}), Number: ts},
				L2Heads:     map[eth.ChainID]eth.BlockID{chainID: {Hash: common.BytesToHash([]byte{byte(ts + 100)}), Number: ts}},
			})
			require.NoError(t, err)
		}

		// Rewind to 103
		_, err = db.Rewind(103)
		require.NoError(t, err)

		// Should be able to commit 103 again (sequential from 102)
		err = db.Commit(VerifiedResult{
			Timestamp:   103,
			L1Inclusion: eth.BlockID{Hash: common.HexToHash("0xNEW"), Number: 103},
			L2Heads:     map[eth.ChainID]eth.BlockID{chainID: {Hash: common.HexToHash("0xNEW2"), Number: 103}},
		})
		require.NoError(t, err)

		// Verify new data
		result, err := db.Get(103)
		require.NoError(t, err)
		require.Equal(t, common.HexToHash("0xNEW"), result.L1Inclusion.Hash)
	})
}
