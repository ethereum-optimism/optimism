package interop

import (
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/reads"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// =============================================================================
// TestLogsDB_Persistence
// =============================================================================

func TestLogsDB_Persistence(t *testing.T) {
	t.Parallel()

	t.Run("data survives close and reopen", func(t *testing.T) {
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

			latestBlock, ok := db.LatestSealedBlock()
			require.True(t, ok)
			require.Equal(t, uint64(100), latestBlock.Number)
			require.Equal(t, common.Hash{0x03}, latestBlock.Hash)
		}
	})

	t.Run("multiple chains are isolated", func(t *testing.T) {
		t.Parallel()
		dataDir := t.TempDir()

		chainID1 := eth.ChainIDFromUInt64(10)
		chainID2 := eth.ChainIDFromUInt64(8453)

		db1, err := openLogsDB(gethlog.New(), chainID1, dataDir)
		require.NoError(t, err)
		defer db1.Close()

		db2, err := openLogsDB(gethlog.New(), chainID2, dataDir)
		require.NoError(t, err)
		defer db2.Close()

		// Seal different blocks on each chain
		parentBlock1 := eth.BlockID{Hash: common.Hash{0x01}, Number: 99}
		err = db1.SealBlock(common.Hash{}, parentBlock1, 998)
		require.NoError(t, err)

		block1 := eth.BlockID{Hash: common.Hash{0x02}, Number: 100}
		err = db1.SealBlock(parentBlock1.Hash, block1, 1000)
		require.NoError(t, err)

		parentBlock2 := eth.BlockID{Hash: common.Hash{0x11}, Number: 199}
		err = db2.SealBlock(common.Hash{}, parentBlock2, 1998)
		require.NoError(t, err)

		block2 := eth.BlockID{Hash: common.Hash{0x12}, Number: 200}
		err = db2.SealBlock(parentBlock2.Hash, block2, 2000)
		require.NoError(t, err)

		// Verify each chain has its own data
		latestBlock1, ok := db1.LatestSealedBlock()
		require.True(t, ok)
		require.Equal(t, uint64(100), latestBlock1.Number)

		latestBlock2, ok := db2.LatestSealedBlock()
		require.True(t, ok)
		require.Equal(t, uint64(200), latestBlock2.Number)
	})
}

// =============================================================================
// TestVerifyPreviousTimestampSealed
// =============================================================================

