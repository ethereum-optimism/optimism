package bgpo

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/sources/caching"
)

// BTOBackend is the interface for the blob tip oracle to interact with the L1 chain.
// ethclient.Client implements this interface.
type BTOBackend interface {
	BlockNumber(ctx context.Context) (uint64, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
}

// BlobTipOracle tracks blob base gas prices by subscribing to new block headers
// and extracts the blob tip caps from blob txs from each block.
type BlobTipOracle struct {
	sync.Mutex

	backend     BTOBackend
	chainConfig *params.ChainConfig
	log         log.Logger
	config      *BlobTipOracleConfig

	// LRU cache for blob base fees by block number
	baseFees *caching.LRUCache[uint64, *big.Int]

	// Cache for blob txs priority fees extracted from blocks (for SuggestBlobTipCap)
	priorityFees *caching.LRUCache[uint64, []*big.Int]

	// Track the latest block number for GetLatestBlobBaseFee
	latestBlock uint64

	ctx    context.Context
	cancel context.CancelFunc

	loopDone chan struct{}
}

// BlobTipOracleConfig configures the blob tip oracle.
type BlobTipOracleConfig struct {
	// NetworkTimeout is the timeout for network requests
	NetworkTimeout time.Duration
	// PricesCacheSize is the maximum number of blob base fees to cache
	PricesCacheSize int
	// BlockCacheSize is the maximum number of blocks to cache for RPC calls
	BlockCacheSize int
	// MaxBlocks is the default number of recent blocks to analyze in SuggestBlobTipCap
	MaxBlocks int
	// Percentile is the default percentile to use for blob tip cap suggestion
	Percentile int
	// Poll rate is the rate at which the oracle will poll for new blocks
	PollRate time.Duration
	// Metrics for cache tracking
	Metrics caching.Metrics
	// DefaultPriorityFee is the default priority fee to use for blob tip cap suggestion, if there are no recent blob txs
	DefaultPriorityFee *big.Int
}

// DefaultBlobTipOracleConfig returns a default configuration.
func DefaultBlobTipOracleConfig() *BlobTipOracleConfig {
	return &BlobTipOracleConfig{
		PricesCacheSize:    1000,
		BlockCacheSize:     100,
		MaxBlocks:          20,
		Percentile:         60,
		PollRate:           2500 * time.Millisecond,
		NetworkTimeout:     3 * time.Second,
		Metrics:            nil,
		DefaultPriorityFee: big.NewInt(1), // 1 wei
	}
}

// NewBlobTipOracle creates a new blob tip oracle that will subscribe
// to newHeads and track blob base fees, and extract blob tip caps from blob txs.
func NewBlobTipOracle(backend BTOBackend, chainConfig *params.ChainConfig, log log.Logger, config *BlobTipOracleConfig) *BlobTipOracle {
	defaultConfig := DefaultBlobTipOracleConfig()
	if config == nil {
		config = defaultConfig
	}
	if config.PricesCacheSize <= 0 {
		config.PricesCacheSize = defaultConfig.PricesCacheSize
	}
	if config.BlockCacheSize <= 0 {
		config.BlockCacheSize = defaultConfig.BlockCacheSize
	}
	if config.MaxBlocks <= 0 {
		config.MaxBlocks = defaultConfig.MaxBlocks
	}
	if config.Percentile <= 0 || config.Percentile > 100 {
		config.Percentile = defaultConfig.Percentile
	}

	oracleCtx, cancel := context.WithCancel(context.Background())
	return &BlobTipOracle{
		config:       config,
		backend:      backend,
		chainConfig:  chainConfig,
		log:          log.With("module", "bgpo"),
		baseFees:     caching.NewLRUCache[uint64, *big.Int](config.Metrics, "bgpo_prices", config.PricesCacheSize),
		priorityFees: caching.NewLRUCache[uint64, []*big.Int](config.Metrics, "bgpo_tips", config.BlockCacheSize),
		ctx:          oracleCtx,
		cancel:       cancel,
	}
}

// Start starts the oracle's background processing. It returns after the cache is prepopulated and
// the polling loop is started. To stop the background processing, call [BlobTipOracle.Close].
func (o *BlobTipOracle) Start() error {
	// Pre-populate cache with recent blocks before starting the polling loop
	if err := o.prePopulateCache(); err != nil {
		o.log.Warn("Failed to pre-populate cache, continuing anyway", "err", err)
	}

	o.log.Info("Blob tip oracle started, polling for new headers")

	o.loopDone = make(chan struct{})
	go o.pollLoop()
	return nil
}

func (o *BlobTipOracle) pollLoop() {
	defer o.log.Debug("Blob tip oracle polling loop exited")
	defer close(o.loopDone)

	ticker := time.NewTicker(o.config.PollRate)
	defer ticker.Stop()

	for {
		select {
		case <-o.ctx.Done():
			o.log.Info("Blob tip oracle context canceled")
			return
		case <-ticker.C:
			nextBlock := o.latestBlock + 1
			header, err := func() (*types.Header, error) {
				ctx, cancel := context.WithTimeout(o.ctx, o.config.NetworkTimeout)
				defer cancel()
				return o.backend.HeaderByNumber(ctx, big.NewInt(int64(nextBlock)))
			}()
			if errors.Is(err, ethereum.NotFound) {
				continue // Block not yet available
			}
			if err != nil {
				o.log.Warn("Failed to get header", "err", err, "block", nextBlock)
				continue
			}
			if err := o.processHeader(header); err != nil {
				o.log.Error("Error processing header", "err", err, "block", nextBlock)
			}
		}
	}
}

// prePopulateCache fetches and processes the last MaxBlocks blocks to pre-populate the cache.
func (o *BlobTipOracle) prePopulateCache() error {
	now := time.Now()

	ctx, cancel := context.WithTimeout(o.ctx, o.config.NetworkTimeout)
	defer cancel()

	// Get the latest block number
	latest, err := o.backend.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest block number: %w", err)
	}

	var startBlock uint64
	if latest >= uint64(o.config.MaxBlocks) {
		startBlock = latest - uint64(o.config.MaxBlocks) + 1
	} else {
		startBlock = 0
	}

	o.log.Debug("Pre-populating cache", "from", startBlock, "to", latest, "blocks", latest-startBlock+1)

	// Fetch and process each block
	for blockNum := startBlock; blockNum <= latest; blockNum++ {
		// Fetch header
		header, err := o.backend.HeaderByNumber(ctx, big.NewInt(int64(blockNum)))
		if err != nil {
			o.log.Debug("Failed to fetch header for pre-population", "block", blockNum, "err", err)
			continue
		}

		// Process header (this will also trigger blob fee cap fetching)
		if err := o.processHeader(header); err != nil {
			o.log.Debug("Failed to process header for pre-population", "block", blockNum, "err", err)
			continue
		}
	}

	o.log.Info("Cache pre-population complete", "blocks_processed", latest-startBlock+1, "took", time.Since(now))
	return nil
}

