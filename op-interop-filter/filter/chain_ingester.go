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

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-interop-filter/metrics"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/processors"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/reads"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

const (
	defaultBlockTime = 2 * time.Second
	pollInterval     = 2 * time.Second
)

// ExecutingMessageCallback is called when executing messages are detected during ingestion
type ExecutingMessageCallback func(chainID eth.ChainID, timestamp uint64, execMsgs []*types.ExecutingMessage)

// ReorgCallback is called when a reorg is detected
type ReorgCallback func(chainID eth.ChainID)

// ChainIngester handles block ingestion and log storage for a single chain
type ChainIngester struct {
	log     log.Logger
	metrics metrics.Metricer
	chainID eth.ChainID

	rpcClient  client.RPC
	ethClient  *sources.EthClient
	logsDB     *logs.DB
	dataDir    string
	backfillDuration time.Duration

	ready   atomic.Bool
	stopped atomic.Bool

	onExecMsg ExecutingMessageCallback
	onReorg   ReorgCallback

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex // protects logsDB access during rewind
}

// NewChainIngester creates a new ChainIngester for the given chain
func NewChainIngester(
	parentCtx context.Context,
	logger log.Logger,
	m metrics.Metricer,
	chainID eth.ChainID,
	rpcURL string,
	dataDir string,
	backfillDuration time.Duration,
	onExecMsg ExecutingMessageCallback,
	onReorg ReorgCallback,
) (*ChainIngester, error) {
	ctx, cancel := context.WithCancel(parentCtx)

	logger = logger.New("chain", chainID)

	// Create RPC client
	rpcClient, err := client.NewRPC(ctx, logger, rpcURL)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create RPC client for chain %s: %w", chainID, err)
	}

	// Create eth client
	ethClient, err := sources.NewEthClient(
		rpcClient,
		logger,
		nil,    // metrics
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
		cancel()
		return nil, fmt.Errorf("failed to create eth client for chain %s: %w", chainID, err)
	}

	return &ChainIngester{
		log:              logger,
		metrics:          m,
		chainID:          chainID,
		rpcClient:        rpcClient,
		ethClient:        ethClient,
		dataDir:          dataDir,
		backfillDuration: backfillDuration,
		onExecMsg:        onExecMsg,
		onReorg:          onReorg,
		ctx:              ctx,
		cancel:           cancel,
	}, nil
}

// Start begins block ingestion
func (c *ChainIngester) Start() error {
	c.log.Info("Starting chain ingester")

	// Initialize LogsDB
	if err := c.initLogsDB(); err != nil {
		return fmt.Errorf("failed to init logs DB: %w", err)
	}

	// Start ingestion goroutine
	c.wg.Add(1)
	go c.runIngestion()

	return nil
}

// Stop gracefully stops the chain ingester
func (c *ChainIngester) Stop() error {
	if !c.stopped.CompareAndSwap(false, true) {
		return nil
	}
	c.log.Info("Stopping chain ingester")
	c.cancel()
	c.wg.Wait()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Close LogsDB
	if c.logsDB != nil {
		if err := c.logsDB.Close(); err != nil {
			return fmt.Errorf("failed to close logs DB: %w", err)
		}
	}

	// Close RPC clients
	if c.ethClient != nil {
		c.ethClient.Close()
	}
	if c.rpcClient != nil {
		c.rpcClient.Close()
	}

	return nil
}

// Ready returns true if backfill is complete
func (c *ChainIngester) Ready() bool {
	return c.ready.Load()
}

// ChainID returns the chain ID
func (c *ChainIngester) ChainID() eth.ChainID {
	return c.chainID
}

// Contains checks if a log exists in the database
func (c *ChainIngester) Contains(query types.ContainsQuery) (types.BlockSeal, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.logsDB == nil {
		return types.BlockSeal{}, types.ErrUninitialized
	}

	return c.logsDB.Contains(query)
}

// LatestBlock returns the latest sealed block
func (c *ChainIngester) LatestBlock() (eth.BlockID, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.logsDB == nil {
		return eth.BlockID{}, false
	}

	return c.logsDB.LatestSealedBlock()
}

