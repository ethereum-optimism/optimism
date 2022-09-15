package sources

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/sources/caching"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type EthClientConfig struct {
	// Maximum number of requests to make per batch
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
	// Number of payloads to cache
	PayloadsCacheSize int

	// If the RPC is untrusted, then we should not use cached information from responses,
	// and instead verify against the block-hash.
	// Of real L1 blocks no deposits can be missed/faked, no batches can be missed/faked,
	// only the wrong L1 blocks can be retrieved.
	TrustRPC bool

	// If the RPC must ensure that the results fit the ExecutionPayload(Header) format.
	// If this is not checked, disabled header fields like the nonce or difficulty
	// may be used to get a different block-hash.
	MustBePostMerge bool
}

func (c *EthClientConfig) Check() error {
	if c.ReceiptsCacheSize < 0 {
		return fmt.Errorf("invalid receipts cache size: %d", c.ReceiptsCacheSize)
	}
	if c.TransactionsCacheSize < 0 {
		return fmt.Errorf("invalid transactions cache size: %d", c.TransactionsCacheSize)
	}
	if c.HeadersCacheSize < 0 {
		return fmt.Errorf("invalid headers cache size: %d", c.HeadersCacheSize)
	}
	if c.PayloadsCacheSize < 0 {
		return fmt.Errorf("invalid payloads cache size: %d", c.PayloadsCacheSize)
	}
	if c.MaxConcurrentRequests < 1 {
		return fmt.Errorf("expected at least 1 concurrent request, but max is %d", c.MaxConcurrentRequests)
	}
	if c.MaxRequestsPerBatch < 1 {
		return fmt.Errorf("expected at least 1 request per batch, but max is: %d", c.MaxRequestsPerBatch)
	}
	return nil
}

// EthClient retrieves ethereum data with optimized batch requests, cached results, and flag to not trust the RPC.
type EthClient struct {
	client client.RPC

	maxBatchSize int

	trustRPC bool

	mustBePostMerge bool

	log log.Logger

	// cache receipts in bundles per block hash
	// common.Hash -> types.Receipts
	receiptsCache *caching.LRUCache

	// cache transactions in bundles per block hash
	// common.Hash -> types.Transactions
	transactionsCache *caching.LRUCache

	// cache block headers of blocks by hash
	// common.Hash -> *HeaderInfo
	headersCache *caching.LRUCache

	// cache payloads by hash
	// common.Hash -> *eth.ExecutionPayload
	payloadsCache *caching.LRUCache
}

// NewEthClient wraps a RPC with bindings to fetch ethereum data,
// while logging errors, parallel-requests constraint, tracking metrics (optional), and caching.
func NewEthClient(client client.RPC, log log.Logger, metrics caching.Metrics, config *EthClientConfig) (*EthClient, error) {
	if err := config.Check(); err != nil {
		return nil, fmt.Errorf("bad config, cannot create L1 source: %w", err)
	}
	client = LimitRPC(client, config.MaxConcurrentRequests)
	return &EthClient{
		client:            client,
		maxBatchSize:      config.MaxRequestsPerBatch,
		trustRPC:          config.TrustRPC,
		log:               log,
		receiptsCache:     caching.NewLRUCache(metrics, "receipts", config.ReceiptsCacheSize),
		transactionsCache: caching.NewLRUCache(metrics, "txs", config.TransactionsCacheSize),
		headersCache:      caching.NewLRUCache(metrics, "headers", config.HeadersCacheSize),
		payloadsCache:     caching.NewLRUCache(metrics, "payloads", config.PayloadsCacheSize),
	}, nil
}

// SubscribeNewHead subscribes to notifications about the current blockchain head on the given channel.
func (s *EthClient) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	// Note that *types.Header does not cache the block hash unlike *HeaderInfo, it always recomputes.
	// Inefficient if used poorly, but no trust issue.
	return s.client.EthSubscribe(ctx, ch, "newHeads")
}

func (s *EthClient) headerCall(ctx context.Context, method string, id interface{}) (*HeaderInfo, error) {
	var header *rpcHeader
	err := s.client.CallContext(ctx, &header, method, id, false) // headers are just blocks without txs
	if err != nil {
		return nil, err
	}
	if header == nil {
		return nil, ethereum.NotFound
	}
	info, err := header.Info(s.trustRPC, s.mustBePostMerge)
	if err != nil {
		return nil, err
	}
	s.headersCache.Add(info.Hash(), info)
	return info, nil
}

func (s *EthClient) blockCall(ctx context.Context, method string, id interface{}) (*HeaderInfo, types.Transactions, error) {
	var block *rpcBlock
	err := s.client.CallContext(ctx, &block, method, id, true)
	if err != nil {
		return nil, nil, err
	}
	if block == nil {
		return nil, nil, ethereum.NotFound
	}
	info, txs, err := block.Info(s.trustRPC, s.mustBePostMerge)
	if err != nil {
		return nil, nil, err
	}
	s.headersCache.Add(info.Hash(), info)
	s.transactionsCache.Add(info.Hash(), txs)
	return info, txs, nil
}

func (s *EthClient) payloadCall(ctx context.Context, method string, id interface{}) (*eth.ExecutionPayload, error) {
	var block *rpcBlock
	err := s.client.CallContext(ctx, &block, method, id, true)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, ethereum.NotFound
	}
	payload, err := block.ExecutionPayload(s.trustRPC)
	if err != nil {
		return nil, err
	}
	s.payloadsCache.Add(payload.BlockHash, payload)
	return payload, nil
}

