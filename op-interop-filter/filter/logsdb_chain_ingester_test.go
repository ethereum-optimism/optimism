package filter

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-interop-filter/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// =============================================================================
// Test Helpers for LogsDBChainIngester Integration Tests
// =============================================================================

// testIngesterConfig holds configuration for test ingesters.
type testIngesterConfig struct {
	chainID   eth.ChainID
	dataDir   string
	ethClient EthClient
	rollupCfg *rollup.Config
}

// newTestLogsDBChainIngester creates a LogsDBChainIngester for testing
// with an injected mock EthClient.
func newTestLogsDBChainIngester(t *testing.T, cfg testIngesterConfig) *LogsDBChainIngester {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	logger := testlog.Logger(t, log.LevelError)

	ingester := &LogsDBChainIngester{
		log:              logger,
		metrics:          metrics.NoopMetrics,
		chainID:          cfg.chainID,
		ethClient:        cfg.ethClient,
		dataDir:          cfg.dataDir,
		startTimestamp:   10000, // Default for tests - high enough to avoid underflow
		backfillDuration: 0,     // No backfill by default in tests
		pollInterval:     100 * time.Millisecond,
		rollupCfg:        cfg.rollupCfg,
		fetchConcurrency: 4,
		ctx:              ctx,
		cancel:           cancel,
	}

	return ingester
}

// testRollupConfig creates a rollup config suitable for testing.
// l2StartBlock/l2StartTimestamp define where the L2 chain begins.
func testRollupConfig(chainID uint64, l2StartBlock uint64, l2StartTimestamp uint64) *rollup.Config {
	return &rollup.Config{
		L2ChainID: eth.ChainIDFromUInt64(chainID).ToBig(),
		BlockTime: 2, // 2 second blocks
		Genesis: rollup.Genesis{
			L2Time: l2StartTimestamp,
			L2: eth.BlockID{
				Number: l2StartBlock,
			},
		},
	}
}

// createTestBlock creates a mock block info for testing.
func createTestBlock(number uint64, timestamp uint64, parentHash common.Hash) *mockBlockInfo {
	hash := common.Hash{}
	hash[0] = byte(number)
	hash[1] = byte(number >> 8)

	return &mockBlockInfo{
		number:     number,
		hash:       hash,
		parentHash: parentHash,
		timestamp:  timestamp,
	}
}

// createTestReceipts creates test receipts with logs for the given block.
func createTestReceipts(blockNum uint64, logCount int) gethTypes.Receipts {
	var receipts gethTypes.Receipts

	if logCount == 0 {
		return receipts
	}

	logs := make([]*gethTypes.Log, logCount)
	for i := 0; i < logCount; i++ {
		logs[i] = &gethTypes.Log{
			Address: common.Address{byte(blockNum), byte(i)},
			Topics: []common.Hash{
				{0x01, 0x02, 0x03}, // Some topic
			},
			Data:  []byte{0x00}, // Minimal data
			Index: uint(i),
		}
	}

	receipts = append(receipts, &gethTypes.Receipt{
		TxHash: common.Hash{byte(blockNum)},
		Logs:   logs,
	})

	return receipts
}

type delayedFetchEthClient struct {
	*MockEthClient

	mu        sync.Mutex
	delays    map[uint64]time.Duration
	completed []uint64
}

func (m *delayedFetchEthClient) InfoByNumber(ctx context.Context, number uint64) (eth.BlockInfo, error) {
	if delay := m.delays[number]; delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	blockInfo, err := m.MockEthClient.InfoByNumber(ctx, number)
	m.mu.Lock()
	m.completed = append(m.completed, number)
	m.mu.Unlock()
	return blockInfo, err
}

func (m *delayedFetchEthClient) Completed() []uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]uint64(nil), m.completed...)
}

func (m *delayedFetchEthClient) ResetCompleted() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.completed = nil
}

type backfillProgressMetrics struct {
	metrics.Metricer

	chainID  uint64
	progress float64
}

func (m *backfillProgressMetrics) RecordBackfillProgress(chainID uint64, progress float64) {
	m.chainID = chainID
	m.progress = progress
}

// =============================================================================
// LogsDBChainIngester Integration Tests
// =============================================================================

func TestLogsDBChainIngester_InitLogsDB(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(901)

	// Create temp directory for test
	tempDir := t.TempDir()

	mockClient := NewMockEthClient()

	ingester := newTestLogsDBChainIngester(t, testIngesterConfig{
		chainID:   chainID,
		dataDir:   tempDir,
		ethClient: mockClient,
		rollupCfg: testRollupConfig(901, 0, 1000),
	})

	// Initialize the logsDB
	err := ingester.initLogsDB()
	require.NoError(t, err)

	// Verify the logsDB was created
	require.NotNil(t, ingester.logsDB)

	// Verify the chain directory was created
	chainDir := filepath.Join(tempDir, "chain-901")
	_, err = os.Stat(chainDir)
	require.NoError(t, err)

	// Clean up
	err = ingester.logsDB.Close()
	require.NoError(t, err)
}