func TestVerifyPreviousTimestampSealed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		activationTS  uint64
		queryTS       uint64
		blockTime     uint64
		dbHasBlocks   bool
		sealTimestamp uint64
		findSealErr   error
		wantErr       bool
		wantErrIs     error
		wantHashNil   bool
	}{
		{
			name:         "activation timestamp with empty DB returns nil hash",
			activationTS: 1000,
			queryTS:      1000,
			blockTime:    1,
			dbHasBlocks:  false,
			wantErr:      false,
			wantHashNil:  true,
		},
		{
			name:          "activation timestamp with non-empty DB succeeds (restart case)",
			activationTS:  1000,
			queryTS:       1000,
			blockTime:     1,
			dbHasBlocks:   true,
			sealTimestamp: 1000, // DB has block at activation timestamp
			wantErr:       false,
			wantHashNil:   false,
		},
		{
			name:         "non-activation timestamp with empty DB errors",
			activationTS: 1000,
			queryTS:      1001,
			blockTime:    1,
			dbHasBlocks:  false,
			wantErr:      true,
			wantErrIs:    ErrPreviousTimestampNotSealed,
			wantHashNil:  true,
		},
		{
			name:          "seal timestamp == query timestamp succeeds (already sealed)",
			activationTS:  1000,
			queryTS:       1001,
			blockTime:     1,
			dbHasBlocks:   true,
			sealTimestamp: 1001, // Same as queryTS - already past this timestamp
			wantErr:       false,
			wantHashNil:   false,
		},
		{
			name:          "seal timestamp > query timestamp succeeds (already past)",
			activationTS:  1000,
			queryTS:       1001,
			blockTime:     1,
			dbHasBlocks:   true,
			sealTimestamp: 1005, // Past queryTS
			wantErr:       false,
			wantHashNil:   false,
		},
		{
			name:          "seal timestamp < query timestamp (exact ts-1) succeeds",
			activationTS:  1000,
			queryTS:       1001,
			blockTime:     1,
			dbHasBlocks:   true,
			sealTimestamp: 1000, // gap = 1, blockTime = 1
			wantErr:       false,
			wantHashNil:   false,
		},
		{
			name:          "seal timestamp within block time succeeds",
			activationTS:  1000,
			queryTS:       1002,
			blockTime:     2, // blockTime = 2
			dbHasBlocks:   true,
			sealTimestamp: 1000, // gap = 2, blockTime = 2 - OK
			wantErr:       false,
			wantHashNil:   false,
		},
		{
			name:          "gap exceeds block time errors",
			activationTS:  1000,
			queryTS:       1003,
			blockTime:     2, // blockTime = 2
			dbHasBlocks:   true,
			sealTimestamp: 1000, // gap = 3, blockTime = 2 - ERROR
			wantErr:       true,
			wantErrIs:     ErrPreviousTimestampNotSealed,
			wantHashNil:   true,
		},
		{
			name:         "FindSealedBlock error propagated",
			activationTS: 1000,
			queryTS:      1001,
			blockTime:    1,
			dbHasBlocks:  true,
			findSealErr:  errors.New("database error"),
			wantErr:      true,
			wantHashNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			interop := &Interop{
				log:                 gethlog.New(),
				activationTimestamp: tt.activationTS,
			}
			chainID := eth.ChainIDFromUInt64(10)
			expectedHash := common.Hash{0x01}
			db := &mockLogsDB{
				hasBlocks:   tt.dbHasBlocks,
				latestBlock: eth.BlockID{Hash: expectedHash, Number: 100},
				seal: suptypes.BlockSeal{
					Hash:      expectedHash,
					Number:    100,
					Timestamp: tt.sealTimestamp,
				},
				findSealErr: tt.findSealErr,
			}

			block, _, err := interop.verifyCanAddTimestamp(chainID, db, tt.queryTS, tt.blockTime)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrIs != nil {
					require.ErrorIs(t, err, tt.wantErrIs)
				}
			} else {
				require.NoError(t, err)
			}

			if tt.wantHashNil {
				require.Equal(t, common.Hash{}, block.Hash, "expected zero hash")
			} else {
				require.NotEqual(t, common.Hash{}, block.Hash, "expected non-zero hash")
				require.Equal(t, expectedHash, block.Hash)
			}
		})
	}
}

// =============================================================================
// TestProcessBlockLogs
// =============================================================================

