package bgpo

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources/caching"
)

// BlobGasPriceOracle tracks blob base gas prices by subscribing to new block headers
// and calculating the blob base fee for each block.
type BlobGasPriceOracle struct {
	sync.Mutex

	client      client.RPC
	chainConfig *params.ChainConfig
	log         log.Logger

	// LRU cache for blob base fees by block number
	prices *caching.LRUCache[uint64, *big.Int]

	// Cache for blob fee caps extracted from blocks (for SuggestBlobTipCap)
	blobFeeCaps *caching.LRUCache[uint64, []*big.Int]

	// Default values for SuggestBlobTipCap
	maxBlocks  int
	percentile int

	// Track the latest block number for GetLatestBlobBaseFee
	latestBlock uint64

	ctx    context.Context
	cancel context.CancelFunc

	sub ethereum.Subscription
}

// rpcBlock structure for fetching blocks with transactions.
// When eth_getBlockByNumber is called with true, it returns full transaction objects.
type rpcBlock struct {
	Number       hexutil.Uint64       `json:"number"`
	Hash         hexutil.Bytes        `json:"hash"`
	Transactions []*types.Transaction `json:"transactions"`
}

// BlobGasPriceOracleConfig configures the blob gas price oracle.
type BlobGasPriceOracleConfig struct {
	// PricesCacheSize is the maximum number of blob base fees to cache (default: 1000)
	PricesCacheSize int
	// BlockCacheSize is the maximum number of blocks to cache for RPC calls (default: 100)
	BlockCacheSize int
	// MaxBlocks is the default number of recent blocks to analyze in SuggestBlobTipCap (default: 20)
	MaxBlocks int
	// Percentile is the default percentile to use for blob tip cap suggestion (default: 60)
	Percentile int
	// Metrics for cache tracking (optional)
	Metrics caching.Metrics
}

// DefaultBlobGasPriceOracleConfig returns a default configuration.
func DefaultBlobGasPriceOracleConfig() *BlobGasPriceOracleConfig {
	return &BlobGasPriceOracleConfig{
		PricesCacheSize: 1000,
		BlockCacheSize:  100,
		MaxBlocks:       20,
		Percentile:      60,
		Metrics:         nil,
	}
}

// NewBlobGasPriceOracle creates a new blob gas price oracle that will subscribe
// to newHeads and track blob base fees.
func NewBlobGasPriceOracle(ctx context.Context, rpcClient client.RPC, chainConfig *params.ChainConfig, log log.Logger, config *BlobGasPriceOracleConfig) *BlobGasPriceOracle {
	defaultConfig := DefaultBlobGasPriceOracleConfig()
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

	oracleCtx, cancel := context.WithCancel(ctx)
	return &BlobGasPriceOracle{
		client:      rpcClient,
		chainConfig: chainConfig,
		log:         log.With("module", "bgpo"),
		prices:      caching.NewLRUCache[uint64, *big.Int](config.Metrics, "bgpo_prices", config.PricesCacheSize),
		blobFeeCaps: caching.NewLRUCache[uint64, []*big.Int](config.Metrics, "bgpo_fee_caps", config.BlockCacheSize),
		maxBlocks:   config.MaxBlocks,
		percentile:  config.Percentile,
		ctx:         oracleCtx,
		cancel:      cancel,
	}
}

// Start begins subscribing to newHeads and processing headers.
// Before subscribing, it pre-populates the cache with the last MaxBlocks blocks.
// This method blocks until the context is canceled or an error occurs.
func (o *BlobGasPriceOracle) Start() error {
	// Pre-populate cache with recent blocks before subscribing
	if err := o.prePopulateCache(); err != nil {
		o.log.Warn("Failed to pre-populate cache, continuing anyway", "err", err)
	}

	headers := make(chan *types.Header, 10)

	sub, err := o.client.Subscribe(o.ctx, "eth", headers, "newHeads")
	if err != nil {
		return err
	}
	o.sub = sub

	o.log.Info("Blob gas price oracle started, subscribed to newHeads")

	// Process headers as they arrive
	for {
		select {
		case header := <-headers:
			if err := o.processHeader(header); err != nil {
				o.log.Error("Error processing header", "err", err, "block", header.Number.Uint64())
			}
		case err := <-sub.Err():
			if err != nil {
				o.log.Error("Subscription error", "err", err)
				return err
			}
			return nil
		case <-o.ctx.Done():
			o.log.Info("Blob gas price oracle context canceled")
			return nil
		}
	}
}

