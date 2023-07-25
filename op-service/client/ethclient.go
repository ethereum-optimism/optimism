package client

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/backoff"
	"github.com/ethereum/go-ethereum/ethclient"
)

// DialEthClientWithTimeout attempts to dial the L1 provider using the provided
// URL. If the dial doesn't complete within defaultDialTimeout seconds, this
// method will return an error.
func DialEthClientWithTimeout(ctx context.Context, url string, timeout time.Duration) (*ethclient.Client, error) {
	ctxt, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return ethclient.DialContext(ctxt, url)
}

// DialEthClientWithRetry attempts to dial the L1 provider using the provided
// URL. If the dial fails, this method will retry the dial every interval seconds
// until the context is timed out.
func DialEthClientWithRetry(ctx context.Context, url string, strategy backoff.Strategy, maxAttempts int, timeout time.Duration) (*ethclient.Client, error) {
	operation := func() (*ethclient.Client, error) {
		ctxt, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		return ethclient.DialContext(ctxt, url)
	}

	return backoff.DoResult(maxAttempts, strategy, operation)
}
