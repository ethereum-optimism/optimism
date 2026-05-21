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

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
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
	_, err := ingester.Contains(messages.ContainsQuery{})
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
	_, err = ingester.Contains(messages.ContainsQuery{
		Timestamp: 1200,
		BlockNum:  100,
		LogIdx:    99, // Doesn't exist
		Checksum:  messages.MessageChecksum{0xFF},
	})
	require.ErrorIs(t, err, types.ErrConflict)
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

func TestLogsDBChainIngester_ErrorTypes(t *testing.T) {
	require.Equal(t, "data_corruption", ErrorDataCorruption.String())
	require.Equal(t, "invalid_log", ErrorInvalidExecutingMessage.String())
}
