package l1

import (
	"context"
	"fmt"
	"io"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/hashicorp/go-multierror"
)

// IterativeBatchCall is an util to create a job to fetch many RPC requests in batches,
// and enable the caller to parallelize easily, handle and re-try error,
// and pick a batch size all by simply calling Fetch again and again until it returns io.EOF.
type IterativeBatchCall struct {
	completed uint32
	getBatch  batchCallContextFn
	requests  []rpc.BatchElem
	scheduled chan rpc.BatchElem
}

func NewIterativeBatchCall(requests []rpc.BatchElem, getBatch batchCallContextFn) *IterativeBatchCall {
	scheduled := make(chan rpc.BatchElem, len(requests))
	for _, r := range requests {
		scheduled <- r
	}
	return &IterativeBatchCall{
		completed: 0,
		getBatch:  getBatch,
		requests:  requests,
		scheduled: scheduled,
	}
}

// Fetch fetches more of the data, and returns io.EOF when all data has been fetched.
// This method is safe to call concurrently: it will parallelize the fetching work.
// If no work is available, but the fetching is not done yet,
// then Fetch will block until the next thing can be fetched, or until the context expires.
func (ibc *IterativeBatchCall) Fetch(ctx context.Context, maxBatchSize uint) error {
	// collect a batch from the requests channel
	batch := make([]rpc.BatchElem, 0, maxBatchSize)
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
		if uint(len(batch)) >= maxBatchSize {
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
			return ctx.Err()
		default:
			break
		}
		break
	}

	if err := ibc.getBatch(ctx, batch); err != nil {
		for _, r := range batch {
			ibc.scheduled <- r
		}
		return fmt.Errorf("failed batch-retrieval: %w", err)
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
			if atomic.LoadUint32(&ibc.completed) >= uint32(len(ibc.requests)) {
				close(ibc.scheduled)
				return io.EOF
			}
		}
	}
	return result
}

func (ibc *IterativeBatchCall) Complete() bool {
	return atomic.LoadUint32(&ibc.completed) >= uint32(len(ibc.requests))
}

func (ibc *IterativeBatchCall) Result() []rpc.BatchElem {
	return ibc.requests
}
