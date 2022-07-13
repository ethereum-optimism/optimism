package l1

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum/go-ethereum"

	"github.com/ethereum-optimism/optimism/op-node/rollup"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	lru "github.com/hashicorp/golang-lru"
)

type SourceConfig struct {
	// batching parameters
	MaxParallelBatching int
	MaxBatchRetry       int
	MaxRequestsPerBatch int

	// limit concurrent requests, applies to the source as a whole
	MaxConcurrentRequests int

	// cache sizes

	// Number of blocks worth of receipts to cache
	ReceiptsCacheSize int
	// Number of blocks worth of transactions to cache
	TransactionsCacheSize int
	// Number of block headers to cache
	HeadersCacheSize int

	// If the RPC is untrusted, then we should not use cached information from responses,
	// and instead verify against the block-hash.
	// Of real L1 blocks no deposits can be missed/faked, no batches can be missed/faked,
	// only the wrong L1 blocks can be retrieved.
	TrustRPC bool
}

func (c *SourceConfig) Check() error {
	if c.ReceiptsCacheSize < 0 {
		return fmt.Errorf("invalid receipts cache size: %d", c.ReceiptsCacheSize)
	}
	if c.TransactionsCacheSize < 0 {
		return fmt.Errorf("invalid transactions cache size: %d", c.TransactionsCacheSize)
	}
	if c.HeadersCacheSize < 0 {
		return fmt.Errorf("invalid headers cache size: %d", c.HeadersCacheSize)
	}
	if c.MaxConcurrentRequests < 1 {
		return fmt.Errorf("expected at least 1 concurrent request, but max is %d", c.MaxConcurrentRequests)
	}
	if c.MaxParallelBatching < 1 {
		return fmt.Errorf("expected at least 1 batch request to run at a time, but max is %d", c.MaxParallelBatching)
	}
	if c.MaxBatchRetry < 0 || c.MaxBatchRetry > 20 {
		return fmt.Errorf("number of max batch retries is not reasonable: %d", c.MaxBatchRetry)
	}
	if c.MaxRequestsPerBatch < 1 {
		return fmt.Errorf("expected at least 1 request per batch, but max is: %d", c.MaxRequestsPerBatch)
	}
	return nil
}

func DefaultConfig(config *rollup.Config, trustRPC bool) *SourceConfig {
	return &SourceConfig{
		// We only consume receipts once per block,
		// we just need basic redundancy if we share the cache between multiple drivers
		ReceiptsCacheSize: 20,

		// Optimal if at least a few times the size of a sequencing window.
		// When smaller than a window, requests would be repeated every window shift.
		// Additional cache-size for handling reorgs, and thus more unique blocks, also helps.
		TransactionsCacheSize: int(config.SeqWindowSize * 4),
		HeadersCacheSize:      int(config.SeqWindowSize * 4),

		// TODO: tune batch params
		MaxParallelBatching: 8,
		MaxBatchRetry:       3,
		MaxRequestsPerBatch: 20,

		MaxConcurrentRequests: 10,

		TrustRPC: trustRPC,
	}
}

type batchCallContextFn func(ctx context.Context, b []rpc.BatchElem) error

// Source to retrieve L1 data from with optimized batch requests, cached results,
// and flag to not trust the RPC.
type Source struct {
	client client.RPC

	batchCall batchCallContextFn

	trustRPC bool

	// cache receipts in bundles per block hash
	// common.Hash -> types.Receipts
	receiptsCache *lru.Cache

	// cache transactions in bundles per block hash
	// common.Hash -> types.Transactions
	transactionsCache *lru.Cache

	// cache block headers of blocks by hash
	// common.Hash -> *HeaderInfo
	headersCache *lru.Cache
}

func NewSource(client client.RPC, log log.Logger, config *SourceConfig) (*Source, error) {
	if err := config.Check(); err != nil {
		return nil, fmt.Errorf("bad config, cannot create L1 source: %w", err)
	}
	// no errors if the size is positive, as already validated by Check() above.
	receiptsCache, _ := lru.New(config.ReceiptsCacheSize)
	transactionsCache, _ := lru.New(config.TransactionsCacheSize)
	headersCache, _ := lru.New(config.HeadersCacheSize)

	client = LimitRPC(client, config.MaxConcurrentRequests)

	// Batch calls will be split up to handle max-batch size,
	// and parallelized since the RPC server does not parallelize batch contents otherwise.
	getBatch := parallelBatchCall(log, client.BatchCallContext,
		config.MaxBatchRetry, config.MaxRequestsPerBatch, config.MaxParallelBatching)
	return &Source{
		client:            client,
		batchCall:         getBatch,
		trustRPC:          config.TrustRPC,
		receiptsCache:     receiptsCache,
		transactionsCache: transactionsCache,
		headersCache:      headersCache,
	}, nil
}