// prePopulateCache fetches and processes the last MaxBlocks blocks to pre-populate the cache.
func (o *BlobGasPriceOracle) prePopulateCache() error {
	ctx, cancel := context.WithTimeout(o.ctx, 30*time.Second)
	defer cancel()

	// Get the latest block number
	var latestBlockNum hexutil.Uint64
	if err := o.client.CallContext(ctx, &latestBlockNum, "eth_blockNumber"); err != nil {
		return fmt.Errorf("failed to get latest block number: %w", err)
	}

	latest := uint64(latestBlockNum)
	var startBlock uint64
	if latest >= uint64(o.maxBlocks) {
		startBlock = latest - uint64(o.maxBlocks) + 1
	} else {
		startBlock = 0
	}

	o.log.Info("Pre-populating cache", "from", startBlock, "to", latest, "blocks", latest-startBlock+1)

	// Fetch and process each block
	for blockNum := startBlock; blockNum <= latest; blockNum++ {
		// Fetch header
		var header *types.Header
		blockNumHex := hexutil.EncodeUint64(blockNum)
		if err := o.client.CallContext(ctx, &header, "eth_getBlockByNumber", blockNumHex, false); err != nil {
			o.log.Debug("Failed to fetch header for pre-population", "block", blockNum, "err", err)
			continue
		}

		// Process header (this will also trigger blob fee cap fetching)
		if err := o.processHeader(header); err != nil {
			o.log.Debug("Failed to process header for pre-population", "block", blockNum, "err", err)
			continue
		}
	}

	o.log.Info("Cache pre-population complete", "blocks_processed", latest-startBlock+1)
	return nil
}

// processHeader calculates and stores the blob base fee for the given header.
// It also triggers an asynchronous fetch of the full block to extract blob fee caps.
func (o *BlobGasPriceOracle) processHeader(header *types.Header) error {
	defer func(now time.Time) {
		o.log.Debug("Processed header", "block", header.Number.Uint64(), "time", time.Since(now))
	}(time.Now())

	o.Lock()
	defer o.Unlock()

	blockNum := header.Number.Uint64()

	// Calculate blob base fee from the header
	var blobBaseFee *big.Int
	if header.ExcessBlobGas != nil {
		blobBaseFee = eip4844.CalcBlobFee(o.chainConfig, header)
	}

	o.prices.Add(blockNum, blobBaseFee)

	if blockNum > o.latestBlock {
		o.latestBlock = blockNum
	}

	// Fetch full block data and extract blob fee caps
	o.fetchBlockBlobFeeCaps(blockNum)

	if blobBaseFee != nil {
		o.log.Debug("Recorded blob base fee", "block", blockNum, "blobBaseFee", blobBaseFee.String())
	} else {
		o.log.Debug("Block does not support blob transactions", "block", blockNum)
	}

	return nil
}

// fetchBlockBlobFeeCaps fetches a block and extracts blob fee caps, storing them in cache.
func (o *BlobGasPriceOracle) fetchBlockBlobFeeCaps(blockNum uint64) {
	// Check if we already have the blob fee caps cached
	if _, ok := o.blobFeeCaps.Get(blockNum); ok {
		return
	}

	ctx, cancel := context.WithTimeout(o.ctx, 10*time.Second)
	defer cancel()

	// Fetch the block
	var block rpcBlock
	blockNumHex := hexutil.EncodeUint64(blockNum)
	if err := o.client.CallContext(ctx, &block, "eth_getBlockByNumber", blockNumHex, true); err != nil {
		o.log.Debug("Failed to fetch block for blob fee caps", "block", blockNum, "err", err)
		return
	}

	// Extract blob fee caps directly
	feeCaps := o.extractBlobFeeCaps(block)

	// Store in cache (even if empty, to avoid repeated fetches)
	o.blobFeeCaps.Add(blockNum, feeCaps)
}

// GetBlobBaseFee returns the blob base fee for the given block number.
// Returns nil if the block number hasn't been processed yet or if the block
// doesn't support blob transactions.
func (o *BlobGasPriceOracle) GetBlobBaseFee(blockNum uint64) *big.Int {
	price, ok := o.prices.Get(blockNum)
	if !ok {
		return nil
	}
	if price == nil {
		return nil
	}
	// Return a copy to prevent external modification
	return new(big.Int).Set(price)
}

