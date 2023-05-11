package client

import (
	"context"
	"time"

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
