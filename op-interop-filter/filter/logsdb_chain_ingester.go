package filter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-interop-filter/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/processors"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// progressLogInterval is how often to log ingestion progress.
const progressLogInterval = 10 * time.Second

// EthClient defines the interface for fetching block and receipt data.
// This allows for dependency injection in tests.
type EthClient interface {
	InfoByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, error)
	InfoByNumber(ctx context.Context, number uint64) (eth.BlockInfo, error)
	FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, gethTypes.Receipts, error)
	Close()
}

// LogsDBChainIngester handles block ingestion and log storage for a single chain.
// It uses an RPC client to fetch blocks and a logsdb database for storage.
type LogsDBChainIngester struct {
	log     log.Logger
	metrics metrics.Metricer
	chainID eth.ChainID

	rpcClient        client.RPC
	ethClient        EthClient
	logsDB           *logs.DB
	dataDir          string
	startTimestamp   uint64        // Timestamp at which we report Ready (typically now)
	backfillDuration time.Duration // How far back to start ingestion from startTimestamp
	pollInterval     time.Duration
	rollupCfg        *rollup.Config // Rollup config for block number calculation

	stopped atomic.Bool

	errorState atomic.Pointer[IngesterError]

	earliestIngestedBlock    atomic.Uint64
	earliestIngestedBlockSet atomic.Bool

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex
}

// NewLogsDBChainIngester creates a new LogsDBChainIngester for the given chain.
// startTimestamp is when we report Ready() = true (typically now).
// backfillDuration is how far back from startTimestamp to begin ingestion.
func NewLogsDBChainIngester(
	parentCtx context.Context,
	logger log.Logger,
	m metrics.Metricer,
	chainID eth.ChainID,
	rpcURL string,
	dataDir string,
	startTimestamp uint64,
	backfillDuration time.Duration,
	pollInterval time.Duration,
	rollupCfg *rollup.Config,
) (*LogsDBChainIngester, error) {
	ctx, cancel := context.WithCancel(parentCtx)

	logger = logger.New("chain", chainID)

	rpcClient, err := client.NewRPC(ctx, logger, rpcURL)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create RPC client for chain %s: %w", chainID, err)
	}

	ethClient, err := sources.NewEthClient(
		rpcClient,
		logger,
		nil,
		&sources.EthClientConfig{
			ReceiptsCacheSize:     1000,
			TransactionsCacheSize: 1000,
			HeadersCacheSize:      1000,
			PayloadsCacheSize:     100,
			MaxRequestsPerBatch:   20,
			MaxConcurrentRequests: 10,
			TrustRPC:              false,
			MustBePostMerge:       true,
			RPCProviderKind:       sources.RPCKindStandard,
		},
	)
	if err != nil {
		rpcClient.Close()
		cancel()
		return nil, fmt.Errorf("failed to create eth client for chain %s: %w", chainID, err)
	}

	return &LogsDBChainIngester{
		log:              logger,
		metrics:          m,
		chainID:          chainID,
		rpcClient:        rpcClient,
		ethClient:        ethClient,
		dataDir:          dataDir,
		startTimestamp:   startTimestamp,
		backfillDuration: backfillDuration,
		pollInterval:     pollInterval,
		rollupCfg:        rollupCfg,
		ctx:              ctx,
		cancel:           cancel,
	}, nil
}

// Start begins block ingestion
func (c *LogsDBChainIngester) Start() error {
	c.log.Info("Starting chain ingester")

	if err := c.initLogsDB(); err != nil {
		return fmt.Errorf("failed to init logs DB: %w", err)
	}

	c.wg.Add(1)
	go c.runIngestion()

	return nil
}

// Stop gracefully stops the chain ingester
func (c *LogsDBChainIngester) Stop() error {
	if !c.stopped.CompareAndSwap(false, true) {
		return nil
	}
	c.log.Info("Stopping chain ingester")
	c.cancel()
	c.wg.Wait()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.logsDB != nil {
		if err := c.logsDB.Close(); err != nil {
			return fmt.Errorf("failed to close logs DB: %w", err)
		}
	}

	if c.ethClient != nil {
		c.ethClient.Close()
	}
	if c.rpcClient != nil {
		c.rpcClient.Close()
	}

	return nil
}

// Ready returns true if we've ingested up to at least the start timestamp.
func (c *LogsDBChainIngester) Ready() bool {
	latestTs, ok := c.LatestTimestamp()
	if !ok {
		return false
	}
	return latestTs >= c.startTimestamp
}

