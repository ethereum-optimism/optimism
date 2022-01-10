package l1

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	lru "github.com/hashicorp/golang-lru"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

type blockAndReceipts struct {
	// Track if we finished (with error or not). May increment >1 when other sub-tasks fail.
	// First field, aligned atomic changes.
	finished uint32

	// Count receipts to track status
	// First field of struct for memory aligned atomic access
	DownloadedReceipts uint32

	Block *types.Block
	// allocated in advance, one for each transaction, nil until downloaded
	Receipts []*types.Receipt

	// stop fetching if this context is dead
	ctx context.Context

	// for other duplicate requests to get the result
	feed event.Feed
}

// wrappedErr wraps an error, since event.Feed cannot handle nil errors otherwise (reflection on nil)
type wrappedErr struct {
	error
}

func (bl *blockAndReceipts) Finish(err wrappedErr) {
	if atomic.AddUint32(&bl.finished, 1) == 1 {
		bl.feed.Send(err)
	}
}

type receiptTask struct {
	BlockHash common.Hash
	TxHash    common.Hash
	TxIndex   uint64
	// Count the attempts we made to fetch this receipt. Block as a whole fails if we tried to many times.
	Retry uint64
	// Avoid concurrent Downloader cache access and pruning edge cases with receipts
	// Keep a pointer to insert the receipt at
	Dest *blockAndReceipts
}

type Downloader interface {
	Fetch(ctx context.Context, id eth.BlockID) (*types.Block, []*types.Receipt, error)
	AddReceiptWorkers(n int) int
}

type DownloadSource interface {
	eth.BlockByHashSource
	eth.ReceiptSource
}

type downloader struct {
	// cache of ongoing/completed block tasks: block hash -> block
	cacheEvict *lru.Cache
	cacheLock  sync.Mutex

	receiptTasks       chan *receiptTask
	receiptWorkers     []ethereum.Subscription
	receiptWorkersLock sync.Mutex

	chr DownloadSource
}

var downloadEvictedErr = errors.New("evicted")

func NewDownloader(chr DownloadSource) Downloader {
	dl := &downloader{
		receiptTasks: make(chan *receiptTask, 100),
		chr:          chr,
	}
	evict := func(k, v interface{}) {
		// stop downloading things if they were evicted (already finished items are unaffected)
		v.(*blockAndReceipts).Finish(wrappedErr{downloadEvictedErr})
	}
	// 500 at 100 KB each would be 50 MB of memory for the L1 block inputs cache
	dl.cacheEvict, _ = lru.NewWithEvict(500, evict)
	return dl
}

func (l1t *downloader) Fetch(ctx context.Context, id eth.BlockID) (*types.Block, []*types.Receipt, error) {
	// check if we are already working on it
	l1t.cacheLock.Lock()

	var bnr *blockAndReceipts
	if bnrIfc, ok := l1t.cacheEvict.Get(id.Hash); ok {
		bnr = bnrIfc.(*blockAndReceipts)
		l1t.cacheEvict.Add(id.Hash, bnr) // add it again, so it moves to the front and avoid eviction
	} else {
		bnr = &blockAndReceipts{ctx: ctx}
		l1t.cacheEvict.Add(id.Hash, bnr)

		// pull the block in the background
		go func() {
			ctx, cancel := context.WithTimeout(ctx, time.Second*10)
			defer cancel()
			bl, err := l1t.chr.BlockByHash(ctx, id.Hash)
			if err != nil {
				bnr.Finish(wrappedErr{fmt.Errorf("failed to download block %s: %v", id.Hash, err)})
				return
			}

			txs := bl.Transactions()
			bnr.Block = bl
			bnr.Receipts = make([]*types.Receipt, len(txs))

			for i, tx := range txs {
				l1t.receiptTasks <- &receiptTask{BlockHash: id.Hash, TxHash: tx.Hash(), TxIndex: uint64(i), Dest: bnr}
			}

			// no receipts to fetch? Then we are done!
			if len(txs) == 0 {
				bnr.Finish(wrappedErr{nil})
			}
		}()
	}
	l1t.cacheLock.Unlock()

	ch := make(chan wrappedErr)
	sub := bnr.feed.Subscribe(ch)
	select {
	case wErr := <-ch:
		if wErr.error != nil {
			return nil, nil, wErr.error
		}
		return bnr.Block, bnr.Receipts, nil
	case err := <-sub.Err():
		return nil, nil, err
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}
}

const maxReceiptRetry = 5

func (l1t *downloader) newReceiptWorker() ethereum.Subscription {
	return event.NewSubscription(func(quit <-chan struct{}) error {
		for {
			select {
			case task := <-l1t.receiptTasks:
				// scheduled tasks may be stale if other receipts of the block failed too many times
				if task.Dest.finished > 0 { // no need for locks, a very rare stale download does not hurt
					continue
				}
				// limit fetching to the task as a whole, and constrain to 10 seconds for receipt itself
				ctx, cancel := context.WithTimeout(task.Dest.ctx, time.Second*10)
				defer cancel()
				receipt, err := l1t.chr.TransactionReceipt(ctx, task.TxHash)
				if err != nil {
					// if a single receipt fails out of the whole block, we can retry a few times.
					if task.Retry >= maxReceiptRetry {
						// Failed to get the receipt too many times, block fails!
						task.Dest.Finish(wrappedErr{fmt.Errorf("failed to download receipt again, and reached max %d retries: %v", maxReceiptRetry, err)})
						continue
					} else {
						task.Retry += 1
						select {
						case l1t.receiptTasks <- task:
							// all good, retry scheduled successfully
						default:
							// failed to schedule, too much receipt work, stop block to relieve pressure.
							task.Dest.Finish(wrappedErr{fmt.Errorf("receipt downloader too busy, not downloading receipt again (%d retries): %v", task.Retry, err)})
							continue
						}
						continue
					}
				}
				task.Dest.Receipts[task.TxIndex] = receipt
				// We count the receipts we have so far (atomic, avoid parallel counting race condition)
				total := atomic.AddUint32(&task.Dest.DownloadedReceipts, 1)
				if total == uint32(len(task.Dest.Receipts)) {
					// block completed without error!
					task.Dest.Finish(wrappedErr{nil})
					continue
				}
				// task completed, but block is not complete without other receipt tasks finishing first
			case <-quit:
				return nil
			}
		}
	})
}

// AddReceiptWorkers can add or remove (negative value) worker routines to parallelize receipt downloads with.
// It returns the number of active workers.
func (l1t *downloader) AddReceiptWorkers(n int) int {
	l1t.receiptWorkersLock.Lock()
	defer l1t.receiptWorkersLock.Unlock()
	if n < 0 {
		for i := 0; i < -n && len(l1t.receiptWorkers) > 0; i++ {
			last := len(l1t.receiptWorkers) - 1
			l1t.receiptWorkers[last].Unsubscribe()
			l1t.receiptWorkers = l1t.receiptWorkers[:last]
		}
	}
	for i := 0; i < n; i++ {
		l1t.receiptWorkers = append(l1t.receiptWorkers, l1t.newReceiptWorker())
	}
	return len(l1t.receiptWorkers)
}
