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

const (
	fetchBlockTimeout   = 10 * time.Second
	fetchReceiptTimeout = 10 * time.Second
	maxReceiptRetry     = 5

	// 500 at 100 KB each would be 50 MB of memory for the L1 block inputs cache
	cacheSize = 500

	// Amount of receipt tasks to buffer before applying back-pressure (blocking new block-requests)
	receiptQueueSize = 100
)

type downloadTask struct {
	// Incremented when Finish is called, to check whether the task already completed (with or without error).
	// May increment >1 when other sub-tasks fail.
	//
	// First field, aligned atomic changes.
	finished uint32

	// Count already downloaded receipts to track status
	//
	// Aligned after above field, atomic changes.
	downloadedReceipts uint32

	block *types.Block
	// receipts slice is allocated in advance, one slot for each transaction, nil until downloaded
	receipts []*types.Receipt

	// stop fetching if this context is dead
	ctx context.Context

	// feed to subscribe the requests to (de-duplicate work)
	feed event.Feed
}

// wrappedErr wraps an error, since event.Feed cannot handle nil errors otherwise (reflection on nil)
type wrappedErr struct {
	error
}

func (bl *downloadTask) Finish(err wrappedErr) {
	if atomic.AddUint32(&bl.finished, 1) == 1 {
		bl.feed.Send(err)
	}
}

type receiptTask struct {
	blockHash common.Hash
	txHash    common.Hash
	txIndex   uint64
	// Count the attempts we made to fetch this receipt. Block as a whole fails if we tried too many times.
	retry uint64
	// Avoid concurrent Downloader cache access and pruning edge cases with receipts.
	// Keep a pointer to the parent task to insert the receipt into.
	dest *downloadTask
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

	src DownloadSource
}

var downloadEvictedErr = errors.New("evicted")

func NewDownloader(dlSource DownloadSource) Downloader {
	dl := &downloader{
		receiptTasks: make(chan *receiptTask, receiptQueueSize),
		src:          dlSource,
	}
	evict := func(k, v interface{}) {
		// stop downloading things if they were evicted (already finished items are unaffected)
		v.(*downloadTask).Finish(wrappedErr{downloadEvictedErr})
	}
	dl.cacheEvict, _ = lru.NewWithEvict(cacheSize, evict)
	return dl
}

func (dl *downloader) Fetch(ctx context.Context, id eth.BlockID) (*types.Block, []*types.Receipt, error) {
	// check if we are already working on it
	dl.cacheLock.Lock()

	var dlTask *downloadTask
	if dlTaskIfc, ok := dl.cacheEvict.Get(id.Hash); ok {
		dlTask = dlTaskIfc.(*downloadTask)
		dl.cacheEvict.Add(id.Hash, dlTask) // add it again, so it moves to the front and avoid eviction
	} else {
		dlTask = &downloadTask{ctx: ctx}
		dl.cacheEvict.Add(id.Hash, dlTask)

		// pull the block in the background
		go func() {
			ctx, cancel := context.WithTimeout(ctx, fetchBlockTimeout)
			defer cancel()
			bl, err := dl.src.BlockByHash(ctx, id.Hash)
			if err != nil {
				dlTask.Finish(wrappedErr{fmt.Errorf("failed to download block %s: %v", id.Hash, err)})
				return
			}

			txs := bl.Transactions()
			dlTask.block = bl
			dlTask.receipts = make([]*types.Receipt, len(txs))

			for i, tx := range txs {
				dl.receiptTasks <- &receiptTask{blockHash: id.Hash, txHash: tx.Hash(), txIndex: uint64(i), dest: dlTask}
			}

			// no receipts to fetch? Then we are done!
			if len(txs) == 0 {
				dlTask.Finish(wrappedErr{nil})
			}
		}()
	}
	dl.cacheLock.Unlock()

	ch := make(chan wrappedErr)
	sub := dlTask.feed.Subscribe(ch)
	select {
	case wErr := <-ch:
		if wErr.error != nil {
			return nil, nil, wErr.error
		}
		return dlTask.block, dlTask.receipts, nil
	case err := <-sub.Err():
		return nil, nil, err
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}
}

func (dl *downloader) processTask(task *receiptTask) {
	// scheduled tasks may be stale if other receipts of the block failed too many times
	if task.dest.finished > 0 { // no need for locks, a very rare stale download does not hurt
		return
	}
	// stop fetching when the task is cancelled or when the individual receipt times out
	ctx, cancel := context.WithTimeout(task.dest.ctx, fetchReceiptTimeout)
	defer cancel()
	receipt, err := dl.src.TransactionReceipt(ctx, task.txHash)
	if err != nil {
		// if a single receipt fails out of the whole block, we can retry a few times.
		if task.retry >= maxReceiptRetry {
			// Failed to get the receipt too many times, block fails!
			task.dest.Finish(wrappedErr{fmt.Errorf("failed to download receipt again, and reached max %d retries: %v", maxReceiptRetry, err)})
			return
		} else {
			task.retry += 1
			select {
			case dl.receiptTasks <- task:
				// all good, retry scheduled successfully
				return
			default:
				// failed to schedule, too much receipt work, stop block to relieve pressure.
				task.dest.Finish(wrappedErr{fmt.Errorf("receipt downloader too busy, not downloading receipt again (%d retries): %v", task.retry, err)})
				return
			}
		}
	}
	task.dest.receipts[task.txIndex] = receipt
	// We count the receipts we have so far (atomic, avoid parallel counting race condition)
	total := atomic.AddUint32(&task.dest.downloadedReceipts, 1)
	if total == uint32(len(task.dest.receipts)) {
		// block completed without error!
		task.dest.Finish(wrappedErr{nil})
		return
	}
	// task completed, but no Finish call without other receipt tasks finishing first
}

func (dl *downloader) newReceiptWorker() ethereum.Subscription {
	return event.NewSubscription(func(quit <-chan struct{}) error {
		for {
			select {
			case task := <-dl.receiptTasks:
				dl.processTask(task)
			case <-quit:
				return nil
			}
		}
	})
}

// AddReceiptWorkers can add or remove (negative value) worker routines to parallelize receipt downloads with.
// It returns the number of active workers.
func (dl *downloader) AddReceiptWorkers(n int) int {
	dl.receiptWorkersLock.Lock()
	defer dl.receiptWorkersLock.Unlock()
	if n < 0 {
		for i := 0; i < -n && len(dl.receiptWorkers) > 0; i++ {
			last := len(dl.receiptWorkers) - 1
			dl.receiptWorkers[last].Unsubscribe()
			dl.receiptWorkers = dl.receiptWorkers[:last]
		}
	}
	for i := 0; i < n; i++ {
		dl.receiptWorkers = append(dl.receiptWorkers, dl.newReceiptWorker())
	}
	return len(dl.receiptWorkers)
}
