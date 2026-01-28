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
		L1Head: eth.BlockID{
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
	require.Equal(t, result1.L1Head, retrieved.L1Head)
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
		Timestamp: 100,
		L1Head:    eth.BlockID{Hash: common.HexToHash("0x01"), Number: 1},
		L2Heads:   map[eth.ChainID]eth.BlockID{chainID: {Hash: common.HexToHash("0x02"), Number: 2}},
	})
	require.NoError(t, err)

	// Commit next sequential timestamp should succeed
	err = db.Commit(VerifiedResult{
		Timestamp: 101,
		L1Head:    eth.BlockID{Hash: common.HexToHash("0x03"), Number: 3},
		L2Heads:   map[eth.ChainID]eth.BlockID{chainID: {Hash: common.HexToHash("0x04"), Number: 4}},
	})
	require.NoError(t, err)

	// Try to commit non-sequential timestamp (gap)
	err = db.Commit(VerifiedResult{
		Timestamp: 105,
		L1Head:    eth.BlockID{Hash: common.HexToHash("0x05"), Number: 5},
		L2Heads:   map[eth.ChainID]eth.BlockID{chainID: {Hash: common.HexToHash("0x06"), Number: 6}},
	})
	require.ErrorIs(t, err, ErrNonSequential)

	// Try to commit already committed timestamp
	err = db.Commit(VerifiedResult{
		Timestamp: 100,
		L1Head:    eth.BlockID{Hash: common.HexToHash("0x07"), Number: 7},
		L2Heads:   map[eth.ChainID]eth.BlockID{chainID: {Hash: common.HexToHash("0x08"), Number: 8}},
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
		Timestamp: 500,
		L1Head:    eth.BlockID{Hash: common.HexToHash("0xaaaa"), Number: 50},
		L2Heads:   map[eth.ChainID]eth.BlockID{chainID: {Hash: common.HexToHash("0xbbbb"), Number: 100}},
	})
	require.NoError(t, err)

	err = db.Commit(VerifiedResult{
		Timestamp: 501,
		L1Head:    eth.BlockID{Hash: common.HexToHash("0xcccc"), Number: 51},
		L2Heads:   map[eth.ChainID]eth.BlockID{chainID: {Hash: common.HexToHash("0xdddd"), Number: 101}},
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
	require.Equal(t, common.HexToHash("0xaaaa"), result.L1Head.Hash)

	result, err = db2.Get(501)
	require.NoError(t, err)
	require.Equal(t, uint64(501), result.Timestamp)

	// Next commit should continue from last timestamp
	err = db2.Commit(VerifiedResult{
		Timestamp: 502,
		L1Head:    eth.BlockID{Hash: common.HexToHash("0xeeee"), Number: 52},
		L2Heads:   map[eth.ChainID]eth.BlockID{chainID: {Hash: common.HexToHash("0xffff"), Number: 102}},
	})
	require.NoError(t, err)
}
