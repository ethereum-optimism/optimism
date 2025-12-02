package batching

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/hashicorp/go-multierror"

	"github.com/ethereum/go-ethereum/rpc"
)

// IterativeBatchCall batches many RPC requests with safe and easy parallelization.
// Request errors are handled and re-tried, and the batch size is configurable.
// Executing IterativeBatchCall is as simple as calling Fetch repeatedly until it returns io.EOF.
type IterativeBatchCall[K any, V any] struct {
	completed uint32       // tracks how far to completing all requests we are
	resetLock sync.RWMutex // ensures we do not concurrently read (incl. fetch) / reset

	requestsKeys []K
	batchSize    int

	makeRequest func(K) (V, rpc.BatchElem)
	getBatch    BatchCallContextFn
	getSingle   CallContextFn

	requestsValues []V
	scheduled      chan rpc.BatchElem
}

// NewIterativeBatchCall constructs a batch call, fetching the values with the given keys,
// and transforms them into a verified final result.
func NewIterativeBatchCall[K any, V any](
	requestsKeys []K,
	makeRequest func(K) (V, rpc.BatchElem),
	getBatch BatchCallContextFn,
	getSingle CallContextFn,
	batchSize int) *IterativeBatchCall[K, V] {

	if len(requestsKeys) < batchSize {
		batchSize = len(requestsKeys)
	}
	if batchSize < 1 {
		batchSize = 1
	}

	out := &IterativeBatchCall[K, V]{
		completed:    0,
		getBatch:     getBatch,
		getSingle:    getSingle,
		requestsKeys: requestsKeys,
		batchSize:    batchSize,
		makeRequest:  makeRequest,
	}
	out.Reset()
	return out
}

// Reset will clear the batch call, to start fetching all contents from scratch.
func (ibc *IterativeBatchCall[K, V]) Reset() {
	ibc.resetLock.Lock()
	defer ibc.resetLock.Unlock()

	scheduled := make(chan rpc.BatchElem, len(ibc.requestsKeys))
	requestsValues := make([]V, len(ibc.requestsKeys))
	for i, k := range ibc.requestsKeys {
		v, r := ibc.makeRequest(k)
		requestsValues[i] = v
		scheduled <- r
	}

	atomic.StoreUint32(&ibc.completed, 0)
	ibc.requestsValues = requestsValues
	ibc.scheduled = scheduled
	if len(ibc.requestsKeys) == 0 {
		close(ibc.scheduled)
	}
}

// Fetch fetches more of the data, and returns io.EOF when all data has been fetched.
// This method is safe to call concurrently; it will parallelize the fetching work.
// If no work is available, but the fetching is not done yet,
// then Fetch will block until the next thing can be fetched, or until the context expires.
func (ibc *IterativeBatchCall[K, V]) Fetch(ctx context.Context) error {
	ibc.resetLock.RLock()
	defer ibc.resetLock.RUnlock()

	// return early if context is Done
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// collect a batch from the requests channel
	batch := make([]rpc.BatchElem, 0, ibc.batchSize)
	// wait for first element
	select {
	case reqElem, ok := <-ibc.scheduled:
		if !ok { // no more requests to do
			return io.EOF
		}
		batch = append(batch, reqElem)
	case <-ctx.Done():
		return ctx.Err()
	}

	// collect more elements, if there are any.
	for {
		if len(batch) >= ibc.batchSize {
			break
		}
		select {
		case reqElem, ok := <-ibc.scheduled:
			if !ok { // no more requests to do
				return io.EOF
			}
			batch = append(batch, reqElem)
			continue
		case <-ctx.Done():
			for _, r := range batch {
				ibc.scheduled <- r
			}
			return ctx.Err()
		default:
		}
		break
	}

	if len(batch) == 0 {
		return nil
	}

	if ibc.batchSize == 1 {
		first := batch[0]
		if err := ibc.getSingle(ctx, &first.Result, first.Method, first.Args...); err != nil {
			ibc.scheduled <- first
			return err
		}
	} else {
		if err := ibc.getBatch(ctx, batch); err != nil {
			for _, r := range batch {
				ibc.scheduled <- r
			}
			return fmt.Errorf("failed batch-retrieval: %w", err)
		}
	}
	var result error
	for _, elem := range batch {
		if elem.Error != nil {
			result = multierror.Append(result, elem.Error)
			elem.Error = nil // reset, we'll try this element again
			ibc.scheduled <- elem
			continue
		} else {
			atomic.AddUint32(&ibc.completed, 1)
			if atomic.LoadUint32(&ibc.completed) >= uint32(len(ibc.requestsKeys)) {
				close(ibc.scheduled)
				return io.EOF
			}
		}
	}
	return result
}

// Complete indicates if the batch call is done.
func (ibc *IterativeBatchCall[K, V]) Complete() bool {
	ibc.resetLock.RLock()
	defer ibc.resetLock.RUnlock()
	return atomic.LoadUint32(&ibc.completed) >= uint32(len(ibc.requestsKeys))
}

// Result returns the fetched values, checked and transformed to the final output type, if available.
// If the check fails, the IterativeBatchCall will Reset itself, to be ready for a re-attempt in fetching new data.
func (ibc *IterativeBatchCall[K, V]) Result() ([]V, error) {
	ibc.resetLock.RLock()
	if atomic.LoadUint32(&ibc.completed) < uint32(len(ibc.requestsKeys)) {
		ibc.resetLock.RUnlock()
		return nil, errors.New("results not available yet, Fetch more first")
	}
	ibc.resetLock.RUnlock()
	return ibc.requestsValues, nil
}