func (s *EthClient) InfoByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, error) {
	if header, ok := s.headersCache.Get(hash); ok {
		return header.(*HeaderInfo), nil
	}
	return s.headerCall(ctx, "eth_getBlockByHash", hash)
}

func (s *EthClient) InfoByNumber(ctx context.Context, number uint64) (eth.BlockInfo, error) {
	// can't hit the cache when querying by number due to reorgs.
	return s.headerCall(ctx, "eth_getBlockByNumber", hexutil.EncodeUint64(number))
}

func (s *EthClient) InfoByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, error) {
	// can't hit the cache when querying the head due to reorgs / changes.
	return s.headerCall(ctx, "eth_getBlockByNumber", string(label))
}

func (s *EthClient) InfoByRpcNumber(ctx context.Context, num rpc.BlockNumber) (eth.BlockInfo, error) {
	// can't hit the cache when querying the head due to reorgs / changes.
	return s.headerCall(ctx, "eth_getBlockByNumber", num)
}

func (s *EthClient) InfoAndTxsByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, types.Transactions, error) {
	if header, ok := s.headersCache.Get(hash); ok {
		if txs, ok := s.transactionsCache.Get(hash); ok {
			return header.(*HeaderInfo), txs.(types.Transactions), nil
		}
	}
	return s.blockCall(ctx, "eth_getBlockByHash", hash)
}

func (s *EthClient) InfoAndTxsByNumber(ctx context.Context, number uint64) (eth.BlockInfo, types.Transactions, error) {
	// can't hit the cache when querying by number due to reorgs.
	return s.blockCall(ctx, "eth_getBlockByNumber", hexutil.EncodeUint64(number))
}

func (s *EthClient) InfoAndTxsByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, types.Transactions, error) {
	// can't hit the cache when querying the head due to reorgs / changes.
	return s.blockCall(ctx, "eth_getBlockByNumber", string(label))
}

func (s *EthClient) PayloadByHash(ctx context.Context, hash common.Hash) (*eth.ExecutionPayload, error) {
	if payload, ok := s.payloadsCache.Get(hash); ok {
		return payload.(*eth.ExecutionPayload), nil
	}
	return s.payloadCall(ctx, "eth_getBlockByHash", hash)
}

func (s *EthClient) PayloadByNumber(ctx context.Context, number uint64) (*eth.ExecutionPayload, error) {
	return s.payloadCall(ctx, "eth_getBlockByNumber", hexutil.EncodeUint64(number))
}

func (s *EthClient) PayloadByLabel(ctx context.Context, label eth.BlockLabel) (*eth.ExecutionPayload, error) {
	return s.payloadCall(ctx, "eth_getBlockByNumber", string(label))
}

func (s *EthClient) Fetch(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Transactions, eth.ReceiptsFetcher, error) {
	info, txs, err := s.InfoAndTxsByHash(ctx, blockHash)
	if err != nil {
		return nil, nil, nil, err
	}
	if v, ok := s.receiptsCache.Get(blockHash); ok {
		return info, txs, v.(eth.ReceiptsFetcher), nil
	}
	txHashes := make([]common.Hash, len(txs))
	for i := 0; i < len(txs); i++ {
		txHashes[i] = txs[i].Hash()
	}
	r := NewReceiptsFetcher(info.ID(), info.ReceiptHash(), txHashes, s.client.BatchCallContext, s.maxBatchSize)
	s.receiptsCache.Add(blockHash, r)
	return info, txs, r, nil
}

// BlockIDRange returns a range of block IDs from the provided begin up to max blocks after the begin.
// This batch-requests all blocks by number in the range at once, and then verifies the consistency
func (s *EthClient) BlockIDRange(ctx context.Context, begin eth.BlockID, max uint64) ([]eth.BlockID, error) {
	headerRequests := make([]rpc.BatchElem, max)
	for i := uint64(0); i < max; i++ {
		headerRequests[i] = rpc.BatchElem{
			Method: "eth_getBlockByNumber",
			Args:   []interface{}{hexutil.EncodeUint64(begin.Number + 1 + i), false},
			Result: new(*rpcHeader),
			Error:  nil,
		}
	}
	if err := s.client.BatchCallContext(ctx, headerRequests); err != nil {
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
			info, err := result.Info(s.trustRPC, s.mustBePostMerge)
			if err != nil {
				return nil, fmt.Errorf("bad header data for block %s: %w", headerRequests[i].Args[0], err)
			}
			s.headersCache.Add(info.Hash(), info)
			out = append(out, info.ID())
			prev := begin
			if i > 0 {
				prev = out[i-1]
			}
			if prev.Hash != info.ParentHash() {
				return nil, fmt.Errorf("inconsistent results from L1 chain range request, block %s not expected parent %s of %s", prev, info.ParentHash(), info.ID())
			}
		} else if errors.Is(headerRequests[i].Error, ethereum.NotFound) {
			break // no more headers from here
		} else {
			return nil, fmt.Errorf("failed to retrieve block: %s: %w", headerRequests[i].Args[0], headerRequests[i].Error)
		}
	}
	return out, nil
}

func (s *EthClient) GetProof(ctx context.Context, address common.Address, blockTag string) (*eth.AccountResult, error) {
	var getProofResponse *eth.AccountResult
	err := s.client.CallContext(ctx, &getProofResponse, "eth_getProof", address, []common.Hash{}, blockTag)
	if err == nil && getProofResponse == nil {
		err = ethereum.NotFound
	}
	return getProofResponse, err
}

func (s *EthClient) Close() {
	s.client.Close()
}