func TestLogsDBChainIngester_SealParentBlock(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(901)
	tempDir := t.TempDir()

	// Set up mock client with a parent block
	mockClient := NewMockEthClient()
	parentBlock := createTestBlock(99, 1198, common.Hash{})
	mockClient.AddBlock(parentBlock, nil)

	ingester := newTestLogsDBChainIngester(t, testIngesterConfig{
		chainID:   chainID,
		dataDir:   tempDir,
		ethClient: mockClient,
		rollupCfg: testRollupConfig(901, 0, 1000),
	})

	// Initialize logsDB first
	err := ingester.initLogsDB()
	require.NoError(t, err)
	t.Cleanup(func() { ingester.logsDB.Close() })

	// Seal the parent block
	err = ingester.sealParentBlock(99)
	require.NoError(t, err)

	// Verify the block was sealed
	latestBlock, ok := ingester.LatestBlock()
	require.True(t, ok)
	require.Equal(t, uint64(99), latestBlock.Number)

	// Note: earliestIngestedBlock is NOT set in sealParentBlock anymore.
	// It's now set in ingestBlock when the first block with actual log data is ingested.
	// The anchor block is just a checkpoint, not a block with queryable log data.
	require.False(t, ingester.earliestIngestedBlockSet.Load(), "sealParentBlock should not set earliestIngestedBlock")
}

func TestLogsDBChainIngester_IngestBlock(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(901)
	tempDir := t.TempDir()

	// Set up mock client with blocks
	mockClient := NewMockEthClient()

	// Parent block (will be sealed first)
	parentBlock := createTestBlock(99, 1198, common.Hash{})
	mockClient.AddBlock(parentBlock, nil)

	// Block to ingest
	block100 := createTestBlock(100, 1200, parentBlock.Hash())
	receipts100 := createTestReceipts(100, 2) // 2 logs
	mockClient.AddBlock(block100, receipts100)

	ingester := newTestLogsDBChainIngester(t, testIngesterConfig{
		chainID:   chainID,
		dataDir:   tempDir,
		ethClient: mockClient,
		rollupCfg: testRollupConfig(901, 0, 1000),
	})

	// Initialize and seal parent
	err := ingester.initLogsDB()
	require.NoError(t, err)
	t.Cleanup(func() { ingester.logsDB.Close() })

	err = ingester.sealParentBlock(99)
	require.NoError(t, err)

	// Ingest block 100
	err = ingester.ingestBlock(100)
	require.NoError(t, err)

	// Verify the block was sealed
	latestBlock, ok := ingester.LatestBlock()
	require.True(t, ok)
	require.Equal(t, uint64(100), latestBlock.Number)
	require.Equal(t, block100.Hash(), latestBlock.Hash)

	// Verify timestamp
	ts, ok := ingester.LatestTimestamp()
	require.True(t, ok)
	require.Equal(t, uint64(1200), ts)
}

func TestLogsDBChainIngester_IngestMultipleBlocks(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(901)
	tempDir := t.TempDir()

	mockClient := NewMockEthClient()

	// Create a chain of 5 blocks
	parentHash := common.Hash{}
	for i := uint64(99); i <= 103; i++ {
		block := createTestBlock(i, 1000+i*2, parentHash)
		parentHash = block.Hash()

		var receipts gethTypes.Receipts
		if i >= 100 {
			receipts = createTestReceipts(i, 1)
		}
		mockClient.AddBlock(block, receipts)
	}

	ingester := newTestLogsDBChainIngester(t, testIngesterConfig{
		chainID:   chainID,
		dataDir:   tempDir,
		ethClient: mockClient,
		rollupCfg: testRollupConfig(901, 0, 1000),
	})

	err := ingester.initLogsDB()
	require.NoError(t, err)
	t.Cleanup(func() { ingester.logsDB.Close() })

	// Seal parent block 99
	err = ingester.sealParentBlock(99)
	require.NoError(t, err)

	// Ingest blocks 100-103
	for blockNum := uint64(100); blockNum <= 103; blockNum++ {
		err = ingester.ingestBlock(blockNum)
		require.NoError(t, err)
	}

	// Verify final state
	latestBlock, ok := ingester.LatestBlock()
	require.True(t, ok)
	require.Equal(t, uint64(103), latestBlock.Number)

	ts, ok := ingester.LatestTimestamp()
	require.True(t, ok)
	require.Equal(t, uint64(1206), ts) // 1000 + 103*2
}

func TestLogsDBChainIngester_IngestBlockRange_WritesInOrder(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(901)
	tempDir := t.TempDir()

	mockClient := &delayedFetchEthClient{
		MockEthClient: NewMockEthClient(),
		delays:        map[uint64]time.Duration{100: 50 * time.Millisecond},
	}

	parentHash := common.Hash{}
	for i := uint64(99); i <= 101; i++ {
		block := createTestBlock(i, 1000+i*2, parentHash)
		parentHash = block.Hash()

		var receipts gethTypes.Receipts
		if i >= 100 {
			receipts = createTestReceipts(i, 1)
		}
		mockClient.AddBlock(block, receipts)
	}

	ingester := newTestLogsDBChainIngester(t, testIngesterConfig{
		chainID:   chainID,
		dataDir:   tempDir,
		ethClient: mockClient,
		rollupCfg: testRollupConfig(901, 0, 1000),
	})
	ingester.fetchConcurrency = 2

	require.NoError(t, ingester.initLogsDB())
	t.Cleanup(func() { ingester.logsDB.Close() })
	require.NoError(t, ingester.sealParentBlock(99))
	mockClient.ResetCompleted()

	nextBlock, _, err := ingester.ingestBlockRange(100, 101, clock.SystemClock.Now())
	require.NoError(t, err)
	require.Equal(t, uint64(102), nextBlock)
	require.Equal(t, []uint64{101, 100}, mockClient.Completed())

	latestBlock, ok := ingester.LatestBlock()
	require.True(t, ok)
	require.Equal(t, uint64(101), latestBlock.Number)
}

