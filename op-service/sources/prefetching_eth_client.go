package sources

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/caching"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type PrefetchingEthClient struct {
	inner *EthClient
	// state that manages pre-fetching, like in-flight fetching jobs
}

// NewPrefetchingEthClient returns a [PrefetchingEthClient], wrapping an RPC with bindings to fetch ethereum data with added error logging,
// metric tracking, and caching. The [PrefetchingEthClient] uses concurrent RPC requests to keep a cache of data ready for callers.
func NewPrefetchingEthClient(client client.RPC, log log.Logger, metrics caching.Metrics, config *EthClientConfig) (*PrefetchingEthClient, error) {
	if err := config.Check(); err != nil {
		return nil, fmt.Errorf("bad config, cannot create L1 source: %w", err)
	}
	client = LimitRPC(client, config.MaxConcurrentRequests)
	inner := &EthClient{
		client:                  client,
		maxBatchSize:            config.MaxRequestsPerBatch,
		trustRPC:                config.TrustRPC,
		mustBePostMerge:         config.MustBePostMerge,
		provKind:                config.RPCProviderKind,
		log:                     log,
		receiptsCache:           caching.NewLRUCache[common.Hash, *receiptsFetchingJob](metrics, "receipts", config.ReceiptsCacheSize),
		transactionsCache:       caching.NewLRUCache[common.Hash, types.Transactions](metrics, "txs", config.TransactionsCacheSize),
		headersCache:            caching.NewLRUCache[common.Hash, eth.BlockInfo](metrics, "headers", config.HeadersCacheSize),
		payloadsCache:           caching.NewLRUCache[common.Hash, *eth.ExecutionPayload](metrics, "payloads", config.PayloadsCacheSize),
		availableReceiptMethods: AvailableReceiptsFetchingMethods(config.RPCProviderKind),
		lastMethodsReset:        time.Now(),
		methodResetDuration:     config.MethodResetDuration,
		rethDbPath:              config.RethDBPath,
	}
	return &PrefetchingEthClient{
		inner: inner,
	}, nil
}

func (p *PrefetchingEthClient) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	return p.inner.SubscribeNewHead(ctx, ch)
}

func (p *PrefetchingEthClient) ChainID(ctx context.Context) (*big.Int, error) {
	return p.inner.ChainID(ctx)
}

func (p *PrefetchingEthClient) InfoByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, error) {
	return p.inner.InfoByHash(ctx, hash)
}

func (p *PrefetchingEthClient) InfoByNumber(ctx context.Context, number uint64) (eth.BlockInfo, error) {
	return p.inner.InfoByNumber(ctx, number)
}

func (p *PrefetchingEthClient) InfoByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, error) {
	return p.inner.InfoByLabel(ctx, label)
}

func (p *PrefetchingEthClient) InfoAndTxsByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, types.Transactions, error) {
	return p.inner.InfoAndTxsByHash(ctx, hash)
}

func (p *PrefetchingEthClient) InfoAndTxsByNumber(ctx context.Context, number uint64) (eth.BlockInfo, types.Transactions, error) {
	return p.inner.InfoAndTxsByNumber(ctx, number)
}

func (p *PrefetchingEthClient) InfoAndTxsByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, types.Transactions, error) {
	return p.inner.InfoAndTxsByLabel(ctx, label)
}

func (p *PrefetchingEthClient) PayloadByHash(ctx context.Context, hash common.Hash) (*eth.ExecutionPayload, error) {
	return p.inner.PayloadByHash(ctx, hash)
}

func (p *PrefetchingEthClient) PayloadByNumber(ctx context.Context, number uint64) (*eth.ExecutionPayload, error) {
	return p.inner.PayloadByNumber(ctx, number)
}

func (p *PrefetchingEthClient) PayloadByLabel(ctx context.Context, label eth.BlockLabel) (*eth.ExecutionPayload, error) {
	return p.inner.PayloadByLabel(ctx, label)
}

func (p *PrefetchingEthClient) FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error) {
	return p.inner.FetchReceipts(ctx, blockHash)
}

func (p *PrefetchingEthClient) GetProof(ctx context.Context, address common.Address, storage []common.Hash, blockTag string) (*eth.AccountResult, error) {
	return p.inner.GetProof(ctx, address, storage, blockTag)
}

func (p *PrefetchingEthClient) GetStorageAt(ctx context.Context, address common.Address, storageSlot common.Hash, blockTag string) (common.Hash, error) {
	return p.inner.GetStorageAt(ctx, address, storageSlot, blockTag)
}

func (p *PrefetchingEthClient) ReadStorageAt(ctx context.Context, address common.Address, storageSlot common.Hash, blockHash common.Hash) (common.Hash, error) {
	return p.inner.ReadStorageAt(ctx, address, storageSlot, blockHash)
}

func (p *PrefetchingEthClient) Close() {
	p.inner.Close()
}