func TestProcessBlockLogs(t *testing.T) {
	t.Parallel()

	t.Run("empty receipts seals block with no logs", func(t *testing.T) {
		t.Parallel()

		interop := &Interop{log: gethlog.New()}
		db := &mockLogsDB{}
		blockInfo := &testBlockInfo{
			hash:       common.Hash{0x02},
			parentHash: common.Hash{0x01},
			number:     100,
			timestamp:  1000,
		}

		err := interop.processBlockLogs(db, blockInfo, types.Receipts{}, false)

		require.NoError(t, err)
		require.Len(t, db.sealBlockCalls, 1)
		require.Equal(t, common.Hash{0x01}, db.sealBlockCalls[0].parentHash)
		require.Equal(t, uint64(100), db.sealBlockCalls[0].block.Number)
		require.Equal(t, uint64(1000), db.sealBlockCalls[0].timestamp)
		require.Equal(t, 0, db.addLogCalls)
	})

	t.Run("multiple logs extracted from receipts", func(t *testing.T) {
		t.Parallel()

		interop := &Interop{log: gethlog.New()}
		db := &mockLogsDB{}
		blockInfo := &testBlockInfo{
			hash:       common.Hash{0x02},
			parentHash: common.Hash{0x01},
			number:     100,
			timestamp:  1000,
		}

		receipts := types.Receipts{
			&types.Receipt{
				Logs: []*types.Log{
					{Address: common.Address{0xAA}, Data: []byte{0x01}},
					{Address: common.Address{0xBB}, Data: []byte{0x02}},
				},
			},
			&types.Receipt{
				Logs: []*types.Log{
					{Address: common.Address{0xCC}, Data: []byte{0x03}},
				},
			},
		}

		err := interop.processBlockLogs(db, blockInfo, receipts, false)

		require.NoError(t, err)
		require.Equal(t, 3, db.addLogCalls)
		require.Len(t, db.sealBlockCalls, 1)
	})

	t.Run("genesis block handled correctly", func(t *testing.T) {
		t.Parallel()

		interop := &Interop{log: gethlog.New()}
		db := &mockLogsDB{}
		blockInfo := &testBlockInfo{
			hash:       common.Hash{0x01},
			parentHash: common.Hash{}, // Genesis has no parent
			number:     0,
			timestamp:  1000,
		}

		err := interop.processBlockLogs(db, blockInfo, types.Receipts{}, true)

		require.NoError(t, err)
		require.Len(t, db.sealBlockCalls, 1)
		require.Equal(t, uint64(0), db.sealBlockCalls[0].block.Number)
	})

	t.Run("first block at non-zero number seals virtual parent first", func(t *testing.T) {
		t.Parallel()

		interop := &Interop{log: gethlog.New()}
		db := &mockLogsDB{}
		blockInfo := &testBlockInfo{
			hash:       common.Hash{0x02},
			parentHash: common.Hash{0x01}, // Real parent hash
			number:     10,                // Non-zero block number
			timestamp:  1000,
		}

		// isFirstBlock=true should first seal a "virtual parent" block,
		// then seal the actual block. This allows logs to reference a sealed parent.
		err := interop.processBlockLogs(db, blockInfo, types.Receipts{}, true)

		require.NoError(t, err)
		require.Len(t, db.sealBlockCalls, 2)

		// First call: seal the virtual parent (block 9) with empty parent hash
		require.Equal(t, common.Hash{}, db.sealBlockCalls[0].parentHash)
		require.Equal(t, uint64(9), db.sealBlockCalls[0].block.Number)
		require.Equal(t, common.Hash{0x01}, db.sealBlockCalls[0].block.Hash)

		// Second call: seal the actual block (block 10) with real parent hash
		require.Equal(t, common.Hash{0x01}, db.sealBlockCalls[1].parentHash)
		require.Equal(t, uint64(10), db.sealBlockCalls[1].block.Number)
		require.Equal(t, common.Hash{0x02}, db.sealBlockCalls[1].block.Hash)
	})

	t.Run("first block with logs succeeds", func(t *testing.T) {
		t.Parallel()

		interop := &Interop{log: gethlog.New()}
		db := &mockLogsDB{}
		blockInfo := &testBlockInfo{
			hash:       common.Hash{0x02},
			parentHash: common.Hash{0x01},
			number:     100,
			timestamp:  1000,
		}

		receipts := types.Receipts{
			&types.Receipt{
				Logs: []*types.Log{
					{Address: common.Address{0xAA}, Data: []byte{0x01}},
				},
			},
		}

		// This is the key test: first block with logs should work because
		// we seal the virtual parent first, allowing AddLog to reference it
		err := interop.processBlockLogs(db, blockInfo, receipts, true)

		require.NoError(t, err)
		require.Len(t, db.sealBlockCalls, 2) // virtual parent + actual block
		require.Equal(t, 1, db.addLogCalls)
	})

	t.Run("integration: first block with logs against real DB", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		chainID := eth.ChainIDFromUInt64(10)

		db, err := openLogsDB(gethlog.New(), chainID, dataDir)
		require.NoError(t, err)
		defer db.Close()

		interop := &Interop{log: gethlog.New()}
		blockInfo := &testBlockInfo{
			hash:       common.Hash{0x02},
			parentHash: common.Hash{0x01},
			number:     100,
			timestamp:  1000,
		}
		receipts := types.Receipts{
			&types.Receipt{
				Logs: []*types.Log{
					{Address: common.Address{0xAA}, Data: []byte{0x01}},
					{Address: common.Address{0xBB}, Data: []byte{0x02}},
				},
			},
		}

		// This is the key integration test: first block with logs must work
		// against the real logs.DB, not just the mock.
		err = interop.processBlockLogs(db, blockInfo, receipts, true)
		require.NoError(t, err)

		// Verify data is correctly in the DB
		latestBlock, ok := db.LatestSealedBlock()
		require.True(t, ok)
		require.Equal(t, uint64(100), latestBlock.Number)
		require.Equal(t, common.Hash{0x02}, latestBlock.Hash)

		// Verify we can open the block and see the logs
		ref, logCount, _, err := db.OpenBlock(100)
		require.NoError(t, err)
		require.Equal(t, uint32(2), logCount)
		require.Equal(t, common.Hash{0x01}, ref.ParentHash)
	})

	t.Run("AddLog error propagated", func(t *testing.T) {
		t.Parallel()

		interop := &Interop{log: gethlog.New()}
		db := &mockLogsDB{addLogErr: errors.New("add log failed")}
		blockInfo := &testBlockInfo{
			hash:       common.Hash{0x02},
			parentHash: common.Hash{0x01},
			number:     100,
			timestamp:  1000,
		}
		receipts := types.Receipts{
			&types.Receipt{
				Logs: []*types.Log{{Address: common.Address{0xAA}}},
			},
		}

		err := interop.processBlockLogs(db, blockInfo, receipts, false)

		require.Error(t, err)
		require.Contains(t, err.Error(), "add log failed")
	})

	t.Run("SealBlock error propagated", func(t *testing.T) {
		t.Parallel()

		interop := &Interop{log: gethlog.New()}
		db := &mockLogsDB{sealBlockErr: errors.New("seal failed")}
		blockInfo := &testBlockInfo{
			hash:       common.Hash{0x02},
			parentHash: common.Hash{0x01},
			number:     100,
			timestamp:  1000,
		}

		err := interop.processBlockLogs(db, blockInfo, types.Receipts{}, false)

		require.Error(t, err)
		require.Contains(t, err.Error(), "seal failed")
	})
}