func TestLogsDBChainIngester_IngestBlockRange_FetchFailureReturnsFailingBlock(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(901)
	tempDir := t.TempDir()

	mockClient := NewMockEthClient()

	parentBlock := createTestBlock(99, 1198, common.Hash{})
	block100 := createTestBlock(100, 1200, parentBlock.Hash())
	block101 := createTestBlock(101, 1202, block100.Hash())
	mockClient.AddBlock(parentBlock, nil)
	mockClient.AddBlock(block100, createTestReceipts(100, 1))
	mockClient.AddBlock(block101, createTestReceipts(101, 1))

	ingester := newTestLogsDBChainIngester(t, testIngesterConfig{
		chainID:   chainID,
		dataDir:   tempDir,
		ethClient: mockClient,
		rollupCfg: testRollupConfig(901, 0, 1000),
	})
	ingester.fetchConcurrency = 2

	require.NoError(t, ingester.initLogsDB())
	t.Cleanup(func() { ingester.logsDB.Close() })
	require.NoError(t, ingester.sealParentBlock(99))

	nextBlock, _, err := ingester.ingestBlockRange(100, 102, clock.SystemClock.Now())
	require.Error(t, err)
	require.Contains(t, err.Error(), "block 102 not found")
	require.Equal(t, uint64(102), nextBlock)

	latestBlock, ok := ingester.LatestBlock()
	require.True(t, ok)
	require.Equal(t, uint64(101), latestBlock.Number)
}

func TestLogsDBChainIngester_RecordIngestionProgressCountsCompletedBlock(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(901)
	recorder := &backfillProgressMetrics{Metricer: metrics.NoopMetrics}

	ingester := newTestLogsDBChainIngester(t, testIngesterConfig{
		chainID:   chainID,
		dataDir:   t.TempDir(),
		ethClient: NewMockEthClient(),
		rollupCfg: testRollupConfig(901, 0, 0),
	})
	ingester.metrics = recorder
	ingester.startTimestamp = 400
	ingester.backfillDuration = 0
	ingester.earliestIngestedBlock.Store(100)
	ingester.earliestIngestedBlockSet.Store(true)

	ingester.recordIngestionProgress(200, 200)

	require.Equal(t, uint64(901), recorder.chainID)
	require.Equal(t, 1.0, recorder.progress)
}

func TestLogsDBChainIngester_Ready(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(901)
	tempDir := t.TempDir()

	mockClient := NewMockEthClient()

	// Create blocks - block 100 has timestamp 1200
	parentBlock := createTestBlock(99, 1198, common.Hash{})
	mockClient.AddBlock(parentBlock, nil)

	block100 := createTestBlock(100, 1200, parentBlock.Hash())
	mockClient.AddBlock(block100, createTestReceipts(100, 1))

	ingester := newTestLogsDBChainIngester(t, testIngesterConfig{
		chainID:   chainID,
		dataDir:   tempDir,
		ethClient: mockClient,
		rollupCfg: testRollupConfig(901, 0, 1000),
	})

	// Set startTimestamp to 1200 (block 100's timestamp) for this test
	ingester.startTimestamp = 1200

	// Not ready initially - no blocks ingested yet
	require.False(t, ingester.Ready())

	err := ingester.initLogsDB()
	require.NoError(t, err)
	t.Cleanup(func() { ingester.logsDB.Close() })

	// Still not ready - no blocks sealed
	require.False(t, ingester.Ready())

	// Seal parent and ingest block 100
	err = ingester.sealParentBlock(99)
	require.NoError(t, err)

	// Still not ready - latest timestamp is 1198 (block 99), startTimestamp is 1200
	require.False(t, ingester.Ready())

	err = ingester.ingestBlock(100)
	require.NoError(t, err)

	// Now ready - latestTimestamp (1200) >= startTimestamp (1200)
	require.True(t, ingester.Ready())
}

func TestLogsDBChainIngester_ErrorState(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(901)
	tempDir := t.TempDir()

	mockClient := NewMockEthClient()

	ingester := newTestLogsDBChainIngester(t, testIngesterConfig{
		chainID:   chainID,
		dataDir:   tempDir,
		ethClient: mockClient,
		rollupCfg: testRollupConfig(901, 0, 1000),
	})

	// No error initially
	require.Nil(t, ingester.Error())

	// Set error
	ingester.SetError(ErrorReorg, "reorg detected at block 100")
	err := ingester.Error()
	require.NotNil(t, err)
	require.Equal(t, ErrorReorg, err.Reason)
	require.Contains(t, err.Message, "reorg detected")

	// Clear error
	ingester.ClearError()
	require.Nil(t, ingester.Error())
}

