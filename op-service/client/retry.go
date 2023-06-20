package client

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/hashicorp/go-multierror"

	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-service/backoff"
)

var (
	// ExponentialBackoff is the default backoff strategy.
	ExponentialBackoff = backoff.Exponential()
)

// retryingClient wraps a [client.RPC] with a backoff strategy.
type retryingClient struct {
	log           log.Logger
	c             client.RPC
	retryAttempts int
	strategy      backoff.Strategy
}

// NewRetryingClient creates a new retrying client.
// The backoff strategy is optional, if not provided, the default exponential backoff strategy is used.
func NewRetryingClient(logger log.Logger, c client.RPC, retries int, strategy ...backoff.Strategy) *retryingClient {
	if len(strategy) == 0 {
		strategy = []backoff.Strategy{ExponentialBackoff}
	}
	return &retryingClient{
		log:           logger,
		c:             c,
		retryAttempts: retries,
		strategy:      strategy[0],
	}
}

// BackoffStrategy returns the [backoff.Strategy] used by the client.
func (b *retryingClient) BackoffStrategy() backoff.Strategy {
	return b.strategy
}

func (b *retryingClient) Close() {
	b.c.Close()
}

func (b *retryingClient) CallContext(ctx context.Context, result any, method string, args ...any) error {
	return backoff.DoCtx(ctx, b.retryAttempts, b.strategy, func() error {
		cCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		err := b.c.CallContext(cCtx, result, method, args...)
		if err != nil {
			b.log.Warn("RPC request failed", "method", method, "err", err)
		}
		return err
	})
}

// pendingReq combines BatchElem information with the index of this request in the original []rpc.BatchElem
type pendingReq struct {
	// req is a copy of the BatchElem individual request to make.
	// It never has Result or Error set as it gets copied again as part of being passed to the underlying client.
	req rpc.BatchElem

	// idx tracks the index of the original BatchElem in the supplied input array
	// This can then be used to set the result on the original input
	idx int
}

func (b *retryingClient) BatchCallContext(ctx context.Context, input []rpc.BatchElem) error {
	// Add all BatchElem to the initial pending set
	// Each time we retry, we'll remove successful BatchElem for this list so we only retry ones that fail.
	pending := make([]*pendingReq, len(input))
	for i, req := range input {
		pending[i] = &pendingReq{
			req: req,
			idx: i,
		}
	}
	return backoff.DoCtx(ctx, b.retryAttempts, b.strategy, func() error {
		cCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()

		batch := make([]rpc.BatchElem, len(pending))
		for i, req := range pending {
			batch[i] = req.req
		}
		err := b.c.BatchCallContext(cCtx, batch)
		if err != nil {
			b.log.Warn("Batch request failed", "err", err)
			// Whole call failed, retry all pending elems again
			return err
		}
		var failed []*pendingReq
		var combinedErr error
		for i, elem := range batch {
			req := pending[i]
			idx := req.idx // Index into input of the original BatchElem

			// Set the result on the original batch to pass back to the caller in case we stop retrying
			input[idx].Error = elem.Error
			input[idx].Result = elem.Result

			// If the individual request failed, add it to the list to retry
			if elem.Error != nil {
				// Need to retry this request
				failed = append(failed, req)
				combinedErr = multierror.Append(elem.Error, combinedErr)
			}
		}
		if len(failed) > 0 {
			pending = failed
			b.log.Warn("Batch request returned errors", "err", combinedErr)
			return combinedErr
		}
		return nil
	})
}

func (b *retryingClient) EthSubscribe(ctx context.Context, channel any, args ...any) (ethereum.Subscription, error) {
	var sub ethereum.Subscription
	err := backoff.DoCtx(ctx, b.retryAttempts, b.strategy, func() error {
		var err error
		sub, err = b.c.EthSubscribe(ctx, channel, args...)
		if err != nil {
			b.log.Warn("Subscription request failed", "err", err)
		}
		return err
	})
	return sub, err
}