// LatestTimestamp returns the timestamp of the latest sealed block
func (c *ChainIngester) LatestTimestamp() (uint64, bool) {
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

// initLogsDB initializes the logs database
func (c *ChainIngester) initLogsDB() error {
	var dbPath string
	if c.dataDir != "" {
		chainDir := filepath.Join(c.dataDir, fmt.Sprintf("chain-%s", c.chainID))
		if err := os.MkdirAll(chainDir, 0755); err != nil {
			return fmt.Errorf("failed to create chain directory: %w", err)
		}
		dbPath = filepath.Join(chainDir, "logs.db")
	} else {
		// Use temp directory if no data dir specified
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

// runIngestion runs the main ingestion loop
func (c *ChainIngester) runIngestion() {
	defer c.wg.Done()

	// Get current head
	head, err := c.ethClient.InfoByLabel(c.ctx, eth.Unsafe)
	if err != nil {
		c.log.Error("Failed to get current head", "err", err)
		return
	}
	c.log.Info("Current chain head", "block", head.NumberU64(), "hash", head.Hash())

	// Calculate backfill start block
	blocksToBackfill := uint64(c.backfillDuration / defaultBlockTime)
	var startBlock uint64
	if head.NumberU64() > blocksToBackfill {
		startBlock = head.NumberU64() - blocksToBackfill
	} else {
		startBlock = 1 // Start from block 1, not genesis
	}

	c.log.Info("Starting backfill", "from", startBlock, "to", head.NumberU64(), "blocks", head.NumberU64()-startBlock+1)

	// Initialize the LogsDB with the parent block of the start block
	// This is needed because the LogsDB requires a sealed parent block before adding logs
	if startBlock > 0 {
		if err := c.initializeAnchorBlock(startBlock - 1); err != nil {
			c.log.Error("Failed to initialize anchor block", "err", err)
			return
		}
	}

	// Backfill
	if err := c.backfill(startBlock, head.NumberU64()); err != nil {
		if errors.Is(err, context.Canceled) {
			c.log.Info("Backfill canceled")
			return
		}
		c.log.Error("Backfill failed", "err", err)
		c.triggerReorg()
		return
	}

	// Mark as ready
	c.ready.Store(true)
	c.log.Info("Backfill complete, starting live ingestion")

	// Live polling loop
	c.pollLoop()
}

// initializeAnchorBlock seals the anchor block in the LogsDB
// This must be done before any logs can be added, as logs reference their parent block
func (c *ChainIngester) initializeAnchorBlock(blockNum uint64) error {
	c.log.Info("Initializing anchor block", "block", blockNum)

	// Fetch the anchor block info
	blockInfo, err := c.ethClient.InfoByNumber(c.ctx, blockNum)
	if err != nil {
		return fmt.Errorf("failed to get anchor block info: %w", err)
	}

	blockID := eth.BlockID{Hash: blockInfo.Hash(), Number: blockInfo.NumberU64()}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Seal the anchor block with no logs
	// For the very first block, use zero hash as parent
	parentHash := blockInfo.ParentHash()
	if err := c.logsDB.SealBlock(parentHash, blockID, blockInfo.Time()); err != nil {
		return fmt.Errorf("failed to seal anchor block: %w", err)
	}

	c.log.Info("Initialized anchor block", "block", blockNum, "hash", blockID.Hash)
	return nil
}

// backfill ingests blocks from startBlock to endBlock
func (c *ChainIngester) backfill(startBlock, endBlock uint64) error {
	totalBlocks := endBlock - startBlock + 1
	lastProgress := 0
	lastLog := time.Now()

	for blockNum := startBlock; blockNum <= endBlock; blockNum++ {
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		default:
		}

		if err := c.ingestBlock(blockNum); err != nil {
			return fmt.Errorf("failed to ingest block %d: %w", blockNum, err)
		}

		// Progress reporting
		progress := int((blockNum - startBlock + 1) * 100 / totalBlocks)
		if progress >= lastProgress+10 || time.Since(lastLog) > 10*time.Second {
			c.log.Info("Backfill progress",
				"block", blockNum,
				"total", endBlock,
				"progress", fmt.Sprintf("%d%%", progress))
			lastProgress = progress
			lastLog = time.Now()
			// Record backfill progress metric
			chainIDUint64, _ := c.chainID.Uint64()
			c.metrics.RecordBackfillProgress(chainIDUint64, float64(progress)/100.0)
		}
	}

	return nil
}

// pollLoop polls for new blocks
func (c *ChainIngester) pollLoop() {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if err := c.pollNewBlocks(); err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				c.log.Error("Failed to poll new blocks", "err", err)
				// Don't trigger failsafe on transient errors
			}
		}
	}
}

// pollNewBlocks checks for and ingests new blocks
func (c *ChainIngester) pollNewBlocks() error {
	head, err := c.ethClient.InfoByLabel(c.ctx, eth.Unsafe)
	if err != nil {
		return fmt.Errorf("failed to get head: %w", err)
	}

	latestBlock, ok := c.LatestBlock()
	if !ok {
		return fmt.Errorf("no latest block in DB")
	}

	// Check for reorg
	if head.NumberU64() < latestBlock.Number {
		c.log.Warn("Detected reorg: head is behind latest block",
			"head", head.NumberU64(),
			"latest", latestBlock.Number)
		c.triggerReorg()
		return nil
	}

	// Ingest any missing blocks
	for blockNum := latestBlock.Number + 1; blockNum <= head.NumberU64(); blockNum++ {
		if err := c.ingestBlock(blockNum); err != nil {
			return fmt.Errorf("failed to ingest block %d: %w", blockNum, err)
		}
	}

	return nil
}

