package query

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/backoff"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// HeaderByNumberWithRetry retries getting headers.
func HeaderByNumberWithRetry(ctx context.Context, client *ethclient.Client) (*types.Header, error) {
	return backoff.Do(ctx, 3, backoff.Exponential(), func() (*types.Header, error) {
		return client.HeaderByNumber(ctx, nil)
	})
}
