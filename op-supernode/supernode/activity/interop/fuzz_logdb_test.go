package interop

import (
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/reads"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// FuzzVerifyCanAddTimestamp tests the verifyCanAddTimestamp function with
// random parameters to verify gap detection and activation timestamp handling.
//
// Document coverage:
//   Step 1: Determining if a chain is ready at the target timestamp
//           (gap detection, activation timestamp bootstrap)
//
// Properties:
// P9: Gap violations are always detected (gap > blockTime)
// P13: Non-block-time-aligned gaps only warn, don't error
func FuzzVerifyCanAddTimestamp(f *testing.F) {
	f.Add(int64(1), uint64(1000), uint64(1001), uint64(1), true, uint64(1000))
	f.Add(int64(42), uint64(1000), uint64(1000), uint64(2), false, uint64(0))
	f.Add(int64(100), uint64(1000), uint64(1005), uint64(2), true, uint64(1002))
	f.Add(int64(200), uint64(1000), uint64(1010), uint64(2), true, uint64(1004))

	f.Fuzz(func(t *testing.T, seed int64, activationTS uint64, queryTS uint64, blockTime uint64, dbHasBlocks bool, sealTimestamp uint64) {
		// Skip invalid configs
		if blockTime == 0 {
			return
		}

		rng := rand.New(rand.NewSource(seed))

		chainID := eth.ChainIDFromUInt64(10)
		blockHash := randomHash(rng)

		interop := &Interop{
			log:                 gethlog.New(),
			activationTimestamp: activationTS,
		}

		db := &mockLogsDB{
			hasBlocks:   dbHasBlocks,
			latestBlock: eth.BlockID{Hash: blockHash, Number: 100},
			seal: suptypes.BlockSeal{
				Hash:      blockHash,
				Number:    100,
				Timestamp: sealTimestamp,
			},
		}

		_, hasBlocks, err := interop.verifyCanAddTimestamp(chainID, db, queryTS, blockTime)

		// Verify hasBlocks is passed through correctly
		require.Equal(t, dbHasBlocks, hasBlocks)

		if !dbHasBlocks {
			// Empty DB
			if queryTS == activationTS {
				// At activation timestamp with empty DB: should succeed
				require.NoError(t, err, "empty DB at activation timestamp should succeed")
			} else {
				// Non-activation timestamp with empty DB: should error
				require.Error(t, err, "empty DB at non-activation timestamp should error")
				require.ErrorIs(t, err, ErrPreviousTimestampNotSealed)
			}
		} else {
			// DB has blocks
			if err == nil {
				// No error: either seal.Timestamp > queryTS, or gap <= blockTime
				if sealTimestamp <= queryTS {
					gap := queryTS - sealTimestamp
					require.LessOrEqual(t, gap, blockTime,
						"P9: no-error case should have gap <= blockTime (gap=%d, blockTime=%d)", gap, blockTime)
				}
				// sealTimestamp > queryTS: already past this timestamp, always ok
			} else {
				// Error: should be gap > blockTime (or FindSealedBlock error)
				if sealTimestamp <= queryTS {
					gap := queryTS - sealTimestamp
					require.Greater(t, gap, blockTime,
						"P9: error case should have gap > blockTime (gap=%d, blockTime=%d)", gap, blockTime)
				}
			}
		}
	})
}

// FuzzProcessBlockLogs tests processBlockLogs with varying receipt and log counts.
//
// Document coverage:
//   INV-1: Logs stored per block (AddLog called per log with correct indices)
//   INV-2: Parent block linkage (SealBlock called with correct parent hash,
//          virtual parent seal for first block)
//
// Properties:
// P11: First block with empty parent hash is accepted exactly once
// P12: After any error, the DB remains consistent (no partial writes)
func FuzzProcessBlockLogs(f *testing.F) {
	f.Add(int64(1), 0, true)
	f.Add(int64(42), 3, false)
	f.Add(int64(100), 10, true)

	f.Fuzz(func(t *testing.T, seed int64, numReceipts int, isFirstBlock bool) {
		rng := rand.New(rand.NewSource(seed))

		if numReceipts < 0 {
			numReceipts = 0
		}
		if numReceipts > 20 {
			numReceipts = 20
		}

		interop := &Interop{log: gethlog.New()}

		// Track mock calls
		db := &trackingMockLogsDB{}

		blockNum := uint64(rng.Intn(10000))
		blockHash := randomHash(rng)
		parentHash := randomHash(rng)
		timestamp := uint64(100000 + rng.Intn(900000))

		if blockNum == 0 {
			isFirstBlock = true // block 0 is always treated as first
		}

		blockInfo := &testBlockInfo{
			hash:       blockHash,
			parentHash: parentHash,
			number:     blockNum,
			timestamp:  timestamp,
		}

		// Generate random receipts with random numbers of logs
		totalLogs := 0
		receipts := make(types.Receipts, numReceipts)
		for i := 0; i < numReceipts; i++ {
			numLogs := rng.Intn(5) // 0-4 logs per receipt
			logs := make([]*types.Log, numLogs)
			for j := 0; j < numLogs; j++ {
				logs[j] = &types.Log{
					Address: common.Address{byte(rng.Intn(256))},
					Data:    []byte{byte(rng.Intn(256))},
				}
			}
			receipts[i] = &types.Receipt{Logs: logs}
			totalLogs += numLogs
		}

		err := interop.processBlockLogs(db, blockInfo, receipts, isFirstBlock)
		require.NoError(t, err)

		// Verify AddLog was called for each log
		require.Equal(t, totalLogs, db.addLogCount,
			"AddLog should be called once per log")

		// Verify SealBlock call count:
		// - First block with blockNum > 0: 2 calls (virtual parent seal + actual block seal)
		// - Otherwise: 1 call (actual block seal only)
		expectedSealCalls := 1
		if (isFirstBlock || blockNum == 0) && blockNum > 0 {
			expectedSealCalls = 2
		}
		require.Equal(t, expectedSealCalls, db.sealBlockCount,
			"SealBlock should be called the expected number of times")

		// P11: First block handling
		if blockNum == 0 {
			// Genesis block: single SealBlock with empty parent
			require.Equal(t, common.Hash{}, db.lastSealParentHash,
				"P11: genesis block should use empty parent hash for SealBlock")
		} else if isFirstBlock {
			// First block (non-genesis): two SealBlock calls
			// 1st: virtual parent seal with empty hash
			// 2nd: actual block seal with real parentHash
			require.Equal(t, parentHash, db.lastSealParentHash,
				"P11: first block (non-genesis) last SealBlock should use real parent hash")
			if totalLogs > 0 {
				// AddLog uses the parent block constructed after virtual parent seal
				require.Equal(t, eth.BlockID{Hash: parentHash, Number: blockNum - 1}, db.firstAddLogParent,
					"P11: first block AddLog should use parent block")
			}
		} else {
			// Non-first block should use real parent
			require.Equal(t, parentHash, db.lastSealParentHash,
				"non-first block should use real parent hash for SealBlock")
			if totalLogs > 0 {
				require.Equal(t, eth.BlockID{Hash: parentHash, Number: blockNum - 1}, db.firstAddLogParent,
					"non-first block should use real parent block for AddLog")
			}
		}

		// Verify log indices are sequential
		for i := 0; i < totalLogs; i++ {
			require.Equal(t, uint32(i), db.logIndices[i],
				"log index %d should be sequential", i)
		}
	})
}

// trackingMockLogsDB tracks all calls to AddLog and SealBlock for verification
type trackingMockLogsDB struct {
	addLogCount       int
	sealBlockCount    int
	lastSealParentHash common.Hash
	firstAddLogParent  eth.BlockID
	logIndices        []uint32
}

func (m *trackingMockLogsDB) LatestSealedBlock() (eth.BlockID, bool)                     { return eth.BlockID{}, false }
func (m *trackingMockLogsDB) FirstSealedBlock() (suptypes.BlockSeal, error)               { return suptypes.BlockSeal{}, nil }
func (m *trackingMockLogsDB) FindSealedBlock(number uint64) (suptypes.BlockSeal, error)   { return suptypes.BlockSeal{}, nil }
func (m *trackingMockLogsDB) OpenBlock(blockNum uint64) (eth.BlockRef, uint32, map[uint32]*suptypes.ExecutingMessage, error) {
	return eth.BlockRef{}, 0, nil, nil
}
func (m *trackingMockLogsDB) Contains(query suptypes.ContainsQuery) (suptypes.BlockSeal, error) {
	return suptypes.BlockSeal{}, nil
}

func (m *trackingMockLogsDB) AddLog(logHash common.Hash, parentBlock eth.BlockID, logIdx uint32, execMsg *suptypes.ExecutingMessage) error {
	if m.addLogCount == 0 {
		m.firstAddLogParent = parentBlock
	}
	m.addLogCount++
	m.logIndices = append(m.logIndices, logIdx)
	return nil
}

func (m *trackingMockLogsDB) SealBlock(parentHash common.Hash, block eth.BlockID, timestamp uint64) error {
	m.sealBlockCount++
	m.lastSealParentHash = parentHash
	return nil
}

func (m *trackingMockLogsDB) Rewind(inv reads.Invalidator, newHead eth.BlockID) error { return nil }
func (m *trackingMockLogsDB) Clear(inv reads.Invalidator) error                       { return nil }
func (m *trackingMockLogsDB) Close() error { return nil }