// SetError sets the error state, logs the error, and records metrics.
func (c *LogsDBChainIngester) SetError(reason IngesterErrorReason, msg string) {
	err := &IngesterError{
		Reason:  reason,
		Message: msg,
	}
	c.errorState.Store(err)
	c.log.Error("Ingester halted", "reason", reason.String(), "msg", msg)

	chainIDUint64, _ := c.chainID.Uint64()
	if reason == ErrorReorg || reason == ErrorConflict {
		c.metrics.RecordReorgDetected(chainIDUint64)
	}
}

// Error returns the current error state, or nil if no error.
func (c *LogsDBChainIngester) Error() *IngesterError {
	return c.errorState.Load()
}

// ClearError clears the error state.
func (c *LogsDBChainIngester) ClearError() {
	c.errorState.Store(nil)
	c.log.Info("Ingester error state cleared")
}

// Contains checks if a log exists in the database
func (c *LogsDBChainIngester) Contains(query types.ContainsQuery) (types.BlockSeal, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.logsDB == nil {
		return types.BlockSeal{}, types.ErrUninitialized
	}

	return c.logsDB.Contains(query)
}

// LatestBlock returns the latest sealed block
func (c *LogsDBChainIngester) LatestBlock() (eth.BlockID, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.logsDB == nil {
		return eth.BlockID{}, false
	}

	return c.logsDB.LatestSealedBlock()
}

// BlockHashAt returns the hash of the sealed block at the given height.
func (c *LogsDBChainIngester) BlockHashAt(blockNum uint64) (common.Hash, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.logsDB == nil {
		return common.Hash{}, false
	}

	seal, err := c.logsDB.FindSealedBlock(blockNum)
	if err != nil {
		return common.Hash{}, false
	}

	return seal.Hash, true
}

// LatestTimestamp returns the timestamp of the latest sealed block
func (c *LogsDBChainIngester) LatestTimestamp() (uint64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.logsDB == nil {
		return 0, false
	}

	latestBlock, ok := c.logsDB.LatestSealedBlock()
	if !ok {
		return 0, false
	}

	seal, err := c.logsDB.FindSealedBlock(latestBlock.Number)
	if err != nil {
		return 0, false
	}

	return seal.Timestamp, true
}

// GetExecMsgsAtTimestamp returns executing messages with the given inclusion timestamp.
func (c *LogsDBChainIngester) GetExecMsgsAtTimestamp(timestamp uint64) ([]IncludedMessage, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.logsDB == nil {
		return nil, types.ErrUninitialized
	}

	latestBlock, ok := c.logsDB.LatestSealedBlock()
	if !c.earliestIngestedBlockSet.Load() || !ok {
		return nil, nil
	}
	earliest := c.earliestIngestedBlock.Load()

	var results []IncludedMessage
	for blockNum := earliest; blockNum <= latestBlock.Number; blockNum++ {
		ref, _, execMsgs, err := c.logsDB.OpenBlock(blockNum)
		if err != nil {
			return nil, fmt.Errorf("failed to open block %d: %w", blockNum, err)
		}

		if ref.Time == timestamp {
			for _, msg := range execMsgs {
				results = append(results, IncludedMessage{
					ExecutingMessage:   msg,
					InclusionBlockNum:  blockNum,
					InclusionTimestamp: ref.Time,
				})
			}
		}

		// Timestamps increase, so we can stop early
		if ref.Time > timestamp {
			break
		}
	}

	return results, nil
}

func (c *LogsDBChainIngester) findAndSetEarliestBlock(latestBlock uint64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Walk backward from latest to find the first block that can be opened.
	// The anchor block (sealed but not fully ingested) will fail OpenBlock
	// because it has no predecessor checkpoint data, so we'll identify
	// the first block with actual log data.
	earliest := latestBlock
	for blockNum := latestBlock; blockNum > 0; blockNum-- {
		_, _, _, err := c.logsDB.OpenBlock(blockNum)
		if err != nil {
			// This block can't be opened, the one after it is earliest queryable
			earliest = blockNum + 1
			break
		}
		earliest = blockNum
	}

	c.earliestIngestedBlock.Store(earliest)
	c.earliestIngestedBlockSet.Store(true)
	c.log.Info("Found earliest block in DB", "block", earliest)
}

