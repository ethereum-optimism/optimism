package txmgr

import (
	"context"
	"math"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/core/types"
	"golang.org/x/sync/errgroup"
)

type TxReceipt[T any] struct {
	// ID can be used to identify unique tx receipts within the recept channel
	ID T
	// Receipt result from the transaction send
	Receipt *types.Receipt
	// Err contains any error that occurred during the tx send
	Err error
}

// TxFactory should return the next transaction to send (and associated identifier).
// If no transaction is available, an error should be returned (such as io.EOF).
type TxFactory[T any] func(ctx context.Context) (TxCandidate, T, error)

type Queue[T any] struct {
	txMgr          TxManager
	maxPending     uint64
	pendingChanged func(uint64)
	pending        atomic.Uint64
	groupLock      sync.Mutex
	groupCtx       context.Context
	group          *errgroup.Group
}

// NewQueue creates a new transaction sending Queue, with the following parameters:
//   - maxPending: max number of pending txs at once (0 == no limit)
//   - pendingChanged: called whenever a tx send starts or finishes. The
//     number of currently pending txs is passed as a parameter.
func NewQueue[T any](txMgr TxManager, maxPending uint64, pendingChanged func(uint64)) *Queue[T] {
	if maxPending > math.MaxInt {
		// ensure we don't overflow as errgroup only accepts int; in reality this will never be an issue
		maxPending = math.MaxInt
	}
	return &Queue[T]{
		txMgr:          txMgr,
		maxPending:     maxPending,
		pendingChanged: pendingChanged,
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
// and then send the next tx. The TxFactory should return an error if the
// next tx does not exist, which will be returned from this method.
//
// The actual tx sending is non-blocking, with the receipt returned on the
// provided receipt channel.
func (q *Queue[T]) Send(ctx context.Context, factory TxFactory[T], receiptCh chan TxReceipt[T]) error {
	ctx, cancel := q.mergeWithGroupContext(ctx)
	factoryErrCh := make(chan error)
	q.group.Go(func() error {
		defer cancel()
		return q.sendTx(ctx, factory, factoryErrCh, receiptCh)
	})
	return <-factoryErrCh
}

// TrySend sends the next tx, but only if the number of pending txs is below the
// max pending, otherwise the TxFactory is not called (and no error is returned).
// The TxFactory should return an error if the next tx does not exist, which is
// returned from this method.
//
// Returns false if there is no room in the queue to send. Otherwise, the
// transaction is queued and this method returns true.
//
// The actual tx sending is non-blocking, with the receipt returned on the
// provided receipt channel.
func (q *Queue[T]) TrySend(ctx context.Context, factory TxFactory[T], receiptCh chan TxReceipt[T]) (bool, error) {
	ctx, cancel := q.mergeWithGroupContext(ctx)
	factoryErrCh := make(chan error)
	started := q.group.TryGo(func() error {
		defer cancel()
		return q.sendTx(ctx, factory, factoryErrCh, receiptCh)
	})
	if !started {
		cancel()
		return false, nil
	}
	err := <-factoryErrCh
	return err != nil, err
}

func (q *Queue[T]) sendTx(ctx context.Context, factory TxFactory[T], factoryErrorCh chan error, receiptCh chan TxReceipt[T]) error {
	candidate, id, err := factory(ctx)
	factoryErrorCh <- err
	if err != nil {
		// Factory returned an error which was returned in the channel. This means
		// there is no tx to send, so return nil.
		return nil
	}

	q.pendingChanged(q.pending.Add(1))
	defer func() {
		q.pendingChanged(q.pending.Add(^uint64(0))) // -1
	}()
	receipt, err := q.txMgr.Send(ctx, candidate)
	receiptCh <- TxReceipt[T]{
		ID:      id,
		Receipt: receipt,
		Err:     err,
	}
	return err
}

// mergeWithGroupContext creates a new Context that is canceled if either the given context is
// Done, or the group context is canceled. The returned CancelFunc should be called once finished.
//
// If the group context doesn't exist or has already been canceled, a new one is created after
// waiting for existing group threads to complete.
func (q *Queue[T]) mergeWithGroupContext(ctx context.Context) (context.Context, context.CancelFunc) {
	q.groupLock.Lock()
	defer q.groupLock.Unlock()
	if q.groupCtx == nil || q.groupCtx.Err() != nil {
		// no group exists, or the existing context has an error, so we need to wait
		// for existing group threads to complete (if any) and create a new group
		q.Wait()
		q.group, q.groupCtx = errgroup.WithContext(context.Background())
		if q.maxPending > 0 {
			q.group.SetLimit(int(q.maxPending))
		} else {
			q.group.SetLimit(-1)
		}
	}

	ctx, cancel := context.WithCancel(ctx)
	cl := make(chan struct{})
	groupContext := q.groupCtx
	go func() {
		defer cancel()
		select {
		case <-groupContext.Done():
		case <-cl:
		}
	}()
	return ctx, func() {
		close(cl)
	}
}