// SubscribeNewHead subscribes to notifications about the current blockchain head on the given channel.
func (s *Source) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	// Note that *types.Header does not cache the block hash unlike *HeaderInfo, it always recomputes.
	// Inefficient if used poorly, but no trust issue.
	return s.client.EthSubscribe(ctx, ch, "newHeads")
}

func (s *Source) headerCall(ctx context.Context, method string, id interface{}) (*HeaderInfo, error) {
	var header *rpcHeader
	err := s.client.CallContext(ctx, &header, method, id, false) // headers are just blocks without txs
	if err != nil {
		return nil, err
	}
	if header == nil {
		return nil, ethereum.NotFound
	}
	info, err := header.Info(s.trustRPC)
	if err != nil {
		return nil, err
	}
	s.headersCache.Add(info.hash, info)
	return info, nil
}

func (s *Source) blockCall(ctx context.Context, method string, id interface{}) (*HeaderInfo, types.Transactions, error) {
	var block *rpcBlock
	err := s.client.CallContext(ctx, &block, method, id, true)
	if err != nil {
		return nil, nil, err
	}
	if block == nil {
		return nil, nil, ethereum.NotFound
	}
	info, txs, err := block.Info(s.trustRPC)
	if err != nil {
		return nil, nil, err
	}
	s.headersCache.Add(info.hash, info)
	s.transactionsCache.Add(info.hash, txs)
	return info, txs, nil
}

func (s *Source) InfoByHash(ctx context.Context, hash common.Hash) (eth.L1Info, error) {
	if header, ok := s.headersCache.Get(hash); ok {
		return header.(*HeaderInfo), nil
	}
	return s.headerCall(ctx, "eth_getBlockByHash", hash)
}

func (s *Source) InfoByNumber(ctx context.Context, number uint64) (eth.L1Info, error) {
	// can't hit the cache when querying by number due to reorgs.
	return s.headerCall(ctx, "eth_getBlockByNumber", hexutil.EncodeUint64(number))
}

func (s *Source) InfoHead(ctx context.Context) (eth.L1Info, error) {
	// can't hit the cache when querying the head due to reorgs / changes.
	return s.headerCall(ctx, "eth_getBlockByNumber", "latest")
}

func (s *Source) InfoAndTxsByHash(ctx context.Context, hash common.Hash) (eth.L1Info, types.Transactions, error) {
	if header, ok := s.headersCache.Get(hash); ok {
		if txs, ok := s.transactionsCache.Get(hash); ok {
			return header.(*HeaderInfo), txs.(types.Transactions), nil
		}
	}
	return s.blockCall(ctx, "eth_getBlockByHash", hash)
}

func (s *Source) InfoAndTxsByNumber(ctx context.Context, number uint64) (eth.L1Info, types.Transactions, error) {
	// can't hit the cache when querying by number due to reorgs.
	return s.blockCall(ctx, "eth_getBlockByNumber", hexutil.EncodeUint64(number))
}

func (s *Source) InfoAndTxsHead(ctx context.Context) (eth.L1Info, types.Transactions, error) {
	// can't hit the cache when querying the head due to reorgs / changes.
	return s.blockCall(ctx, "eth_getBlockByNumber", "latest")
}

func (s *Source) Fetch(ctx context.Context, blockHash common.Hash) (eth.L1Info, types.Transactions, types.Receipts, error) {
	if blockHash == (common.Hash{}) {
		return nil, nil, nil, ethereum.NotFound
	}
	info, txs, err := s.blockCall(ctx, "eth_getBlockByHash", blockHash)
	if err != nil {
		return nil, nil, nil, err
	}

	receipts, err := fetchReceipts(ctx, info.ID(), info.receiptHash, txs, s.batchCall)
	if err != nil {
		return nil, nil, nil, err
	}
	s.receiptsCache.Add(info.hash, receipts)
	return info, txs, receipts, nil
}