// calculateStartingBlock returns the block number where ingestion should start,
// calculated from startTimestamp and backfillDuration.
func (c *LogsDBChainIngester) calculateStartingBlock() uint64 {
	backfillTimestamp := c.startTimestamp - uint64(c.backfillDuration.Seconds())

	startingBlock, err := c.rollupCfg.TargetBlockNumber(backfillTimestamp)
	if err != nil {
		// Timestamp is before genesis, start from genesis block
		return c.rollupCfg.Genesis.L2.Number
	}
	return startingBlock
}

func (c *LogsDBChainIngester) initLogsDB() error {
	var dbPath string
	if c.dataDir != "" {
		chainDir := filepath.Join(c.dataDir, fmt.Sprintf("chain-%s", c.chainID))
		if err := os.MkdirAll(chainDir, 0755); err != nil {
			return fmt.Errorf("failed to create chain directory: %w", err)
		}
		dbPath = filepath.Join(chainDir, "logs.db")
	} else {
		tempDir, err := os.MkdirTemp("", fmt.Sprintf("interop-filter-chain-%s-*", c.chainID))
		if err != nil {
			return fmt.Errorf("failed to create temp directory: %w", err)
		}
		dbPath = filepath.Join(tempDir, "logs.db")
		c.log.Warn("Using temporary directory for logs DB", "path", dbPath)
	}

	db, err := logs.NewFromFile(c.log, &logsDBMetrics{m: c.metrics, chainID: c.chainID}, c.chainID, dbPath, true)
	if err != nil {
		return fmt.Errorf("failed to open logs DB: %w", err)
	}

	c.mu.Lock()
	c.logsDB = db
	c.mu.Unlock()

	c.log.Info("Initialized logs DB", "path", dbPath)
	return nil
}

func (c *LogsDBChainIngester) runIngestion() {
	defer c.wg.Done()

	// One-time setup: determine starting block and next block to ingest
	nextBlock, err := c.initIngestion()
	if err != nil {
		// Application context was canceled (e.g., during shutdown).
		if errors.Is(err, context.Canceled) {
			c.log.Info("Ingestion init canceled")
			return
		}
		c.log.Error("Failed to initialize ingestion", "err", err)
		return
	}

	// Track progress for logging
	lastLogTime := clock.SystemClock.Now()

	// Use ticker for polling interval
	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()

	// Unified ingestion loop - no concept of "backfill" vs "live"
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
		}

		// Skip if in error state
		if c.Error() != nil {
			continue
		}

		head, err := c.ethClient.InfoByLabel(c.ctx, eth.Unsafe)
		if err != nil {
			// Application context was canceled (e.g., during shutdown).
			if errors.Is(err, context.Canceled) {
				return
			}
			c.log.Error("Failed to get head", "err", err)
			continue
		}

		// Reorg detection: if head moved behind our progress, check hash
		if head.NumberU64() < nextBlock-1 {
			if err := c.checkReorg(head); err != nil {
				continue
			}
		}

		// Inner loop: ingest all available blocks without waiting between them
		for nextBlock <= head.NumberU64() {
			// Check for shutdown between blocks
			select {
			case <-c.ctx.Done():
				return
			default:
			}

			if err := c.ingestBlock(nextBlock); err != nil {
				// Application context was canceled (e.g., during shutdown).
				if errors.Is(err, context.Canceled) {
					return
				}
				c.log.Error("Failed to ingest block", "block", nextBlock, "err", err)
				break // Exit inner loop on error, wait for next tick to retry
			}
			nextBlock++

			// Progress logging
			if clock.SystemClock.Since(lastLogTime) > progressLogInterval {
				startingBlock := c.calculateStartingBlock()
				if nextBlock <= startingBlock {
					progress := float64(nextBlock-c.earliestIngestedBlock.Load()) / float64(startingBlock-c.earliestIngestedBlock.Load()+1)
					c.log.Info("Ingestion progress",
						"block", nextBlock-1,
						"target", startingBlock,
						"progress", fmt.Sprintf("%.0f%%", progress*100))
					chainIDUint64, _ := c.chainID.Uint64()
					c.metrics.RecordBackfillProgress(chainIDUint64, progress)
				} else {
					c.log.Debug("Ingestion progress", "block", nextBlock-1, "head", head.NumberU64())
				}
				lastLogTime = clock.SystemClock.Now()
			}
		}
		// Caught up to head, will wait for next ticker tick
	}
}