func TestLogsDBChainIngester_Contains(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(901)
	tempDir := t.TempDir()

	mockClient := NewMockEthClient()

	// Create blocks with logs
	parentBlock := createTestBlock(99, 1198, common.Hash{})
	mockClient.AddBlock(parentBlock, nil)

	block100 := createTestBlock(100, 1200, parentBlock.Hash())
	receipts100 := createTestReceipts(100, 2)
	mockClient.AddBlock(block100, receipts100)

	ingester := newTestLogsDBChainIngester(t, testIngesterConfig{
		chainID:   chainID,
		dataDir:   tempDir,
		ethClient: mockClient,
		rollupCfg: testRollupConfig(901, 0, 1000),
	})

	// Contains should fail when logsDB not initialized
	_, err := ingester.Contains(types.ContainsQuery{})
	require.ErrorIs(t, err, types.ErrUninitialized)

	err = ingester.initLogsDB()
	require.NoError(t, err)
	t.Cleanup(func() { ingester.logsDB.Close() })

	// Seal parent and ingest block
	err = ingester.sealParentBlock(99)
	require.NoError(t, err)

	err = ingester.ingestBlock(100)
	require.NoError(t, err)

	// Query for non-existent log should return ErrConflict
	_, err = ingester.Contains(types.ContainsQuery{
		Timestamp: 1200,
		BlockNum:  100,
		LogIdx:    99, // Doesn't exist
		Checksum:  types.MessageChecksum{0xFF},
	})
	require.ErrorIs(t, err, types.ErrConflict)
}

func TestLogsDBChainIngester_ReorgDetection(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(901)
	tempDir := t.TempDir()

	mockClient := NewMockEthClient()

	// Create blocks
	parentBlock := createTestBlock(99, 1198, common.Hash{})
	mockClient.AddBlock(parentBlock, nil)

	block100 := createTestBlock(100, 1200, parentBlock.Hash())
	mockClient.AddBlock(block100, createTestReceipts(100, 1))

	ingester := newTestLogsDBChainIngester(t, testIngesterConfig{
		chainID:   chainID,
		dataDir:   tempDir,
		ethClient: mockClient,
		rollupCfg: testRollupConfig(901, 0, 1000),
	})

	err := ingester.initLogsDB()
	require.NoError(t, err)
	t.Cleanup(func() { ingester.logsDB.Close() })

	err = ingester.sealParentBlock(99)
	require.NoError(t, err)

	err = ingester.ingestBlock(100)
	require.NoError(t, err)

	// Now try to ingest a block with wrong parent hash (simulating a reorg)
	reorgBlock := createTestBlock(101, 1202, common.Hash{0xDE, 0xAD}) // Wrong parent
	mockClient.AddBlock(reorgBlock, createTestReceipts(101, 1))

	err = ingester.ingestBlock(101)
	require.NoError(t, err) // ingestBlock doesn't return error, it sets error state

	// Check that error state was set
	ingesterErr := ingester.Error()
	require.NotNil(t, ingesterErr)
	require.Equal(t, ErrorReorg, ingesterErr.Reason)
}

func TestLogsDBChainIngester_RewindToFinalized(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(901)
	tempDir := t.TempDir()

	mockClient := NewMockEthClient()

	parentBlock := createTestBlock(99, 1198, common.Hash{})
	mockClient.AddBlock(parentBlock, nil)

	block100 := createTestBlock(100, 1200, parentBlock.Hash())
	mockClient.AddBlock(block100, createTestReceipts(100, 1))

	block101 := createTestBlock(101, 1202, block100.Hash())
	mockClient.AddBlock(block101, createTestReceipts(101, 1))
	mockClient.SetLabelBlock(eth.Finalized, block100)

	ingester := newTestLogsDBChainIngester(t, testIngesterConfig{
		chainID:   chainID,
		dataDir:   tempDir,
		ethClient: mockClient,
		rollupCfg: testRollupConfig(901, 0, 1000),
	})

	err := ingester.initLogsDB()
	require.NoError(t, err)
	t.Cleanup(func() { ingester.logsDB.Close() })

	require.NoError(t, ingester.sealParentBlock(99))
	require.NoError(t, ingester.ingestBlock(100))
	require.NoError(t, ingester.ingestBlock(101))

	target, timestamp, err := ingester.RewindToFinalized(context.Background())
	require.NoError(t, err)
	require.Equal(t, eth.BlockID{Hash: block100.Hash(), Number: 100}, target)
	require.Equal(t, uint64(1200), timestamp)
	require.Equal(t, uint64(101), ingester.applyPendingRewind(102))
	require.Equal(t, uint64(102), ingester.applyPendingRewind(102))

	latest, ok := ingester.LatestBlock()
	require.True(t, ok)
	require.Equal(t, target, latest)
}

