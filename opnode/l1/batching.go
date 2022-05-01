package l1

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	TooManyRetries = errors.New("too many retries")
)

// parallelBatchCall creates a drop-in replacement for the standard batchCallContextFn that splits requests into more batch requests, and will parallelize and retry as configured.
func parallelBatchCall(log log.Logger, getBatch batchCallContextFn, maxRetry int, maxPerBatch int, maxParallel int) batchCallContextFn {
	return func(ctx context.Context, requests []rpc.BatchElem) error {
		return fetchBatched(ctx, log, requests, getBatch, maxRetry, maxPerBatch, maxParallel)
	}
}

type batchResult struct {
	failed  []rpc.BatchElem // if anything has to be retried
	err     error           // if the batch as a whole failed
	success int             // amount of items that completed successfully
}

// fetchBatched fetches the given requests in batches of at most maxPerBatch elements, and with at most maxRetry retries per batch.
// Batch requests may be split into maxParallel go-routines.
// Retries only apply to individual request errors, not to the outer batch-requests that combine them into batches.
func fetchBatched(ctx context.Context, log log.Logger, requests []rpc.BatchElem, getBatch batchCallContextFn, maxRetry int, maxPerBatch int, maxParallel int) error {
	batchRequest := func(ctx context.Context, missing []rpc.BatchElem) (failed []rpc.BatchElem, err error) {
		if err := getBatch(ctx, missing); err != nil {
			return nil, fmt.Errorf("failed batch-retrieval: %w", err)
		}
		for _, elem := range missing {
			if elem.Error != nil {
				log.Trace("batch request element failed", "err", elem.Error, "elem", elem.Args[0])
				elem.Error = nil // reset, we'll try this element again
				failed = append(failed, elem)
				continue
			}
		}
		return failed, nil
	}

	// limit capacity, don't write to underlying array on retries
	requests = requests[:len(requests):len(requests)]

	expectedBatches := (len(requests) + maxPerBatch - 1) / maxPerBatch

	// don't need more go-routines than requests
	if maxParallel > expectedBatches {
		maxParallel = expectedBatches
	}

	// capacity is sufficient for no go-routine to get stuck on writing
	completed := make(chan batchResult, maxParallel)

	// queue of tasks for worker go-routines
	batchRequests := make(chan []rpc.BatchElem, maxParallel)
	defer close(batchRequests)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// starts worker go-routines. Closed when task channel closes
	for i := 0; i < maxParallel; i++ {
		go func(ctx context.Context) {
			for {
				batch, ok := <-batchRequests
				if !ok {
					return // no more batches left
				}
				failed, err := batchRequest(ctx, batch)
				completed <- batchResult{failed: failed, err: err, success: len(batch) - len(failed)}
			}
		}(ctx)
	}

	parallelRequests := func() int {
		// we split the requests into parallel batch requests, and count how many
		i := 0
		for ; i < maxParallel && len(requests) > 0; i++ {
			nextBatch := requests
			if len(nextBatch) > maxPerBatch {
				nextBatch = requests[:maxPerBatch]
			}
			// don't retry this batch of requests again, unless we add them back
			requests = requests[len(nextBatch):]

			// schedule the batch, this may block if all workers are busy and the queue is full
			batchRequests <- nextBatch
		}
		return i
	}

	maxCount := expectedBatches * maxRetry

	awaited := len(requests)

	// start initial round of parallel requests
	count := parallelRequests()

	// We slow down additional batch requests to not spam the server.
	retryTicker := time.NewTicker(time.Millisecond * 20)
	defer retryTicker.Stop()

	// The main requests slice is only ever mutated by the go-routine running this loop.
	// Slices of this are sent to worker go-routines, and never overwritten with different requests.
	for {
		// check if we've all results back successfully
		if awaited <= 0 {
			return nil
		}
		if count > maxCount {
			return TooManyRetries
		}
		select {
		case <-retryTicker.C:
			count += parallelRequests() // retry batch-requests on interval
		case result := <-completed:
			if result.err != nil {
				// batch failed, RPC may be broken, abort
				return fmt.Errorf("batch request failed: %w", result.err)
			}
			// if any element failed, add it to the requests for re-attempt
			requests = append(requests, result.failed...)
			awaited -= result.success
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
