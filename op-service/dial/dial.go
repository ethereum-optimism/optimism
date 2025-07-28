package dial

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

// DefaultDialTimeout is a default timeout for dialing a client.
const DefaultDialTimeout = 1 * time.Minute
const defaultRetryCount = 30
const defaultRetryTime = 2 * time.Second
const defaultConnectTimeout = 10 * time.Second

// DialEthClientWithTimeout attempts to dial the L1 provider using the provided
// URL. If the dial doesn't complete within defaultDialTimeout seconds, this
// method will return an error.
func DialEthClientWithTimeout(ctx context.Context, timeout time.Duration, log log.Logger, url string) (*ethclient.Client, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	c, err := dialRPCClientWithBackoff(ctx, log, url)
	if err != nil {
		return nil, err
	}

	return ethclient.NewClient(c), nil
}

// DialRollupClientWithTimeout attempts to dial the RPC provider using the provided URL.
// The timeout and retry logic is handled internally by the client.
func DialRollupClientWithTimeout(ctx context.Context, log log.Logger, url string, callerOpts ...client.RPCOption) (*sources.RollupClient, error) {
	opts := []client.RPCOption{
		client.WithFixedDialBackoff(defaultRetryTime),
		client.WithDialAttempts(defaultRetryCount),
		client.WithConnectTimeout(defaultConnectTimeout),
	}
	opts = append(opts, callerOpts...)

	rpcCl, err := client.NewRPC(ctx, log, url, opts...)
	if err != nil {
		return nil, err
	}

	return sources.NewRollupClient(rpcCl), nil
}

// DialRPCClientWithTimeout attempts to dial the RPC provider using the provided URL.
// The timeout and retry logic is handled internally by the client.
func DialRPCClientWithTimeout(ctx context.Context, log log.Logger, url string, opts ...rpc.ClientOption) (*rpc.Client, error) {
	return dialRPCClientWithBackoff(ctx, log, url, opts...)
}

// Dials a JSON-RPC endpoint repeatedly, with a backoff, until a client connection is established. Auth is optional.
func dialRPCClientWithBackoff(ctx context.Context, log log.Logger, addr string, opts ...rpc.ClientOption) (*rpc.Client, error) {
	bOff := retry.Fixed(defaultRetryTime)
	return retry.Do(ctx, defaultRetryCount, bOff, func() (*rpc.Client, error) {
		return dialRPCClient(ctx, log, addr, opts...)
	})
}

// Dials a JSON-RPC endpoint once.
func dialRPCClient(ctx context.Context, log log.Logger, addr string, opts ...rpc.ClientOption) (*rpc.Client, error) {
	return client.CheckAndDial(ctx, log, addr, defaultConnectTimeout, opts...)
}
