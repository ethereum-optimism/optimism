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
	pendingChanged func(uint64)
	pending        atomic.Uint64
	lock           sync.Mutex
	ctx            context.Context
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
	group, cxt := errgroup.WithContext(context.Background())
	if maxPending > 0 {
		group.SetLimit(int(maxPending))
	} else {
		group.SetLimit(-1)
	}
	return &Queue[T]{
		txMgr:          txMgr,
		pendingChanged: pendingChanged,
		ctx:            cxt,
		group:          group,
	}
}

// Wait waits for all pending txs to complete (or fail).
func (q *Queue[T]) Wait() {
	_ = q.group.Wait()
}

// Send will wait until the number of pending txs is below the max pending,
// and then send the next tx. The TxFactory should return an error if the
// next tx does not exist, which will be returned from this method.
//
// The actual tx sending is non-blocking, with the receipt returned on the
// provided receipt channel.
func (q *Queue[T]) Send(ctx context.Context, factory TxFactory[T], receiptCh chan TxReceipt[T]) error {
	ctx, cancel := mergeContexts(ctx, q.ctx)
	defer cancel()
	factoryErrCh := make(chan error)
	q.group.Go(func() error {
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
	ctx, cancel := mergeContexts(ctx, q.ctx)
	defer cancel()
	factoryErrCh := make(chan error)
	started := q.group.TryGo(func() error {
		return q.sendTx(ctx, factory, factoryErrCh, receiptCh)
	})
	if !started {
		return false, nil
	}
	err := <-factoryErrCh
	return err != nil, err
}

func (q *Queue[T]) sendTx(ctx context.Context, factory TxFactory[T], factoryErrorCh chan error, receiptCh chan TxReceipt[T]) error {
	// lock to prevent concurrent access to the tx factory
	q.lock.Lock()
	defer q.lock.Unlock()

	candidate, id, err := factory(ctx)
	factoryErrorCh <- err
	if err != nil {
		// Factory returned an error which was returned in the channel. This means
		// there was no tx to send, so return nil.
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

// mergeContexts creates a new Context that is canceled if either of the two
// contexts are closed. The CancelFunc should be called once finished.
func mergeContexts(ctx1 context.Context, ctx2 context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx1)
	cl := make(chan struct{})
	go func() {
		defer cancel()
		select {
		case <-ctx2.Done():
		case <-cl:
		}
	}()
	return ctx, func() {
		close(cl)
	}
}
