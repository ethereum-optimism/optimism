package txmgr

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/core/types"
)

type TxReceipt[T any] struct {
	Data    T
	Receipt *types.Receipt
	Err     error
}

type TxFactory[T any] func(ctx context.Context) (*TxCandidate, T, error)

type Queue[T any] struct {
	txMgr          TxManager
	maxPending     uint64
	pendingChanged func(uint64)
	pending        uint64
	cond           *sync.Cond
	wg             sync.WaitGroup
}

// NewQueue creates a new transaction sending Queue, with the following parameters:
//   - maxPending: max number of pending txs at once (0 == no limit)
//   - pendingChanged: called whenever a job starts or finishes. The
//     number of currently pending txs is passed as a parameter.
func NewQueue[T any](txMgr TxManager, maxPending uint64, pendingChanged func(uint64)) *Queue[T] {
	return &Queue[T]{
		txMgr:          txMgr,
		maxPending:     maxPending,
		pendingChanged: pendingChanged,
		cond:           sync.NewCond(&sync.Mutex{}),
	}
}

// Wait waits on all running jobs to stop.
func (q *Queue[T]) Wait() {
	q.wg.Wait()
}

// Send will wait until the number of pending txs is below the max pending,
// and then send the next tx. The TxFactory should return `nil` if the next
// tx does not exist. Returns the error returned from the TxFactory (if any).
func (q *Queue[T]) Send(ctx context.Context, factory TxFactory[T], receiptCh chan TxReceipt[T]) error {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	for q.full() {
		q.cond.Wait()
	}
	return q.trySend(ctx, factory, receiptCh)
}

// TrySend sends the next tx, but only if the number of pending txs is below the
// max pending, otherwise the TxFactory is not called (and nil is returned).
//
// The TxFactory should return `nil` if the next tx does not exist. Returns
// the error returned from the TxFactory (if any).
func (q *Queue[T]) TrySend(ctx context.Context, factory TxFactory[T], receiptCh chan TxReceipt[T]) error {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	return q.trySend(ctx, factory, receiptCh)
}

func (q *Queue[T]) trySend(ctx context.Context, factory TxFactory[T], receiptCh chan TxReceipt[T]) error {
	if q.full() {
		return nil
	}
	candidate, data, err := factory(ctx)
	if err != nil {
		return err
	}
	if candidate == nil {
		return nil
	}

	q.pending++
	q.pendingChanged(q.pending)
	q.wg.Add(1)
	go func() {
		defer func() {
			q.cond.L.Lock()
			q.pending--
			q.pendingChanged(q.pending)
			q.wg.Done()
			q.cond.L.Unlock()
			q.cond.Broadcast()
		}()
		receipt, err := q.txMgr.Send(ctx, *candidate)
		receiptCh <- TxReceipt[T]{
			Data:    data,
			Receipt: receipt,
			Err:     err,
		}
	}()
	return nil
}

func (q *Queue[T]) full() bool {
	return q.maxPending > 0 && q.pending >= q.maxPending
}