// processHeader calculates and stores the blob base fee for the given header.
// It also triggers an asynchronous fetch of the full block to extract blob fee caps.
func (o *BlobTipOracle) processHeader(header *types.Header) error {
	defer func(start time.Time) {
		o.log.Debug("Processed header", "block", bigs.Uint64Strict(header.Number), "time", time.Since(start))
	}(time.Now())

	o.Lock()
	defer o.Unlock()

	blockNum := bigs.Uint64Strict(header.Number)

	// Calculate blob base fee from the header
	if _, ok := o.baseFees.Get(blockNum); ok {
		o.log.Debug("Skipping blob base fee calculation, already processed", "block", blockNum, "latestBlock", o.latestBlock)
	} else {
		var blobBaseFee *big.Int
		if header.ExcessBlobGas != nil {
			blobBaseFee = eip4844.CalcBlobFee(o.chainConfig, header)
		}

		if blobBaseFee != nil {
			o.log.Debug("Adding blob base fee", "block", blockNum, "blobBaseFee", blobBaseFee.String())
			o.baseFees.Add(blockNum, blobBaseFee)
		} else {
			o.log.Debug("Block does not support blob transactions", "block", blockNum)
			o.baseFees.Add(blockNum, big.NewInt(0))
		}
	}

	// Fetch full block data and extract blob fee caps
	o.fetchBlockBlobFeeCaps(blockNum, header.BaseFee)

	if blockNum > o.latestBlock {
		o.latestBlock = blockNum
	}

	return nil
}

// fetchBlockBlobFeeCaps fetches a block and extracts blob fee caps, storing them in cache.
func (o *BlobTipOracle) fetchBlockBlobFeeCaps(blockNum uint64, baseFee *big.Int) {
	// Check if we already have the blob fee caps cached
	if _, ok := o.priorityFees.Get(blockNum); ok {
		o.log.Debug("Skipping blob fee caps fetch, already processed", "block", blockNum)
		return
	}

	ctx, cancel := context.WithTimeout(o.ctx, o.config.NetworkTimeout)
	defer cancel()

	// Fetch the block
	block, err := o.backend.BlockByNumber(ctx, big.NewInt(int64(blockNum)))
	if err != nil {
		o.log.Warn("Failed to fetch block for blob fee caps", "block", blockNum, "err", err)
		return
	}

	// Extract blob fee caps directly
	tips := o.extractTipsForBlobTxs(block, baseFee)

	// Store in cache (even if empty, to avoid repeated fetches)
	o.priorityFees.Add(blockNum, tips)
}

