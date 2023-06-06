package client

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-service/backoff"
)

var (
	// ExponentialBackoff is the default backoff strategy.
	ExponentialBackoff = backoff.Exponential()
)

// retryingClient wraps a [client.RPC] with a backoff strategy.
type retryingClient struct {
	c             client.RPC
	retryAttempts int
	strategy      backoff.Strategy
}

// NewRetryingClient creates a new retrying client.
// The backoff strategy is optional, if not provided, the default exponential backoff strategy is used.
func NewRetryingClient(c client.RPC, retries int, strategy ...backoff.Strategy) *retryingClient {
	if len(strategy) == 0 {
		strategy = []backoff.Strategy{ExponentialBackoff}
	}
	return &retryingClient{
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
		return b.c.CallContext(cCtx, result, method, args...)
	})
}

func (b *retryingClient) BatchCallContext(ctx context.Context, batch []rpc.BatchElem) error {
	return backoff.DoCtx(ctx, b.retryAttempts, b.strategy, func() error {
		cCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		err := b.c.BatchCallContext(cCtx, batch)
		return err
	})
}

func (b *retryingClient) EthSubscribe(ctx context.Context, channel any, args ...any) (ethereum.Subscription, error) {
	var sub ethereum.Subscription
	err := backoff.DoCtx(ctx, b.retryAttempts, b.strategy, func() error {
		var err error
		sub, err = b.c.EthSubscribe(ctx, channel, args...)
		return err
	})
	return sub, err
}