// ingestBlock ingests a single block
func (c *ChainIngester) ingestBlock(blockNum uint64) error {
	// Fetch block info
	blockInfo, err := c.ethClient.InfoByNumber(c.ctx, blockNum)
	if err != nil {
		return fmt.Errorf("failed to get block info: %w", err)
	}

	// Construct block ID
	blockID := eth.BlockID{Hash: blockInfo.Hash(), Number: blockInfo.NumberU64()}

	// Fetch receipts
	_, receipts, err := c.ethClient.FetchReceipts(c.ctx, blockInfo.Hash())
	if err != nil {
		return fmt.Errorf("failed to get receipts: %w", err)
	}

	// Check for reorg (parent hash mismatch)
	c.mu.RLock()
	latestBlock, hasLatest := c.logsDB.LatestSealedBlock()
	c.mu.RUnlock()

	if hasLatest && blockNum == latestBlock.Number+1 {
		if blockInfo.ParentHash() != latestBlock.Hash {
			c.log.Warn("Detected reorg: parent hash mismatch",
				"block", blockNum,
				"expected_parent", latestBlock.Hash,
				"actual_parent", blockInfo.ParentHash())
			c.triggerReorg()
			return nil
		}
	}

	// Process logs and add to DB
	var execMsgs []*types.ExecutingMessage
	var logIndex uint32

	c.mu.Lock()
	defer c.mu.Unlock()

	// Get parent block ID for AddLog
	parentBlock := eth.BlockID{Hash: blockInfo.ParentHash(), Number: blockNum - 1}
	if blockNum == 0 {
		parentBlock = eth.BlockID{}
	}

	for _, receipt := range receipts {
		for _, l := range receipt.Logs {
			// Compute log hash
			logHash := processors.LogToLogHash(l)

			// Check if this is an executing message
			execMsg, err := processors.DecodeExecutingMessageLog(l)
			if err != nil {
				c.log.Debug("Failed to decode executing message", "err", err)
			}

			// Add log to DB
			if err := c.logsDB.AddLog(logHash, parentBlock, logIndex, execMsg); err != nil {
				// Check for conflict (reorg indicator)
				if errors.Is(err, types.ErrConflict) {
					c.log.Warn("Conflict adding log, triggering reorg", "err", err)
					c.mu.Unlock()
					c.triggerReorg()
					c.mu.Lock()
					return nil
				}
				return fmt.Errorf("failed to add log: %w", err)
			}

			if execMsg != nil {
				execMsgs = append(execMsgs, execMsg)
			}
			logIndex++
		}
	}

	// Seal the block
	if err := c.logsDB.SealBlock(blockInfo.ParentHash(), blockID, blockInfo.Time()); err != nil {
		if errors.Is(err, types.ErrConflict) {
			c.log.Warn("Conflict sealing block, triggering reorg", "err", err)
			c.mu.Unlock()
			c.triggerReorg()
			c.mu.Lock()
			return nil
		}
		return fmt.Errorf("failed to seal block: %w", err)
	}

	// Update metrics
	chainIDUint64, _ := c.chainID.Uint64()
	c.metrics.RecordChainHead(chainIDUint64, blockNum)
	c.metrics.RecordBlocksSealed(chainIDUint64, 1)
	c.metrics.RecordLogsAdded(chainIDUint64, int64(logIndex))

	// Notify about executing messages
	// Release lock before callback to avoid deadlock - the callback may call
	// back into this ingester (e.g., LatestTimestamp) which needs the lock
	if len(execMsgs) > 0 && c.onExecMsg != nil {
		c.mu.Unlock()
		c.onExecMsg(c.chainID, blockInfo.Time(), execMsgs)
		c.mu.Lock()
	}

	return nil
}

// triggerReorg handles reorg detection
func (c *ChainIngester) triggerReorg() {
	c.log.Warn("Reorg detected, triggering failsafe")
	chainIDUint64, _ := c.chainID.Uint64()
	c.metrics.RecordReorgDetected(chainIDUint64)
	if c.onReorg != nil {
		c.onReorg(c.chainID)
	}
}

// Rewind rewinds the chain to the specified block
func (c *ChainIngester) Rewind(newHead eth.BlockID) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.logsDB == nil {
		return types.ErrUninitialized
	}

	// Use a no-op invalidator since we don't have cross-safe tracking
	inv := &noopInvalidator{}

	return c.logsDB.Rewind(inv, newHead)
}

// logsDBMetrics implements the logs.Metrics interface
type logsDBMetrics struct {
	m       metrics.Metricer
	chainID eth.ChainID
}

func (l *logsDBMetrics) RecordDBEntryCount(kind string, count int64) {
	// Could add more detailed metrics here if needed
}

func (l *logsDBMetrics) RecordDBSearchEntriesRead(count int64) {
	// Could add more detailed metrics here if needed
}

// noopInvalidator is a no-op implementation of reads.Invalidator
type noopInvalidator struct{}

func (n *noopInvalidator) TryInvalidate(inv reads.InvalidationRule) (func(), error) {
	return func() {}, nil
}