// GetLatestBlobBaseFee returns the blob base fee for the most recently processed block.
// Returns (0, nil) if no blocks have been processed yet, the price was evicted from cache,
// or if the latest block doesn't support blob transactions.
func (o *BlobTipOracle) GetLatestBlobBaseFee() (uint64, *big.Int) {
	o.Lock()
	defer o.Unlock()

	if o.latestBlock == 0 {
		return 0, nil
	}

	price, ok := o.baseFees.Get(o.latestBlock)
	if !ok {
		// Price was evicted from cache or block was never processed
		return 0, nil
	}
	if price == nil {
		// Block doesn't contain blob transactions
		return o.latestBlock, nil
	}
	// Return a copy to prevent external modification
	return o.latestBlock, new(big.Int).Set(price)
}

// SuggestBlobTipCap analyzes recent blocks to suggest an appropriate blob tip cap
// for blob transactions. It examines the last maxBlocks blocks and returns the
// percentile-th percentile of blob tip caps from blob transactions.
// This is similar to go-ethereum's oracle.SuggestTipCap but for tips solely on blob transactions (type 3).
//
// This method only reads from cache and does not make any RPC calls. Block data
// is fetched during block processing.
//
// If no blob transactions are found in recent blocks, it returns the current blob base fee
// plus a small buffer to ensure the transaction is competitive.
func (o *BlobTipOracle) SuggestBlobTipCap(ctx context.Context, maxBlocks int, percentile int) (*big.Int, error) {
	if maxBlocks <= 0 {
		maxBlocks = o.config.MaxBlocks
	}
	if percentile <= 0 || percentile > 100 {
		percentile = o.config.Percentile
	}

	// Get the latest block number from our tracked state (no RPC call)
	o.Lock()
	latestBlockNum := o.latestBlock
	o.Unlock()

	if latestBlockNum == 0 {
		return nil, fmt.Errorf("no blocks have been processed yet")
	}

	// Collect blob fee caps from recent blocks (only from cache, no RPC calls)
	var tips []*big.Int
	startBlock := latestBlockNum
	if startBlock >= uint64(maxBlocks) {
		startBlock -= uint64(maxBlocks)
	} else {
		startBlock = 0
	}

	for blockNum := startBlock; blockNum <= latestBlockNum; blockNum++ {
		// Only read from cache - no RPC calls
		if t, ok := o.priorityFees.Get(blockNum); ok {
			tips = append(tips, t...)
		}
	}

	// If we found blob transactions, calculate percentile
	if len(tips) > 0 {
		sort.Slice(tips, func(i, j int) bool {
			return tips[i].Cmp(tips[j]) < 0
		})
		idx := (len(tips) - 1) * percentile / 100
		suggested := new(big.Int).Set(tips[idx])
		o.log.Debug("Suggested blob tip cap from recent transactions", "suggested", suggested.String(), "samples", len(tips), "percentile", percentile)
		return suggested, nil
	}

	// No blob transactions found, use the default priority fee - that should almost never happen, so we warn about it
	o.log.Warn("No recent blob transactions found, using default prio fee", "block", latestBlockNum, "defaultPriorityFee", o.config.DefaultPriorityFee)
	return new(big.Int).Set(o.config.DefaultPriorityFee), nil
}

// extractTipsForBlobTxs extracts tips for blob transactions from a block's transactions.
func (o *BlobTipOracle) extractTipsForBlobTxs(block *types.Block, baseFee *big.Int) []*big.Int {
	var tips []*big.Int
	for _, tx := range block.Transactions() {
		// Check if it's a blob transaction (type 3) and has blob fee cap
		if tx.Type() == types.BlobTxType {
			tip, err := tx.EffectiveGasTip(baseFee) // tip calculated from execution gas, for a type 3 transaction
			if err != nil {
				o.log.Error("Failed to calculate effective gas tip", "block", block.NumberU64(), "err", err)
				continue
			}

			tips = append(tips, tip)
			o.log.Debug("Extracted tip from blob tx", "block", block.NumberU64(), "tip", tip.String())
		}
	}
	return tips
}

// Close stops the oracle and cleans up resources.
func (o *BlobTipOracle) Close() {
	o.cancel()
	if o.loopDone != nil {
		<-o.loopDone
	}
	o.log.Info("Blob tip oracle closed")
}
