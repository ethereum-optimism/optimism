package sources

import (
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type EthClientInterface interface {
	SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error)
	ChainID(ctx context.Context) (*big.Int, error)
	InfoByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, error)
	InfoByNumber(ctx context.Context, number uint64) (eth.BlockInfo, error)
	InfoByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, error)
	InfoAndTxsByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, types.Transactions, error)
	InfoAndTxsByNumber(ctx context.Context, number uint64) (eth.BlockInfo, types.Transactions, error)
	InfoAndTxsByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, types.Transactions, error)
	PayloadByHash(ctx context.Context, hash common.Hash) (*eth.ExecutionPayload, error)
	PayloadByNumber(ctx context.Context, number uint64) (*eth.ExecutionPayload, error)
	PayloadByLabel(ctx context.Context, label eth.BlockLabel) (*eth.ExecutionPayload, error)
	FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error)
	GetProof(ctx context.Context, address common.Address, storage []common.Hash, blockTag string) (*eth.AccountResult, error)
	GetStorageAt(ctx context.Context, address common.Address, storageSlot common.Hash, blockTag string) (common.Hash, error)
	ReadStorageAt(ctx context.Context, address common.Address, storageSlot common.Hash, blockHash common.Hash) (common.Hash, error)
	Close()
}

type PrefetchingEthClient struct {
	inner                 EthClientInterface
	PrefetchingRange      uint64
	PrefetchingTimeout    time.Duration
	runningCtx            context.Context
	runningCancel         context.CancelFunc
	highestHeadRequesting uint64
	highestHeadLock       sync.Mutex
	wg                    *sync.WaitGroup // used for testing
}

// NewPrefetchingEthClient creates a new [PrefetchingEthClient] with the given underlying [EthClient]
// and a prefetching range.
func NewPrefetchingEthClient(inner EthClientInterface, prefetchingRange uint64, timeout time.Duration) (*PrefetchingEthClient, error) {
	// Create a new context for the prefetching goroutines
	runningCtx, runningCancel := context.WithCancel(context.Background())
	return &PrefetchingEthClient{
		inner:                 inner,
		PrefetchingRange:      prefetchingRange,
		PrefetchingTimeout:    timeout,
		runningCtx:            runningCtx,
		runningCancel:         runningCancel,
		highestHeadRequesting: 0,
	}, nil
}

func (p *PrefetchingEthClient) updateRequestingHead(start, end uint64) (newStart uint64, shouldFetch bool) {
	// Acquire lock before reading/updating highestHeadRequesting
	p.highestHeadLock.Lock()
	defer p.highestHeadLock.Unlock()
	if start <= p.highestHeadRequesting {
		start = p.highestHeadRequesting + 1
	}
	if p.highestHeadRequesting < end {
		p.highestHeadRequesting = end
	}
	return start, start <= end
}

func (p *PrefetchingEthClient) FetchWindow(start, end uint64) {
	if p.wg != nil {
		defer p.wg.Done()
	}

	start, shouldFetch := p.updateRequestingHead(start, end)
	if !shouldFetch {
		return
	}

	ctx, cancel := context.WithTimeout(p.runningCtx, p.PrefetchingTimeout)
	defer cancel()
	for i := start; i <= end; i++ {
		p.FetchBlockAndReceipts(ctx, i)
	}
}

func (p *PrefetchingEthClient) FetchBlockAndReceipts(ctx context.Context, number uint64) {
	blockInfo, _, err := p.inner.InfoAndTxsByNumber(ctx, number)
	if err != nil {
		return
	}
	_, _, _ = p.inner.FetchReceipts(ctx, blockInfo.Hash())
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

	if p.wg != nil {
		p.wg.Add(1)
	}
	// Prefetch the next n blocks and their receipts starting from the block number of the fetched block
	go p.FetchWindow(blockInfo.NumberU64()+1, blockInfo.NumberU64()+p.PrefetchingRange)

	return blockInfo, nil
}

func (p *PrefetchingEthClient) InfoByNumber(ctx context.Context, number uint64) (eth.BlockInfo, error) {
	if p.wg != nil {
		p.wg.Add(1)
	}
	// Trigger prefetching in the background
	go p.FetchWindow(number+1, number+p.PrefetchingRange)

	// Fetch the requested block
	return p.inner.InfoByNumber(ctx, number)
}