// initIngestion performs one-time setup and returns the first block to ingest.
func (c *LogsDBChainIngester) initIngestion() (uint64, error) {
	head, err := c.ethClient.InfoByLabel(c.ctx, eth.Unsafe)
	if err != nil {
		return 0, fmt.Errorf("failed to get current head: %w", err)
	}
	c.log.Info("Current chain head", "block", head.NumberU64(), "hash", head.Hash())

	startingBlock := c.calculateStartingBlock()

	// Clamp to head if needed
	if startingBlock > head.NumberU64() {
		startingBlock = head.NumberU64()
	}

	// Guard against underflow: genesis block has no parent to seal
	if startingBlock == 0 {
		startingBlock = 1
		c.log.Info("Starting from block 1 (genesis has no parent to seal)")
	}

	c.log.Info("Determined starting block", "block", startingBlock, "startTimestamp", c.startTimestamp)

	// Check if we have existing data to resume from
	c.mu.RLock()
	latestSealed, hasSealed := c.logsDB.LatestSealedBlock()
	c.mu.RUnlock()

	if hasSealed {
		// Resume from existing DB
		nextBlock := latestSealed.Number + 1
		c.log.Info("Resuming from existing DB", "lastSealed", latestSealed.Number, "resumeFrom", nextBlock)

		if !c.earliestIngestedBlockSet.Load() {
			c.findAndSetEarliestBlock(latestSealed.Number)
		}

		return nextBlock, nil
	}

	// Fresh start: seal parent block as anchor
	if err := c.sealParentBlock(startingBlock - 1); err != nil {
		return 0, fmt.Errorf("failed to seal parent block: %w", err)
	}

	c.log.Info("Starting fresh ingestion",
		"from", startingBlock,
		"to", head.NumberU64(),
		"blocks", head.NumberU64()-startingBlock+1)

	return startingBlock, nil
}

// checkReorg checks if a reorg occurred when head moves behind our progress.
func (c *LogsDBChainIngester) checkReorg(head eth.BlockInfo) error {
	headNum := head.NumberU64()

	// If head is before our earliest block, we can't verify - this is expected
	if c.earliestIngestedBlockSet.Load() && headNum < c.earliestIngestedBlock.Load() {
		c.log.Debug("Head before our earliest block, can't verify",
			"head", headNum, "earliest", c.earliestIngestedBlock.Load())
		return nil
	}

	dbHash, ok := c.BlockHashAt(headNum)
	if !ok {
		// We should have this block but can't get it - unexpected
		c.log.Error("Failed to get block hash for reorg verification", "block", headNum)
		return nil
	}

	if dbHash == head.Hash() {
		c.log.Debug("Head temporarily behind, same hash - skipping", "head", headNum)
		return nil
	}

	// Hash mismatch = reorg
	c.log.Warn("Detected reorg: different block at same height",
		"height", headNum, "db_hash", dbHash, "head_hash", head.Hash())
	c.SetError(ErrorReorg, fmt.Sprintf("reorg at height %d: db has %s, chain has %s",
		headNum, dbHash, head.Hash()))
	return fmt.Errorf("reorg detected")
}

func (c *LogsDBChainIngester) sealParentBlock(blockNum uint64) error {
	c.log.Info("Sealing parent block as starting point", "block", blockNum)

	blockInfo, err := c.ethClient.InfoByNumber(c.ctx, blockNum)
	if err != nil {
		return fmt.Errorf("failed to get block info: %w", err)
	}

	blockID := eth.BlockID{Hash: blockInfo.Hash(), Number: blockInfo.NumberU64()}

	c.mu.Lock()
	defer c.mu.Unlock()

	parentHash := blockInfo.ParentHash()
	if err := c.logsDB.SealBlock(parentHash, blockID, blockInfo.Time()); err != nil {
		return fmt.Errorf("failed to seal block: %w", err)
	}

	// Note: We don't set earliestIngestedBlock here because the parent block is just
	// an anchor checkpoint. earliestIngestedBlock will be set in ingestBlock when
	// the first block with actual log data is ingested.

	c.log.Info("Sealed parent block", "block", blockNum, "hash", blockID.Hash)
	return nil
}

