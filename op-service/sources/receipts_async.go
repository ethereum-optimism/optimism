package sources

import (
	"context"
	"sync"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

type receiptsFetchFn func(context.Context, common.Hash) (eth.BlockInfo, optypes.Receipts, error)

type receiptsRequest struct {
	done chan struct{}
	info eth.BlockInfo
	recs optypes.Receipts
	err  error
}

// asyncReceiptsFetcher owns block-receipt fetches independently of callers.
// Requests for the same block hash share one in-flight fetch. A caller may stop
// waiting without cancelling the fetch for other callers or future cache users.
type asyncReceiptsFetcher struct {
	ctx    context.Context
	cancel context.CancelFunc
	fetch  receiptsFetchFn

	mu       sync.Mutex
	requests map[common.Hash]*receiptsRequest
	closed   bool
	wg       sync.WaitGroup
}

func newAsyncReceiptsFetcher(fetch receiptsFetchFn) *asyncReceiptsFetcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &asyncReceiptsFetcher{
		ctx:      ctx,
		cancel:   cancel,
		fetch:    fetch,
		requests: make(map[common.Hash]*receiptsRequest),
	}
}

// Prefetch starts fetching the block and its receipts, if they are not already
// being fetched. It does not wait for the result.
func (f *asyncReceiptsFetcher) Prefetch(hash common.Hash) {
	f.getOrStart(hash)
}

func (f *asyncReceiptsFetcher) Fetch(ctx context.Context, hash common.Hash) (eth.BlockInfo, optypes.Receipts, error) {
	req := f.getOrStart(hash)
	if req == nil {
		return nil, nil, context.Canceled
	}
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	case <-req.done:
		return req.info, req.recs, req.err
	}
}

func (f *asyncReceiptsFetcher) getOrStart(hash common.Hash) *receiptsRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return nil
	}
	if req, ok := f.requests[hash]; ok {
		return req
	}
	req := &receiptsRequest{done: make(chan struct{})}
	f.requests[hash] = req
	f.wg.Add(1)
	go f.run(hash, req)
	return req
}

func (f *asyncReceiptsFetcher) run(hash common.Hash, req *receiptsRequest) {
	defer f.wg.Done()
	req.info, req.recs, req.err = f.fetch(f.ctx, hash)
	close(req.done)

	f.mu.Lock()
	if f.requests[hash] == req {
		delete(f.requests, hash)
	}
	f.mu.Unlock()
}

func (f *asyncReceiptsFetcher) Close() {
	f.mu.Lock()
	if f.closed {
		f.mu.Unlock()
		return
	}
	f.closed = true
	f.cancel()
	f.mu.Unlock()
	f.wg.Wait()
}