func (p *PrefetchingEthClient) InfoByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, error) {
	// Fetch the block information for the given label
	blockInfo, err := p.inner.InfoByLabel(ctx, label)
	if err != nil {
		return blockInfo, err
	}

	if p.wg != nil {
		p.wg.Add(1)
	}
	// Prefetch the next n blocks and their receipts starting from the block number of the fetched block
	go p.FetchWindow(blockInfo.NumberU64()+1, blockInfo.NumberU64()+p.PrefetchingRange)

	return blockInfo, nil
}

func (p *PrefetchingEthClient) InfoAndTxsByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, types.Transactions, error) {
	// Fetch the block info and transactions for the requested hash
	blockInfo, txs, err := p.inner.InfoAndTxsByHash(ctx, hash)
	if err != nil {
		return blockInfo, txs, err
	}

	if p.wg != nil {
		p.wg.Add(1)
	}
	// Prefetch the next n blocks and their receipts
	go p.FetchWindow(blockInfo.NumberU64()+1, blockInfo.NumberU64()+p.PrefetchingRange)

	return blockInfo, txs, nil
}

func (p *PrefetchingEthClient) InfoAndTxsByNumber(ctx context.Context, number uint64) (eth.BlockInfo, types.Transactions, error) {
	// Fetch the block info and transactions for the requested number
	blockInfo, txs, err := p.inner.InfoAndTxsByNumber(ctx, number)
	if err != nil {
		return blockInfo, txs, err
	}

	if p.wg != nil {
		p.wg.Add(1)
	}
	// Prefetch the next n blocks and their receipts
	go p.FetchWindow(number+1, number+p.PrefetchingRange)

	return blockInfo, txs, nil
}

func (p *PrefetchingEthClient) InfoAndTxsByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, types.Transactions, error) {
	// Fetch the block info and transactions for the requested label
	blockInfo, txs, err := p.inner.InfoAndTxsByLabel(ctx, label)
	if err != nil {
		return blockInfo, txs, err
	}

	if p.wg != nil {
		p.wg.Add(1)
	}
	// Prefetch the next n blocks and their receipts
	go p.FetchWindow(blockInfo.NumberU64()+1, blockInfo.NumberU64()+p.PrefetchingRange)

	return blockInfo, txs, nil
}

func (p *PrefetchingEthClient) PayloadByHash(ctx context.Context, hash common.Hash) (*eth.ExecutionPayload, error) {
	// Fetch the payload for the requested hash
	payload, err := p.inner.PayloadByHash(ctx, hash)
	if err != nil {
		return payload, err
	}

	if p.wg != nil {
		p.wg.Add(1)
	}
	// Prefetch the next n blocks and their receipts
	go p.FetchWindow(uint64(payload.BlockNumber)+1, uint64(payload.BlockNumber)+p.PrefetchingRange)

	return payload, nil
}

func (p *PrefetchingEthClient) PayloadByNumber(ctx context.Context, number uint64) (*eth.ExecutionPayload, error) {
	// Fetch the payload for the requested number
	payload, err := p.inner.PayloadByNumber(ctx, number)
	if err != nil {
		return payload, err
	}

	if p.wg != nil {
		p.wg.Add(1)
	}
	// Prefetch the next n blocks and their receipts
	go p.FetchWindow(number+1, number+p.PrefetchingRange)

	return payload, nil
}

func (p *PrefetchingEthClient) PayloadByLabel(ctx context.Context, label eth.BlockLabel) (*eth.ExecutionPayload, error) {
	// Fetch the payload for the requested label
	payload, err := p.inner.PayloadByLabel(ctx, label)
	if err != nil {
		return payload, err
	}

	if p.wg != nil {
		p.wg.Add(1)
	}
	// Prefetch the next n blocks and their receipts
	go p.FetchWindow(uint64(payload.BlockNumber)+1, uint64(payload.BlockNumber)+p.PrefetchingRange)

	return payload, nil
}

func (p *PrefetchingEthClient) FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error) {
	// Fetch the block info and receipts for the requested hash
	blockInfo, receipts, err := p.inner.FetchReceipts(ctx, blockHash)
	if err != nil {
		return blockInfo, receipts, err
	}

	if p.wg != nil {
		p.wg.Add(1)
	}
	// Prefetch the next n blocks and their receipts
	go p.FetchWindow(blockInfo.NumberU64(), blockInfo.NumberU64()+p.PrefetchingRange)

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
	p.runningCancel()
	p.inner.Close()
}
