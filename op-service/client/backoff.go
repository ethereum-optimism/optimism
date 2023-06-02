package client

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/backoff"
)

var (
	// ExponentialBackoff is the default backoff strategy.
	ExponentialBackoff = backoff.Exponential()
)

// InnerRPC is a minimal [client.RPC] interface that is used by the backoff client.
//
//go:generate mockery --name InnerRPC --output ./mocks/
type InnerRPC interface {
	Close()
	CallContext(ctx context.Context, result any, method string, args ...any) error
	BatchCallContext(ctx context.Context, b []rpc.BatchElem) error
	EthSubscribe(ctx context.Context, channel any, args ...any) (ethereum.Subscription, error)
}

// backoffEthClient wraps a [InnerRPC] with a backoff strategy.
type backoffEthClient struct {
	c             InnerRPC
	retryAttempts int
	strategy      backoff.Strategy
}

// NewBackoffClient creates a new backoff client.
// The backoff strategy is optional, if not provided, the default exponential backoff strategy is used.
func NewBackoffClient(c InnerRPC, retries int, strategy ...backoff.Strategy) *backoffEthClient {
	if len(strategy) == 0 {
		strategy = []backoff.Strategy{ExponentialBackoff}
	}
	return &backoffEthClient{
		c:             c,
		retryAttempts: retries,
		strategy:      strategy[0],
	}
}

// BackoffStrategy returns the [backoff.Strategy] used by the client.
func (b *backoffEthClient) BackoffStrategy() backoff.Strategy {
	return b.strategy
}

func (b *backoffEthClient) Close() {
	b.c.Close()
}

func (b *backoffEthClient) CallContext(ctx context.Context, result any, method string, args ...any) error {
	return backoff.DoCtx(ctx, b.retryAttempts, b.strategy, func() error {
		cCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		return b.c.CallContext(cCtx, result, method, args...)
	})
}

func (b *backoffEthClient) BatchCallContext(ctx context.Context, batch []rpc.BatchElem) error {
	return backoff.DoCtx(ctx, b.retryAttempts, b.strategy, func() error {
		cCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		err := b.c.BatchCallContext(cCtx, batch)
		return err
	})
}

func (b *backoffEthClient) EthSubscribe(ctx context.Context, channel any, args ...any) (ethereum.Subscription, error) {
	var sub ethereum.Subscription
	err := backoff.DoCtx(ctx, b.retryAttempts, b.strategy, func() error {
		var err error
		sub, err = b.c.EthSubscribe(ctx, channel, args...)
		return err
	})
	return sub, err
}
