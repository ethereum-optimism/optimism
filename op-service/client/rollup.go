package client

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum/go-ethereum/rpc"
)

// DialRollupClientWithTimeout attempts to dial the RPC provider using the provided
// URL. If the dial doesn't complete within defaultDialTimeout seconds, this
// method will return an error.
func DialRollupClientWithTimeout(ctx context.Context, url string, timeout time.Duration) (*sources.RollupClient, error) {
	ctxt, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	rpcCl, err := rpc.DialContext(ctxt, url)
	if err != nil {
		return nil, err
	}

	return sources.NewRollupClient(client.NewBaseRPCClient(rpcCl)), nil
}
