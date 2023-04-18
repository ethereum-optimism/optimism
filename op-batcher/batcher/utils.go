package batcher

import (
	"context"
	"net/http"
	"net/http/cookiejar"

	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// dialEthClientWithTimeout attempts to dial the L1 provider using the provided
// URL. If the dial doesn't complete within defaultDialTimeout seconds, this
// method will return an error.
func dialEthClientWithTimeout(ctx context.Context, url string, cookies bool, headers http.Header) (*ethclient.Client, error) {
	ctxt, cancel := context.WithTimeout(ctx, defaultDialTimeout)
	defer cancel()

	var opts []rpc.ClientOption
	if cookies {
		jar, err := cookiejar.New(nil)
		if err != nil {
			return nil, err
		}
		opts = append(opts, rpc.WithHTTPClient(&http.Client{Jar: jar}))
	}
	if headers != nil && len(headers) > 0 {
		opts = append(opts, rpc.WithHeaders(headers))
	}

	r, err := rpc.DialOptions(ctxt, url, opts...)
	if err != nil {
		return nil, err
	}

	return ethclient.NewClient(r), nil
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