// FetchAllTransactions fetches transaction lists of a window of blocks, and caches each block and the transactions
func (s *Source) FetchAllTransactions(ctx context.Context, window []eth.BlockID) ([]types.Transactions, error) {
	// list of transaction lists
	allTxLists := make([]types.Transactions, len(window))

	var blockRequests []rpc.BatchElem
	var requestIndices []int

	for i := 0; i < len(window); i++ {
		// if we are shifting the window by 1 block at a time, most of the results should already be in the cache.
		txs, ok := s.transactionsCache.Get(window[i].Hash)
		if ok {
			allTxLists[i] = txs.(types.Transactions)
		} else {
			blockRequests = append(blockRequests, rpc.BatchElem{
				Method: "eth_getBlockByHash",
				Args:   []interface{}{window[i].Hash, true}, // request block including transactions list
				Result: new(rpcBlock),
				Error:  nil,
			})
			requestIndices = append(requestIndices, i) // remember the block index this request corresponds to
		}
	}

	if len(blockRequests) > 0 {
		if err := s.batchCall(ctx, blockRequests); err != nil {
			return nil, err
		}
	}

	// try to cache everything we have before halting on the results with errors
	for i := 0; i < len(blockRequests); i++ {
		if blockRequests[i].Error == nil {
			info, txs, err := blockRequests[i].Result.(*rpcBlock).Info(s.trustRPC)
			if err != nil {
				return nil, fmt.Errorf("bad block data for block %s: %w", blockRequests[i].Args[0], err)
			}
			s.headersCache.Add(info.hash, info)
			s.transactionsCache.Add(info.hash, txs)
			allTxLists[requestIndices[i]] = txs
		}
	}

	for i := 0; i < len(blockRequests); i++ {
		if blockRequests[i].Error != nil {
			return nil, fmt.Errorf("failed to retrieve transactions of block %s in batch of %d blocks: %w", window[i], len(blockRequests), blockRequests[i].Error)
		}
	}

	return allTxLists, nil
}

func (s *Source) L1HeadBlockRef(ctx context.Context) (eth.L1BlockRef, error) {
	head, err := s.InfoHead(ctx)
	if err != nil {
		return eth.L1BlockRef{}, fmt.Errorf("failed to fetch head header: %w", err)
	}
	return head.BlockRef(), nil
}

func (s *Source) L1BlockRefByNumber(ctx context.Context, l1Num uint64) (eth.L1BlockRef, error) {
	head, err := s.InfoByNumber(ctx, l1Num)
	if err != nil {
		return eth.L1BlockRef{}, fmt.Errorf("failed to fetch header by num %d: %w", l1Num, err)
	}
	return head.BlockRef(), nil
}

func (s *Source) L1BlockRefByHash(ctx context.Context, hash common.Hash) (eth.L1BlockRef, error) {
	block, err := s.InfoByHash(ctx, hash)
	if err != nil {
		return eth.L1BlockRef{}, fmt.Errorf("failed to fetch header by hash %v: %w", hash, err)
	}
	return block.BlockRef(), nil
}

// L1Range returns a range of L1 block beginning just after begin, up to max blocks.
// This batch-requests all blocks by number in the range at once, and then verifies the consistency
func (s *Source) L1Range(ctx context.Context, begin eth.BlockID, max uint64) ([]eth.BlockID, error) {
	headerRequests := make([]rpc.BatchElem, max)
	for i := uint64(0); i < max; i++ {
		headerRequests[i] = rpc.BatchElem{
			Method: "eth_getBlockByNumber",
			Args:   []interface{}{hexutil.EncodeUint64(begin.Number + 1 + i), false},
			Result: new(*rpcHeader),
			Error:  nil,
		}
	}
	if err := s.batchCall(ctx, headerRequests); err != nil {
		return nil, err
	}

	out := make([]eth.BlockID, 0, max)

	// try to cache everything we have before halting on the results with errors
	for i := 0; i < len(headerRequests); i++ {
		result := *headerRequests[i].Result.(**rpcHeader)
		if headerRequests[i].Error == nil {
			if result == nil {
				break // no more headers from here
			}
			info, err := result.Info(s.trustRPC)
			if err != nil {
				return nil, fmt.Errorf("bad header data for block %s: %w", headerRequests[i].Args[0], err)
			}
			s.headersCache.Add(info.hash, info)
			out = append(out, info.ID())
			prev := begin
			if i > 0 {
				prev = out[i-1]
			}
			if prev.Hash != info.parentHash {
				return nil, fmt.Errorf("inconsistent results from L1 chain range request, block %s not expected parent %s of %s", prev, info.parentHash, info.ID())
			}
		} else if errors.Is(headerRequests[i].Error, ethereum.NotFound) {
			break // no more headers from here
		} else {
			return nil, fmt.Errorf("failed to retrieve block: %s: %w", headerRequests[i].Args[0], headerRequests[i].Error)
		}
	}
	return out, nil
}

func (s *Source) Close() {
	s.client.Close()
}
