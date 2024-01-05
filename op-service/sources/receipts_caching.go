package sources

import (
	"context"
	"sync"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/caching"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// A CachingReceiptsProvider caches successful receipt fetches from the inner
// ReceiptsProvider. It also avoids duplicate in-flight requests per block hash.
type CachingReceiptsProvider struct {
	inner ReceiptsProvider
	cache *caching.LRUCache[common.Hash, types.Receipts]

	// lock fetching process for each block hash to avoid duplicate requests
	fetching   map[common.Hash]*sync.Mutex
	fetchingMu sync.Mutex // only protects map
}

func NewCachingReceiptsProvider(inner ReceiptsProvider, m caching.Metrics, cacheSize int) *CachingReceiptsProvider {
	return &CachingReceiptsProvider{
		inner:    inner,
		cache:    caching.NewLRUCache[common.Hash, types.Receipts](m, "receipts", cacheSize),
		fetching: make(map[common.Hash]*sync.Mutex),
	}
}

func NewCachingRPCReceiptsProvider(client rpcClient, log log.Logger, config RPCReceiptsConfig, m caching.Metrics, cacheSize int) *CachingReceiptsProvider {
	return NewCachingReceiptsProvider(NewRPCReceiptsFetcher(client, log, config), m, cacheSize)
}

func (p *CachingReceiptsProvider) getOrCreateFetchingLock(blockHash common.Hash) *sync.Mutex {
	p.fetchingMu.Lock()
	defer p.fetchingMu.Unlock()
	if mu, ok := p.fetching[blockHash]; ok {
		return mu
	}
	mu := new(sync.Mutex)
	p.fetching[blockHash] = mu
	return mu
}

func (p *CachingReceiptsProvider) deleteFetchingLock(blockHash common.Hash) {
	p.fetchingMu.Lock()
	defer p.fetchingMu.Unlock()
	delete(p.fetching, blockHash)
}

func (p *CachingReceiptsProvider) FetchReceipts(ctx context.Context, block eth.BlockID, txHashes []common.Hash) (types.Receipts, error) {
	if r, ok := p.cache.Get(block.Hash); ok {
		return r, nil
	}

	mu := p.getOrCreateFetchingLock(block.Hash)
	mu.Lock()
	defer mu.Unlock()
	// Other routine might have fetched in the meantime
	if r, ok := p.cache.Get(block.Hash); ok {
		// we might have created a new lock above while the old
		// fetching job completed.
		p.deleteFetchingLock(block.Hash)
		return r, nil
	}

	r, err := p.inner.FetchReceipts(ctx, block, txHashes)
	if err != nil {
		return nil, err
	}
	p.cache.Add(block.Hash, r)
	// result now in cache, can delete fetching lock
	p.deleteFetchingLock(block.Hash)
	return r, nil
}
