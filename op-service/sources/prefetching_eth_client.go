package sources

import (
	"context"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type PrefetchingEthClient struct {
	inner            EthClient
	prefetchingRange uint64
	// other state fields for managing prefetching
}

// NewPrefetchingEthClient creates a new [PrefetchingEthClient] with the given underlying [EthClient]
// and a prefetching range.
func NewPrefetchingEthClient(inner EthClient, prefetchingRange uint64) (*PrefetchingEthClient, error) {
	return &PrefetchingEthClient{
		inner:            inner,
		prefetchingRange: prefetchingRange,
	}, nil
}

func (p *PrefetchingEthClient) FetchWindow(ctx context.Context, start, end uint64) {
	for i := start; i <= end; i++ {
		// Ignoring the error and result as this is just prefetching
		// The actual fetching and error handling will be done when the data is requested
		go p.FetchBlockAndReceipts(ctx, i)
	}
}

func (p *PrefetchingEthClient) FetchBlockAndReceipts(ctx context.Context, number uint64) {
	// Ignoring the error as this is just prefetching
	// The actual fetching and error handling will be done when the data is requested
	blockInfo, _ := p.inner.InfoByNumber(ctx, number)
	// Now that we have the block, fetch its receipts
	// Again, ignore error and result as this is just prefetching
	p.inner.FetchReceipts(ctx, blockInfo.Hash())
}

func (p *PrefetchingEthClient) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	return p.inner.SubscribeNewHead(ctx, ch)
}

func (p *PrefetchingEthClient) ChainID(ctx context.Context) (*big.Int, error) {
	return p.inner.ChainID(ctx)
}

func (p *PrefetchingEthClient) InfoByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, error) {
	// Fetch the block information for the given hash
	blockInfo, err := p.inner.InfoByHash(ctx, hash)
	if err != nil {
		return blockInfo, err
	}

	// Prefetch the next n blocks and their receipts starting from the block number of the fetched block
	go p.FetchWindow(ctx, blockInfo.NumberU64()+1, blockInfo.NumberU64()+p.prefetchingRange)

	return blockInfo, nil
}

func (p *PrefetchingEthClient) InfoByNumber(ctx context.Context, number uint64) (eth.BlockInfo, error) {
	// Trigger prefetching in the background
	go p.FetchWindow(ctx, number+1, number+p.prefetchingRange)

	// Fetch the requested block
	return p.inner.InfoByNumber(ctx, number)
}

func (p *PrefetchingEthClient) InfoByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, error) {
	// Fetch the block information for the given label
	blockInfo, err := p.inner.InfoByLabel(ctx, label)
	if err != nil {
		return blockInfo, err
	}

	// Prefetch the next n blocks and their receipts starting from the block number of the fetched block
	go p.FetchWindow(ctx, blockInfo.NumberU64()+1, blockInfo.NumberU64()+p.prefetchingRange)

	return blockInfo, nil
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
	// First, fetch the requested block's receipts
	blockInfo, receipts, err := p.inner.FetchReceipts(ctx, blockHash)
	if err != nil {
		return blockInfo, receipts, err
	}

	// Now prefetch the next n blocks and their receipts
	go p.FetchWindow(ctx, blockInfo.NumberU64(), blockInfo.NumberU64()+p.prefetchingRange)

	return blockInfo, receipts, nil
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
