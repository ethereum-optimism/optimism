package sources

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
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
	inner              EthClientInterface
	logger             log.Logger
	PrefetchingRange   uint64
	PrefetchingTimeout time.Duration
	runningCtx         context.Context
	runningCancel      context.CancelFunc
	requestHead        uint64
	requestMu          sync.Mutex
	latestTip          uint64
	wg                 *sync.WaitGroup // used for testing
}

func (p *PrefetchingEthClient) log() log.Logger {
	return p.logger.New("tip", p.latestTip, "requestHead", p.requestHead)
}

// NewPrefetchingEthClient creates a new [PrefetchingEthClient] with the given underlying [EthClient]
// and a prefetching range.
func NewPrefetchingEthClient(inner EthClientInterface, logger log.Logger, prefetchingRange uint64, timeout time.Duration) (*PrefetchingEthClient, error) {
	tctx, tcancel := context.WithTimeout(context.Background(), time.Second)
	defer tcancel()
	tip, err := inner.InfoByLabel(tctx, eth.Unsafe)
	if err != nil {
		return nil, fmt.Errorf("Getting latest L1 head: %w", err)
	}
	// Create a new context for the prefetching goroutines
	runningCtx, runningCancel := context.WithCancel(context.Background())
	logger.Debug("Created PrefetchingEthClient",
		"range", prefetchingRange, "timeout", timeout, "tip", tip.NumberU64())
	return &PrefetchingEthClient{
		inner:              inner,
		logger:             logger,
		PrefetchingRange:   prefetchingRange,
		PrefetchingTimeout: timeout,
		runningCtx:         runningCtx,
		runningCancel:      runningCancel,
		latestTip:          tip.NumberU64(),
	}, nil
}

func (p *PrefetchingEthClient) updateRange(from uint64) (start uint64, end uint64, shouldFetch bool) {
	p.requestMu.Lock()
	defer p.requestMu.Unlock()

	// don't prefetch beyond tip
	if from >= p.latestTip {
		p.latestTip = from
		return 0, 0, false
	}

	// initialize range
	start = from + 1
	end = start + p.PrefetchingRange

	// avoid duplicate requests within current window
	if start < p.requestHead && start >= p.requestHead-p.PrefetchingRange {
		start = p.requestHead
	} else {
		p.log().Debug("Prefetching new window", "from", from, "start", start, "end", end)
	}

	// don't request beyond tip
	// TODO: need to update tip handling once prefetcher has caught up
	if p.latestTip < end {
		end = p.latestTip
	}

	// update requestHead
	if p.requestHead > end {
		p.log().Debug("Prefetching: Rewinding requestHead to end", "from", from, "start", start, "end", end)
	}
	p.requestHead = end
	return start, end, start < end
}

func (p *PrefetchingEthClient) fetchWindow(from uint64) {
	start, end, shouldFetch := p.updateRange(from)
	p.log().Debug("Prefetching window",
		"from", from, "start", start,
		"end", end, "shouldFetch", shouldFetch)
	if !shouldFetch {
		return
	}

	if p.wg != nil {
		p.wg.Add(int(end - start))
	}
	for i := start; i < end; i++ {
		go p.fetchBlockAndReceipts(i)
	}
}

func (p *PrefetchingEthClient) fetchBlockAndReceipts(number uint64) {
	if p.wg != nil {
		defer p.wg.Done()
	}
	ctx, cancel := context.WithTimeout(p.runningCtx, p.PrefetchingTimeout)
	defer cancel()
	blockInfo, _, err := p.inner.InfoAndTxsByNumber(ctx, number)
	if err != nil {
		// hack to ignore prefetching error beyond current head
		if strings.Contains(err.Error(), "block is out of range") {
			p.log().Debug("Prefetching: tried to fetch block from future", "number", number)
		} else {
			p.log().Warn("Prefetching block error", "number", number, "err", err)
		}
		return
	}
	p.log().Debug("Prefetched block", "number", number, "hash", blockInfo.Hash())
	_, rec, err := p.inner.FetchReceipts(ctx, blockInfo.Hash())
	if err != nil {
		p.log().Warn("Prefetching receipts error", "number", number, "err", err)
	} else {
		p.log().Debug("Prefetched receipts", "number", number, "receipts_count", len(rec))
	}
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
	p.fetchWindow(blockInfo.NumberU64())

	return blockInfo, nil
}

func (p *PrefetchingEthClient) InfoByNumber(ctx context.Context, number uint64) (eth.BlockInfo, error) {
	// Trigger prefetching in the background
	p.fetchWindow(number)

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
	p.fetchWindow(blockInfo.NumberU64())

	return blockInfo, nil
}

func (p *PrefetchingEthClient) InfoAndTxsByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, types.Transactions, error) {
	// Fetch the block info and transactions for the requested hash
	blockInfo, txs, err := p.inner.InfoAndTxsByHash(ctx, hash)
	if err != nil {
		return blockInfo, txs, err
	}

	// Prefetch the next n blocks and their receipts
	p.fetchWindow(blockInfo.NumberU64())

	return blockInfo, txs, nil
}

func (p *PrefetchingEthClient) InfoAndTxsByNumber(ctx context.Context, number uint64) (eth.BlockInfo, types.Transactions, error) {
	// Fetch the block info and transactions for the requested number
	blockInfo, txs, err := p.inner.InfoAndTxsByNumber(ctx, number)
	if err != nil {
		return blockInfo, txs, err
	}

	// Prefetch the next n blocks and their receipts
	p.fetchWindow(number)

	return blockInfo, txs, nil
}

func (p *PrefetchingEthClient) InfoAndTxsByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, types.Transactions, error) {
	// Fetch the block info and transactions for the requested label
	blockInfo, txs, err := p.inner.InfoAndTxsByLabel(ctx, label)
	if err != nil {
		return blockInfo, txs, err
	}

	// Prefetch the next n blocks and their receipts
	p.fetchWindow(blockInfo.NumberU64())

	return blockInfo, txs, nil
}

func (p *PrefetchingEthClient) PayloadByHash(ctx context.Context, hash common.Hash) (*eth.ExecutionPayload, error) {
	// Fetch the payload for the requested hash
	payload, err := p.inner.PayloadByHash(ctx, hash)
	if err != nil {
		return payload, err
	}

	// Prefetch the next n blocks and their receipts
	p.fetchWindow(uint64(payload.BlockNumber))

	return payload, nil
}

func (p *PrefetchingEthClient) PayloadByNumber(ctx context.Context, number uint64) (*eth.ExecutionPayload, error) {
	// Fetch the payload for the requested number
	payload, err := p.inner.PayloadByNumber(ctx, number)
	if err != nil {
		return payload, err
	}

	// Prefetch the next n blocks and their receipts
	p.fetchWindow(number)

	return payload, nil
}

func (p *PrefetchingEthClient) PayloadByLabel(ctx context.Context, label eth.BlockLabel) (*eth.ExecutionPayload, error) {
	// Fetch the payload for the requested label
	payload, err := p.inner.PayloadByLabel(ctx, label)
	if err != nil {
		return payload, err
	}

	// Prefetch the next n blocks and their receipts
	p.fetchWindow(uint64(payload.BlockNumber))

	return payload, nil
}

func (p *PrefetchingEthClient) FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error) {
	// Fetch the block info and receipts for the requested hash
	blockInfo, receipts, err := p.inner.FetchReceipts(ctx, blockHash)
	if err != nil {
		return blockInfo, receipts, err
	}

	// Prefetch the next n blocks and their receipts
	p.fetchWindow(blockInfo.NumberU64())

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
