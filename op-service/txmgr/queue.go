package txmgr

import (
	"context"
	"math"
	"sync"

	"github.com/ethereum/go-ethereum/core/types"
	"golang.org/x/sync/errgroup"
)

type TxReceipt[T any] struct {
	// ID can be used to identify unique tx receipts within the receipt channel
	ID T
	// Receipt result from the transaction send
	Receipt *types.Receipt
	// Err contains any error that occurred during the tx send
	Err error
}

type Queue[T any] struct {
	ctx        context.Context
	txMgr      TxManager
	maxPending uint64
	groupLock  sync.Mutex
	groupCtx   context.Context
	group      *errgroup.Group
}

// NewQueue creates a new transaction sending Queue, with the following parameters:
//   - ctx: runtime context of the queue. If canceled, all ongoing send processes are canceled.
//   - txMgr: transaction manager to use for transaction sending
//   - maxPending: max number of pending txs at once (0 == no limit)
func NewQueue[T any](ctx context.Context, txMgr TxManager, maxPending uint64) *Queue[T] {
	if maxPending > math.MaxInt {
		// ensure we don't overflow as errgroup only accepts int; in reality this will never be an issue
		maxPending = math.MaxInt
	}
	return &Queue[T]{
		ctx:        ctx,
		txMgr:      txMgr,
		maxPending: maxPending,
	}
}

// Wait waits for all pending txs to complete (or fail).
func (q *Queue[T]) Wait() {
	if q.group == nil {
		return
	}
	_ = q.group.Wait()
}

// Send will wait until the number of pending txs is below the max pending,
// and then send the next tx.
//
// The actual tx sending is non-blocking, with the receipt returned on the
// provided receipt channel. If the channel is unbuffered, the goroutine is
// blocked from completing until the channel is read from.
func (q *Queue[T]) Send(id T, candidate TxCandidate, receiptCh chan TxReceipt[T]) {
	group, ctx := q.groupContext()
	group.Go(func() error {
		return q.sendTx(ctx, id, candidate, receiptCh)
	})
}

// TrySend sends the next tx, but only if the number of pending txs is below the
// max pending.
//
// Returns false if there is no room in the queue to send. Otherwise, the
// transaction is queued and this method returns true.
//
// The actual tx sending is non-blocking, with the receipt returned on the
// provided receipt channel. If the channel is unbuffered, the goroutine is
// blocked from completing until the channel is read from.
func (q *Queue[T]) TrySend(id T, candidate TxCandidate, receiptCh chan TxReceipt[T]) bool {
	group, ctx := q.groupContext()
	return group.TryGo(func() error {
		return q.sendTx(ctx, id, candidate, receiptCh)
	})
}

func (q *Queue[T]) sendTx(ctx context.Context, id T, candidate TxCandidate, receiptCh chan TxReceipt[T]) error {
	receipt, err := q.txMgr.Send(ctx, candidate)
	receiptCh <- TxReceipt[T]{
		ID:      id,
		Receipt: receipt,
		Err:     err,
	}
	return err
}

// groupContext returns a Group and a Context to use when sending a tx.
//
// If any of the pending transactions returned an error, the queue's shared error Group is
// canceled. This method will wait on that Group for all pending transactions to return,
// and create a new Group with the queue's global context as its parent.
func (q *Queue[T]) groupContext() (*errgroup.Group, context.Context) {
	q.groupLock.Lock()
	defer q.groupLock.Unlock()
	if q.groupCtx == nil || q.groupCtx.Err() != nil {
		// no group exists, or the existing context has an error, so we need to wait
		// for existing group threads to complete (if any) and create a new group
		if q.group != nil {
			_ = q.group.Wait()
		}
		q.group, q.groupCtx = errgroup.WithContext(q.ctx)
		if q.maxPending > 0 {
			q.group.SetLimit(int(q.maxPending))
		}
	}
	return q.group, q.groupCtx
}
