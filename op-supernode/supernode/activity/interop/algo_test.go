package interop

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// =============================================================================
// LogsDB Tests
// =============================================================================

func TestOpenLogsDB_NewAndClose(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	chainID := eth.ChainIDFromUInt64(10)

	// Open a new logs DB
	db, err := openLogsDB(gethlog.New(), chainID, dataDir)
	require.NoError(t, err)
	require.NotNil(t, db)

	// Close it
	err = db.Close()
	require.NoError(t, err)
}

func TestOpenLogsDB_SealBlock(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	chainID := eth.ChainIDFromUInt64(10)

	db, err := openLogsDB(gethlog.New(), chainID, dataDir)
	require.NoError(t, err)
	defer db.Close()

	// Seal a block
	parentHash := common.Hash{0x01}
	block := eth.BlockID{Hash: common.Hash{0x02}, Number: 100}
	timestamp := uint64(1000)

	err = db.SealBlock(parentHash, block, timestamp)
	require.NoError(t, err)

	// Verify the block was sealed
	latestBlock, ok := db.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, uint64(100), latestBlock.Number)
	require.Equal(t, block.Hash, latestBlock.Hash)
}

func TestOpenLogsDB_Persistence(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	chainID := eth.ChainIDFromUInt64(10)

	// Create and populate a logsDB
	{
		db, err := openLogsDB(gethlog.New(), chainID, dataDir)
		require.NoError(t, err)

		// Seal parent block
		parentBlock := eth.BlockID{Hash: common.Hash{0x01}, Number: 99}
		err = db.SealBlock(common.Hash{}, parentBlock, 998)
		require.NoError(t, err)

		// Add a log
		logHash := common.Hash{0x02}
		err = db.AddLog(logHash, parentBlock, 0, nil)
		require.NoError(t, err)

		// Seal block 100
		block100 := eth.BlockID{Hash: common.Hash{0x03}, Number: 100}
		err = db.SealBlock(parentBlock.Hash, block100, 1000)
		require.NoError(t, err)

		err = db.Close()
		require.NoError(t, err)
	}

	// Reopen and verify persistence
	{
		db, err := openLogsDB(gethlog.New(), chainID, dataDir)
		require.NoError(t, err)
		defer db.Close()

		// Verify the block is still there
		latestBlock, ok := db.LatestSealedBlock()
		require.True(t, ok)
		require.Equal(t, uint64(100), latestBlock.Number)
		require.Equal(t, common.Hash{0x03}, latestBlock.Hash)
	}
}

func TestOpenLogsDB_MultipleChains(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	chainID1 := eth.ChainIDFromUInt64(10)
	chainID2 := eth.ChainIDFromUInt64(8453)

	// Create logsDBs for two chains
	db1, err := openLogsDB(gethlog.New(), chainID1, dataDir)
	require.NoError(t, err)
	defer db1.Close()

	db2, err := openLogsDB(gethlog.New(), chainID2, dataDir)
	require.NoError(t, err)
	defer db2.Close()

	// Seal blocks on chain 1
	parentBlock1 := eth.BlockID{Hash: common.Hash{0x01}, Number: 99}
	err = db1.SealBlock(common.Hash{}, parentBlock1, 998)
	require.NoError(t, err)

	block100_1 := eth.BlockID{Hash: common.Hash{0x02}, Number: 100}
	err = db1.SealBlock(parentBlock1.Hash, block100_1, 1000)
	require.NoError(t, err)

	// Seal blocks on chain 2
	parentBlock2 := eth.BlockID{Hash: common.Hash{0x11}, Number: 199}
	err = db2.SealBlock(common.Hash{}, parentBlock2, 1998)
	require.NoError(t, err)

	block200_2 := eth.BlockID{Hash: common.Hash{0x12}, Number: 200}
	err = db2.SealBlock(parentBlock2.Hash, block200_2, 2000)
	require.NoError(t, err)

	// Verify chain 1
	latestBlock1, ok := db1.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, uint64(100), latestBlock1.Number)

	// Verify chain 2
	latestBlock2, ok := db2.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, uint64(200), latestBlock2.Number)
}