func TestLogsDBChainIngester_QueryMethods(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(901)
	tempDir := t.TempDir()

	mockClient := NewMockEthClient()

	parentBlock := createTestBlock(99, 1198, common.Hash{})
	mockClient.AddBlock(parentBlock, nil)

	block100 := createTestBlock(100, 1200, parentBlock.Hash())
	mockClient.AddBlock(block100, createTestReceipts(100, 1))

	ingester := newTestLogsDBChainIngester(t, testIngesterConfig{
		chainID:   chainID,
		dataDir:   tempDir,
		ethClient: mockClient,
		rollupCfg: testRollupConfig(901, 0, 1000),
	})

	// Uninitialized queries should fail
	_, err := ingester.GetExecMsgsAtTimestamp(1200)
	require.ErrorIs(t, err, types.ErrUninitialized)
	_, ok := ingester.BlockHashAt(100)
	require.False(t, ok)

	err = ingester.initLogsDB()
	require.NoError(t, err)
	t.Cleanup(func() { ingester.logsDB.Close() })

	err = ingester.sealParentBlock(99)
	require.NoError(t, err)

	err = ingester.ingestBlock(100)
	require.NoError(t, err)

	// GetExecMsgsAtTimestamp: existing timestamp
	msgs, err := ingester.GetExecMsgsAtTimestamp(1200)
	require.NoError(t, err)
	_ = msgs

	// GetExecMsgsAtTimestamp: non-existent timestamp
	msgs, err = ingester.GetExecMsgsAtTimestamp(9999)
	require.NoError(t, err)
	require.Empty(t, msgs)

	// BlockHashAt: existing block
	hash, ok := ingester.BlockHashAt(100)
	require.True(t, ok)
	require.Equal(t, block100.Hash(), hash)

	// BlockHashAt: non-existent block
	_, ok = ingester.BlockHashAt(999)
	require.False(t, ok)
}

// TestLogsDBChainIngester_GetExecMsgsAtTimestamp_AnchorOnly verifies that
// GetExecMsgsAtTimestamp returns ErrUninitialized when the DB contains only
// the anchor block (no block with log data has been ingested yet). It must
// not silently report "no executing messages" in this state.
func TestLogsDBChainIngester_GetExecMsgsAtTimestamp_AnchorOnly(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(901)
	tempDir := t.TempDir()

	mockClient := NewMockEthClient()

	parentBlock := createTestBlock(99, 1198, common.Hash{})
	mockClient.AddBlock(parentBlock, nil)

	block100 := createTestBlock(100, 1200, parentBlock.Hash())
	mockClient.AddBlock(block100, createTestReceipts(100, 1))

	ingester := newTestLogsDBChainIngester(t, testIngesterConfig{
		chainID:   chainID,
		dataDir:   tempDir,
		ethClient: mockClient,
		rollupCfg: testRollupConfig(901, 0, 1000),
	})

	require.NoError(t, ingester.initLogsDB())
	t.Cleanup(func() { ingester.logsDB.Close() })

	require.NoError(t, ingester.sealParentBlock(99))
	require.False(t, ingester.earliestIngestedBlockSet.Load(),
		"sealParentBlock alone must not mark the ingester initialized")

	// The anchor block is in range (blockNum <= latestSealed) but no block
	// has been ingested with logs yet, so the lower bound is unset. The
	// query must surface ErrUninitialized rather than silently returning
	// (nil, nil) — that distinguishes "we haven't ingested anything yet"
	// from "no executing messages at this timestamp".
	_, err := ingester.GetExecMsgsAtTimestamp(1198)
	require.ErrorIs(t, err, types.ErrUninitialized)

	// Queries past the latest sealed block still return (nil, nil) — that's
	// the existing "block not sealed yet" semantics, unchanged by this fix.
	msgs, err := ingester.GetExecMsgsAtTimestamp(1200)
	require.NoError(t, err)
	require.Empty(t, msgs)
}

// TestLogsDBChainIngester_GetExecMsgsAtTimestamp_BeforeEarliest verifies that
// once the ingester is initialized, queries for blocks older than
// earliestIngestedBlock return (nil, nil) rather than an error. This is the
// "we don't have data that old" branch, distinct from "uninitialized".
func TestLogsDBChainIngester_GetExecMsgsAtTimestamp_BeforeEarliest(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(901)
	tempDir := t.TempDir()

	mockClient := NewMockEthClient()

	parentBlock := createTestBlock(99, 1198, common.Hash{})
	mockClient.AddBlock(parentBlock, nil)

	block100 := createTestBlock(100, 1200, parentBlock.Hash())
	mockClient.AddBlock(block100, createTestReceipts(100, 1))

	ingester := newTestLogsDBChainIngester(t, testIngesterConfig{
		chainID:   chainID,
		dataDir:   tempDir,
		ethClient: mockClient,
		rollupCfg: testRollupConfig(901, 0, 1000),
	})

	require.NoError(t, ingester.initLogsDB())
	t.Cleanup(func() { ingester.logsDB.Close() })

	require.NoError(t, ingester.sealParentBlock(99))
	require.NoError(t, ingester.ingestBlock(100))

	require.True(t, ingester.earliestIngestedBlockSet.Load())
	require.Equal(t, uint64(100), ingester.earliestIngestedBlock.Load())

	// Anchor block 99 (timestamp 1198) is sealed but predates earliest=100.
	// The query path must distinguish this from the uninitialized state.
	msgs, err := ingester.GetExecMsgsAtTimestamp(1198)
	require.NoError(t, err)
	require.Empty(t, msgs)
}

// =============================================================================
// Integration Test: Full Ingestion Flow with Real LogsDB
// =============================================================================

