package client

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

// DefaultDialTimeout is a default timeout for dialing a client.
const DefaultDialTimeout = 1 * time.Minute
const defaultRetryCount = 30
const defaultRetryTime = 2 * time.Second

// DialEthClientWithTimeout attempts to dial the L1 provider using the provided
// URL. If the dial doesn't complete within defaultDialTimeout seconds, this
// method will return an error.
func DialEthClientWithTimeout(timeout time.Duration, log log.Logger, url string) (*ethclient.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	c, err := DialRPCClientWithBackoff(ctx, log, url)
	if err != nil {
		return nil, err
	}

	return ethclient.NewClient(c), nil
}

// Dials a JSON-RPC endpoint repeatedly, with a backoff, until a client connection is established. Auth is optional.
func DialRPCClientWithBackoff(ctx context.Context, log log.Logger, addr string) (*rpc.Client, error) {
	bOff := retry.Fixed(defaultRetryTime)
	return retry.Do(ctx, defaultRetryCount, bOff, func() (*rpc.Client, error) {
		if !IsURLAvailable(addr) {
			log.Warn("failed to dial address, but may connect later", "addr", addr)
			return nil, fmt.Errorf("address unavailable (%s)", addr)
		}
		client, err := rpc.DialOptions(ctx, addr)
		if err != nil {
			return nil, fmt.Errorf("failed to dial address (%s): %w", addr, err)
		}
		return client, nil
	})
}