// GetLatestBlobBaseFee returns the blob base fee for the most recently processed block.
// Returns (0, nil) if no blocks have been processed yet, the price was evicted from cache,
// or if the latest block doesn't support blob transactions.
func (o *BlobGasPriceOracle) GetLatestBlobBaseFee() (uint64, *big.Int) {
	o.Lock()
	defer o.Unlock()

	if o.latestBlock == 0 {
		return 0, nil
	}

	price, ok := o.prices.Get(o.latestBlock)
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

// SuggestBlobTipCap analyzes recent blocks to suggest an appropriate blob fee cap
// for blob transactions. It examines the last maxBlocks blocks and returns the
// percentile-th percentile of blob fee caps from blob transactions.
// This is similar to go-ethereum's oracle.SuggestTipCap but for blob prices.
//
// This method only reads from cache and does not make any RPC calls. Block data
// is fetched during block processing.
//
// If no blob transactions are found in recent blocks, it returns the current blob base fee
// plus a small buffer to ensure the transaction is competitive.
func (o *BlobGasPriceOracle) SuggestBlobTipCap(ctx context.Context, maxBlocks int, percentile int) (*big.Int, error) {
	if maxBlocks <= 0 {
		maxBlocks = o.maxBlocks
	}
	if percentile <= 0 || percentile > 100 {
		percentile = o.percentile
	}

	// Get the latest block number from our tracked state (no RPC call)
	o.Lock()
	latestBlockNum := o.latestBlock
	o.Unlock()

	if latestBlockNum == 0 {
		return nil, fmt.Errorf("no blocks have been processed yet")
	}

	// Collect blob fee caps from recent blocks (only from cache, no RPC calls)
	var blobFeeCaps []*big.Int
	startBlock := latestBlockNum
	if startBlock >= uint64(maxBlocks) {
		startBlock -= uint64(maxBlocks)
	} else {
		startBlock = 0
	}

	for blockNum := startBlock; blockNum <= latestBlockNum; blockNum++ {
		// Only read from cache - no RPC calls
		if feeCaps, ok := o.blobFeeCaps.Get(blockNum); ok {
			blobFeeCaps = append(blobFeeCaps, feeCaps...)
		}
	}

	// If we found blob transactions, calculate percentile
	if len(blobFeeCaps) > 0 {
		sort.Slice(blobFeeCaps, func(i, j int) bool {
			return blobFeeCaps[i].Cmp(blobFeeCaps[j]) < 0
		})
		idx := (len(blobFeeCaps) - 1) * percentile / 100
		suggested := new(big.Int).Set(blobFeeCaps[idx])
		o.log.Debug("Suggested blob tip cap from recent transactions", "suggested", suggested.String(), "samples", len(blobFeeCaps), "percentile", percentile)
		return suggested, nil
	}

	// No blob transactions found, use current blob base fee + buffer
	latestBlock, blobBaseFee := o.GetLatestBlobBaseFee()
	if blobBaseFee == nil {
		return nil, fmt.Errorf("no blob base fee available and no recent blob transactions found")
	}

	// Add 10% buffer to the base fee to ensure competitiveness
	buffer := new(big.Int).Div(blobBaseFee, big.NewInt(10))
	suggested := new(big.Int).Add(blobBaseFee, buffer)
	o.log.Debug("No recent blob transactions found, using blob base fee + buffer", "block", latestBlock, "blobBaseFee", blobBaseFee.String(), "suggested", suggested.String())
	return suggested, nil
}

// extractBlobFeeCaps extracts blob fee caps from a block's transactions.
func (o *BlobGasPriceOracle) extractBlobFeeCaps(block rpcBlock) []*big.Int {
	var feeCaps []*big.Int
	for _, tx := range block.Transactions {
		// Check if it's a blob transaction (type 3) and has blob fee cap
		if tx.Type() == types.BlobTxType {
			if blobFeeCap := tx.BlobGasFeeCap(); blobFeeCap != nil {
				feeCaps = append(feeCaps, blobFeeCap)
			}
		}
	}
	return feeCaps
}

// Close stops the oracle and cleans up resources.
func (o *BlobGasPriceOracle) Close() {
	o.cancel()
	if o.sub != nil {
		o.sub.Unsubscribe()
	}
	o.log.Info("Blob gas price oracle closed")
}