func TestLogsDBChainIngester_Integration_RealLogsDB(t *testing.T) {
	// This test exercises the full ingestion flow:
	// 1. Create MockEthClient with test blocks
	// 2. Initialize LogsDBChainIngester with real logsDB
	// 3. Ingest multiple blocks
	// 4. Verify data can be queried back

	chainID := eth.ChainIDFromUInt64(901)
	tempDir := t.TempDir()

	mockClient := NewMockEthClient()

	// Create a chain of 10 blocks
	numBlocks := 10
	startBlock := uint64(100)
	startTimestamp := uint64(2000)

	parentHash := common.Hash{0x00}
	for i := 0; i < numBlocks+1; i++ { // +1 for parent block
		blockNum := startBlock - 1 + uint64(i)
		timestamp := startTimestamp + uint64(i)*2

		block := createTestBlock(blockNum, timestamp, parentHash)
		parentHash = block.Hash()

		var receipts gethTypes.Receipts
		if blockNum >= startBlock {
			receipts = createTestReceipts(blockNum, 1)
		}
		mockClient.AddBlock(block, receipts)
	}

	// Set head block
	headBlock := createTestBlock(startBlock+uint64(numBlocks)-1, startTimestamp+uint64(numBlocks)*2-2, common.Hash{})
	mockClient.SetHeadBlock(headBlock)

	ingester := newTestLogsDBChainIngester(t, testIngesterConfig{
		chainID:   chainID,
		dataDir:   tempDir,
		ethClient: mockClient,
		rollupCfg: testRollupConfig(901, 0, 1000),
	})

	// Initialize
	err := ingester.initLogsDB()
	require.NoError(t, err)
	t.Cleanup(func() { ingester.logsDB.Close() })

	// Seal parent block
	err = ingester.sealParentBlock(startBlock - 1)
	require.NoError(t, err)

	// Ingest all blocks
	for blockNum := startBlock; blockNum < startBlock+uint64(numBlocks); blockNum++ {
		err = ingester.ingestBlock(blockNum)
		require.NoError(t, err)
	}

	// Verify final state
	latestBlock, ok := ingester.LatestBlock()
	require.True(t, ok)
	require.Equal(t, startBlock+uint64(numBlocks)-1, latestBlock.Number)

	latestTs, ok := ingester.LatestTimestamp()
	require.True(t, ok)
	// Block 109 (i=10) has timestamp = 2000 + 10*2 = 2020
	expectedTs := startTimestamp + uint64(numBlocks)*2
	require.Equal(t, expectedTs, latestTs)

	// Verify block hashes
	for blockNum := startBlock; blockNum < startBlock+uint64(numBlocks); blockNum++ {
		hash, ok := ingester.BlockHashAt(blockNum)
		require.True(t, ok)
		require.NotEqual(t, common.Hash{}, hash)
	}
}

// =============================================================================
// initIngestion Tests
// =============================================================================

func TestLogsDBChainIngester_InitIngestion_FreshStart(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(901)
	tempDir := t.TempDir()

	mockClient := NewMockEthClient()

	// L2 chain starts at block 100, timestamp 1000
	// startTimestamp=1100, backfillDuration=200s -> backfillTimestamp=900 (before genesis)
	// This should cause TargetBlockNumber to fail and fall back to genesis.
	l2StartBlock := uint64(100)
	l2StartTimestamp := uint64(1000)

	headBlock := createTestBlock(200, 1200, common.Hash{0x99})
	mockClient.AddBlock(headBlock, nil)
	mockClient.SetHeadBlock(headBlock)

	// Need parent block for sealing (block 99)
	parentBlock := createTestBlock(l2StartBlock-1, l2StartTimestamp-2, common.Hash{0x98})
	mockClient.AddBlock(parentBlock, nil)

	ingester := newTestLogsDBChainIngester(t, testIngesterConfig{
		chainID:   chainID,
		dataDir:   tempDir,
		ethClient: mockClient,
		rollupCfg: testRollupConfig(901, l2StartBlock, l2StartTimestamp),
	})

	// Set up for backfill that goes before genesis (but no underflow)
	ingester.startTimestamp = 1100
	ingester.backfillDuration = 200 * time.Second // backfillTimestamp = 900 < genesis (1000)

	err := ingester.initLogsDB()
	require.NoError(t, err)
	t.Cleanup(func() { ingester.logsDB.Close() })

	// Call initIngestion - should fall back to L2 start block since backfill is before genesis
	nextBlock, err := ingester.initIngestion()
	require.NoError(t, err)

	// Should start from L2 start block (genesis fallback)
	startingBlock := ingester.calculateStartingBlock()
	require.Equal(t, l2StartBlock, startingBlock)
	require.Equal(t, startingBlock, nextBlock)
}

func TestLogsDBChainIngester_CalculateStartingBlock_BackfillUnderflow(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(901)
	tempDir := t.TempDir()

	mockClient := NewMockEthClient()

	l2StartBlock := uint64(100)
	l2StartTimestamp := uint64(1000)

	headBlock := createTestBlock(200, 1200, common.Hash{0x99})
	mockClient.AddBlock(headBlock, nil)
	mockClient.SetHeadBlock(headBlock)

	ingester := newTestLogsDBChainIngester(t, testIngesterConfig{
		chainID:   chainID,
		dataDir:   tempDir,
		ethClient: mockClient,
		rollupCfg: testRollupConfig(901, l2StartBlock, l2StartTimestamp),
	})

	// startTimestamp=50, backfillDuration=200s would underflow without the guard
	ingester.startTimestamp = 50
	ingester.backfillDuration = 200 * time.Second

	startingBlock := ingester.calculateStartingBlock()
	require.Equal(t, l2StartBlock, startingBlock)
}

