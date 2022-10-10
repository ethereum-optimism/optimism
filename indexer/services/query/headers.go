package query

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/backoff"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// HeaderByNumberWithRetry retries getting headers.
func HeaderByNumberWithRetry(ctx context.Context, client *ethclient.Client) (*types.Header, error) {
	var res *types.Header
	err := backoff.DoCtx(ctx, 3, backoff.Exponential(), func() error {
		var err error
		res, err = client.HeaderByNumber(ctx, nil)
		return err
	})
	return res, err
}