func (c *LogsDBChainIngester) ingestBlock(blockNum uint64) error {
	if c.Error() != nil {
		return nil
	}

	blockInfo, err := c.ethClient.InfoByNumber(c.ctx, blockNum)
	if err != nil {
		return fmt.Errorf("failed to get block info: %w", err)
	}

	blockID := eth.BlockID{Hash: blockInfo.Hash(), Number: blockInfo.NumberU64()}

	_, receipts, err := c.ethClient.FetchReceipts(c.ctx, blockInfo.Hash())
	if err != nil {
		return fmt.Errorf("failed to get receipts: %w", err)
	}

	c.mu.RLock()
	latestBlock, hasLatest := c.logsDB.LatestSealedBlock()
	c.mu.RUnlock()

	// Always verify parent hash when we have a previous block
	if hasLatest {
		// We should always be ingesting the next sequential block
		if blockNum != latestBlock.Number+1 {
			return fmt.Errorf("expected to ingest block %d but got %d", latestBlock.Number+1, blockNum)
		}
		// Parent hash of new block must match our latest sealed block
		if blockInfo.ParentHash() != latestBlock.Hash {
			c.log.Warn("Detected reorg: parent hash mismatch",
				"block", blockNum,
				"expected_parent", latestBlock.Hash,
				"actual_parent", blockInfo.ParentHash())
			c.SetError(ErrorReorg, fmt.Sprintf("parent hash mismatch at block %d", blockNum))
			return nil
		}
	}

	logCount, err := c.processBlockLogs(blockInfo, blockID, receipts, blockNum)
	if err != nil {
		if errors.Is(err, types.ErrConflict) {
			c.SetError(ErrorConflict, fmt.Sprintf("database conflict at block %d", blockNum))
			return nil
		}
		if errors.Is(err, types.ErrDataCorruption) {
			c.SetError(ErrorDataCorruption, fmt.Sprintf("data corruption at block %d: %v", blockNum, err))
			return nil
		}
		if errors.Is(err, ErrInvalidLog) {
			c.SetError(ErrorInvalidExecutingMessage, fmt.Sprintf("invalid log at block %d: %v", blockNum, err))
			return nil
		}
		return err
	}

	chainIDUint64, _ := c.chainID.Uint64()
	c.metrics.RecordChainHead(chainIDUint64, blockNum)
	c.metrics.RecordBlocksSealed(chainIDUint64, 1)
	c.metrics.RecordLogsAdded(chainIDUint64, int64(logCount))

	// Set earliest block on first successful ingestion (fresh start case).
	// On restart, findAndSetEarliestBlock handles this instead.
	if !c.earliestIngestedBlockSet.Load() {
		c.earliestIngestedBlock.Store(blockNum)
		c.earliestIngestedBlockSet.Store(true)
	}

	return nil
}

func (c *LogsDBChainIngester) processBlockLogs(blockInfo eth.BlockInfo, blockID eth.BlockID,
	receipts gethTypes.Receipts, blockNum uint64) (uint32, error) {

	c.mu.Lock()
	defer c.mu.Unlock()

	var logIndex uint32

	parentBlock := eth.BlockID{Hash: blockInfo.ParentHash(), Number: blockNum - 1}
	if blockNum == 0 {
		parentBlock = eth.BlockID{}
	}

	for _, receipt := range receipts {
		for _, l := range receipt.Logs {
			logHash := processors.LogToLogHash(l)

			execMsg, err := processors.DecodeExecutingMessageLog(l)
			if err != nil {
				return 0, fmt.Errorf("invalid log %d in block %d: %w: %w", l.Index, blockNum, ErrInvalidLog, err)
			}

			if err := c.logsDB.AddLog(logHash, parentBlock, logIndex, execMsg); err != nil {
				return 0, fmt.Errorf("failed to add log: %w", err)
			}
			logIndex++
		}
	}

	if err := c.logsDB.SealBlock(blockInfo.ParentHash(), blockID, blockInfo.Time()); err != nil {
		return 0, fmt.Errorf("failed to seal block: %w", err)
	}

	return logIndex, nil
}

// logsDBMetrics implements the logs.Metrics interface
type logsDBMetrics struct {
	m       metrics.Metricer
	chainID eth.ChainID
}

func (l *logsDBMetrics) RecordDBEntryCount(kind string, count int64) {}

func (l *logsDBMetrics) RecordDBSearchEntriesRead(count int64) {}

// Ensure LogsDBChainIngester implements ChainIngester
var _ ChainIngester = (*LogsDBChainIngester)(nil)