// =============================================================================
// Mock Types for LogsDB Tests
// =============================================================================

type mockLogsDB struct {
	latestBlock    eth.BlockID
	hasBlocks      bool
	seal           suptypes.BlockSeal
	findSealErr    error
	addLogErr      error
	sealBlockErr   error
	addLogCalls    int
	sealBlockCalls []*sealBlockCall // Track all SealBlock calls

	firstSealedBlock    suptypes.BlockSeal
	firstSealedBlockErr error

	openBlockRef     eth.BlockRef
	openBlockLogCnt  uint32
	openBlockExecMsg map[uint32]*suptypes.ExecutingMessage
	openBlockErr     error

	containsSeal suptypes.BlockSeal
	containsErr  error
}

type sealBlockCall struct {
	parentHash common.Hash
	block      eth.BlockID
	timestamp  uint64
}

func (m *mockLogsDB) LatestSealedBlock() (eth.BlockID, bool) {
	return m.latestBlock, m.hasBlocks
}

func (m *mockLogsDB) FirstSealedBlock() (suptypes.BlockSeal, error) {
	if m.firstSealedBlockErr != nil {
		return suptypes.BlockSeal{}, m.firstSealedBlockErr
	}
	return m.firstSealedBlock, nil
}

func (m *mockLogsDB) FindSealedBlock(number uint64) (suptypes.BlockSeal, error) {
	if m.findSealErr != nil {
		return suptypes.BlockSeal{}, m.findSealErr
	}
	return m.seal, nil
}

func (m *mockLogsDB) OpenBlock(blockNum uint64) (eth.BlockRef, uint32, map[uint32]*suptypes.ExecutingMessage, error) {
	if m.openBlockErr != nil {
		return eth.BlockRef{}, 0, nil, m.openBlockErr
	}
	return m.openBlockRef, m.openBlockLogCnt, m.openBlockExecMsg, nil
}

func (m *mockLogsDB) Contains(query suptypes.ContainsQuery) (suptypes.BlockSeal, error) {
	if m.containsErr != nil {
		return suptypes.BlockSeal{}, m.containsErr
	}
	return m.containsSeal, nil
}

func (m *mockLogsDB) AddLog(logHash common.Hash, parentBlock eth.BlockID, logIdx uint32, execMsg *suptypes.ExecutingMessage) error {
	m.addLogCalls++
	return m.addLogErr
}

func (m *mockLogsDB) SealBlock(parentHash common.Hash, block eth.BlockID, timestamp uint64) error {
	m.sealBlockCalls = append(m.sealBlockCalls, &sealBlockCall{
		parentHash: parentHash,
		block:      block,
		timestamp:  timestamp,
	})
	return m.sealBlockErr
}

func (m *mockLogsDB) Rewind(inv reads.Invalidator, newHead eth.BlockID) error { return nil }
func (m *mockLogsDB) Clear(inv reads.Invalidator) error                       { return nil }
func (m *mockLogsDB) Close() error                                            { return nil }

var _ LogsDB = (*mockLogsDB)(nil)