func TestLogsDBChainIngester_InitIngestion_ResumeFromExistingDB(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(901)
	tempDir := t.TempDir()

	mockClient := NewMockEthClient()

	// Set up blocks
	parentBlock := createTestBlock(99, 1198, common.Hash{})
	mockClient.AddBlock(parentBlock, nil)

	block100 := createTestBlock(100, 1200, parentBlock.Hash())
	mockClient.AddBlock(block100, createTestReceipts(100, 1))

	block101 := createTestBlock(101, 1202, block100.Hash())
	mockClient.AddBlock(block101, createTestReceipts(101, 1))

	headBlock := createTestBlock(200, 3000, common.Hash{0x99})
	mockClient.AddBlock(headBlock, nil)
	mockClient.SetHeadBlock(headBlock)

	// First ingester: seal some blocks
	ingester1 := newTestLogsDBChainIngester(t, testIngesterConfig{
		chainID:   chainID,
		dataDir:   tempDir,
		ethClient: mockClient,
		rollupCfg: testRollupConfig(901, 0, 1000),
	})

	err := ingester1.initLogsDB()
	require.NoError(t, err)

	err = ingester1.sealParentBlock(99)
	require.NoError(t, err)

	err = ingester1.ingestBlock(100)
	require.NoError(t, err)

	err = ingester1.ingestBlock(101)
	require.NoError(t, err)

	// Close first ingester
	err = ingester1.logsDB.Close()
	require.NoError(t, err)

	// Second ingester: should resume from existing DB
	ingester2 := newTestLogsDBChainIngester(t, testIngesterConfig{
		chainID:   chainID,
		dataDir:   tempDir,
		ethClient: mockClient,
		rollupCfg: testRollupConfig(901, 0, 1000),
	})

	err = ingester2.initLogsDB()
	require.NoError(t, err)
	t.Cleanup(func() { ingester2.logsDB.Close() })

	// Call initIngestion - should resume from block 102
	nextBlock, err := ingester2.initIngestion()
	require.NoError(t, err)
	require.Equal(t, uint64(102), nextBlock) // Should resume after block 101

	// Verify earliest block was found
	require.True(t, ingester2.earliestIngestedBlockSet.Load())
	require.Equal(t, uint64(100), ingester2.earliestIngestedBlock.Load())
}

// TestLogsDBChainIngester_InitIngestion_ResumeAnchorOnly covers the case where
// a previous run died after sealing the anchor but before ingesting any block.
// On resume the DB contains only the anchor, so initIngestion must succeed
// without setting earliestIngestedBlockSet — ingestion of the first real block
// will set it, exactly like the fresh-start path.
func TestLogsDBChainIngester_InitIngestion_ResumeAnchorOnly(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(901)
	tempDir := t.TempDir()

	mockClient := NewMockEthClient()

	parentBlock := createTestBlock(99, 1198, common.Hash{})
	mockClient.AddBlock(parentBlock, nil)

	block100 := createTestBlock(100, 1200, parentBlock.Hash())
	mockClient.AddBlock(block100, createTestReceipts(100, 1))

	headBlock := createTestBlock(200, 3000, common.Hash{0x99})
	mockClient.AddBlock(headBlock, nil)
	mockClient.SetHeadBlock(headBlock)

	// First ingester: seal the anchor only, then close.
	ingester1 := newTestLogsDBChainIngester(t, testIngesterConfig{
		chainID:   chainID,
		dataDir:   tempDir,
		ethClient: mockClient,
		rollupCfg: testRollupConfig(901, 0, 1000),
	})
	require.NoError(t, ingester1.initLogsDB())
	require.NoError(t, ingester1.sealParentBlock(99))
	require.NoError(t, ingester1.logsDB.Close())

	// Second ingester: resume from a DB containing only the anchor.
	ingester2 := newTestLogsDBChainIngester(t, testIngesterConfig{
		chainID:   chainID,
		dataDir:   tempDir,
		ethClient: mockClient,
		rollupCfg: testRollupConfig(901, 0, 1000),
	})
	require.NoError(t, ingester2.initLogsDB())
	t.Cleanup(func() { ingester2.logsDB.Close() })

	nextBlock, err := ingester2.initIngestion()
	require.NoError(t, err)
	require.Equal(t, uint64(100), nextBlock,
		"resume should pick up immediately after the anchor")

	require.False(t, ingester2.earliestIngestedBlockSet.Load(),
		"with only the anchor sealed, earliest must remain unset until first ingestion")

	// Ingesting the first real block then sets earliest, matching the fresh-start path.
	require.NoError(t, ingester2.ingestBlock(100))
	require.True(t, ingester2.earliestIngestedBlockSet.Load())
	require.Equal(t, uint64(100), ingester2.earliestIngestedBlock.Load())
}

