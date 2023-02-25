package batcher

import (
	"context"

	"github.com/btcsuite/btcd/rpcclient"
	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// @DEV BEDROCK ADD BTC CLIENT HERE

// dialEthClientWithTimeout attempts to dial the L1 provider using the provided
// URL. If the dial doesn't complete within defaultDialTimeout seconds, this
// method will return an error.
func dialEthClientWithTimeout(ctx context.Context, url string) (*ethclient.Client, error) {

	ctxt, cancel := context.WithTimeout(ctx, defaultDialTimeout)
	defer cancel()

	return ethclient.DialContext(ctxt, url)
}

// dialRollupClientWithTimeout attempts to dial the RPC provider using the provided
// URL. If the dial doesn't complete within defaultDialTimeout seconds, this
// method will return an error.
func dialRollupClientWithTimeout(ctx context.Context, url string) (*sources.RollupClient, error) {
	ctxt, cancel := context.WithTimeout(ctx, defaultDialTimeout)
	defer cancel()

	rpcCl, err := rpc.DialContext(ctxt, url)
	if err != nil {
		return nil, err
	}

	return sources.NewRollupClient(client.NewBaseRPCClient(rpcCl)), nil

}

// @DEV BEDROCK ADD BTC CLIENT HERE
func dialBTCClientWithoutTimeout(url string) (*rpcclient.Client, error) {

	connCfg := &rpcclient.ConnConfig{
		Host:         url,
		User:         "test",
		Pass:         "test",
		HTTPPostMode: true,
		DisableTLS:   false,
	}

	return rpcclient.New(connCfg, nil)
}