func TestLogsDBChainIngester_InitIngestion_ErrorGettingHead(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(901)
	tempDir := t.TempDir()

	mockClient := NewMockEthClient()
	mockClient.SetInfoByLabelErr(context.DeadlineExceeded)

	ingester := newTestLogsDBChainIngester(t, testIngesterConfig{
		chainID:   chainID,
		dataDir:   tempDir,
		ethClient: mockClient,
		rollupCfg: testRollupConfig(901, 0, 1000),
	})

	err := ingester.initLogsDB()
	require.NoError(t, err)
	t.Cleanup(func() { ingester.logsDB.Close() })

	// initIngestion should fail if it can't get head
	_, err = ingester.initIngestion()
	require.Error(t, err)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

// =============================================================================
// Error Injection Tests
// =============================================================================

func TestLogsDBChainIngester_IngestBlock_NonSequentialError(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(901)
	tempDir := t.TempDir()

	mockClient := NewMockEthClient()

	parentBlock := createTestBlock(99, 1198, common.Hash{})
	mockClient.AddBlock(parentBlock, nil)

	block100 := createTestBlock(100, 1200, parentBlock.Hash())
	mockClient.AddBlock(block100, createTestReceipts(100, 1))

	// Block 102 (skipping 101)
	block102 := createTestBlock(102, 1204, block100.Hash())
	mockClient.AddBlock(block102, createTestReceipts(102, 1))

	ingester := newTestLogsDBChainIngester(t, testIngesterConfig{
		chainID:   chainID,
		dataDir:   tempDir,
		ethClient: mockClient,
		rollupCfg: testRollupConfig(901, 0, 1000),
	})

	err := ingester.initLogsDB()
	require.NoError(t, err)
	t.Cleanup(func() { ingester.logsDB.Close() })

	err = ingester.sealParentBlock(99)
	require.NoError(t, err)

	err = ingester.ingestBlock(100)
	require.NoError(t, err)

	// Try to ingest block 102 (skipping 101) - should fail
	err = ingester.ingestBlock(102)
	require.Error(t, err)
	require.Contains(t, err.Error(), "expected to ingest block 101 but got 102")
}

func TestLogsDBChainIngester_IngestBlock_RPCError(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(901)
	tempDir := t.TempDir()

	mockClient := NewMockEthClient()

	parentBlock := createTestBlock(99, 1198, common.Hash{})
	mockClient.AddBlock(parentBlock, nil)

	ingester := newTestLogsDBChainIngester(t, testIngesterConfig{
		chainID:   chainID,
		dataDir:   tempDir,
		ethClient: mockClient,
		rollupCfg: testRollupConfig(901, 0, 1000),
	})

	err := ingester.initLogsDB()
	require.NoError(t, err)
	t.Cleanup(func() { ingester.logsDB.Close() })

	err = ingester.sealParentBlock(99)
	require.NoError(t, err)

	// Set RPC error - block 100 is not in mock, will fail
	err = ingester.ingestBlock(100)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to fetch block 100")
	require.Contains(t, err.Error(), "fetch block info")
}

func TestLogsDBChainIngester_IngestBlock_ReceiptsError(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(901)
	tempDir := t.TempDir()

	mockClient := NewMockEthClient()

	parentBlock := createTestBlock(99, 1198, common.Hash{})
	mockClient.AddBlock(parentBlock, nil)

	block100 := createTestBlock(100, 1200, parentBlock.Hash())
	mockClient.AddBlock(block100, nil) // No receipts added

	ingester := newTestLogsDBChainIngester(t, testIngesterConfig{
		chainID:   chainID,
		dataDir:   tempDir,
		ethClient: mockClient,
		rollupCfg: testRollupConfig(901, 0, 1000),
	})

	err := ingester.initLogsDB()
	require.NoError(t, err)
	t.Cleanup(func() { ingester.logsDB.Close() })

	err = ingester.sealParentBlock(99)
	require.NoError(t, err)

	// Set receipts error
	mockClient.SetFetchReceiptsErr(context.DeadlineExceeded)

	err = ingester.ingestBlock(100)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to fetch block 100")
	require.Contains(t, err.Error(), "fetch block receipts")
}

func TestLogsDBChainIngester_IngestBlock_ErrorStateSkipsIngestion(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(901)
	tempDir := t.TempDir()

	mockClient := NewMockEthClient()

	parentBlock := createTestBlock(99, 1198, common.Hash{})
	mockClient.AddBlock(parentBlock, nil)

	block100 := createTestBlock(100, 1200, parentBlock.Hash())
	mockClient.AddBlock(block100, createTestReceipts(100, 1))

	ingester := newTestLogsDBChainIngester(t, testIngesterConfig{
		chainID:   chainID,
		dataDir:   tempDir,
		ethClient: mockClient,
		rollupCfg: testRollupConfig(901, 0, 1000),
	})

	err := ingester.initLogsDB()
	require.NoError(t, err)
	t.Cleanup(func() { ingester.logsDB.Close() })

	err = ingester.sealParentBlock(99)
	require.NoError(t, err)

	// Set error state before ingestion
	ingester.SetError(ErrorReorg, "test error")

	// ingestBlock should return nil without doing anything
	err = ingester.ingestBlock(100)
	require.NoError(t, err)

	// Block 100 should NOT have been ingested
	latestBlock, ok := ingester.LatestBlock()
	require.True(t, ok)
	require.Equal(t, uint64(99), latestBlock.Number) // Still at parent block
}

func TestLogsDBChainIngester_ErrorTypes(t *testing.T) {
	require.Equal(t, "data_corruption", ErrorDataCorruption.String())
	require.Equal(t, "invalid_log", ErrorInvalidExecutingMessage.String())
}
