package client

import (
	"context"
	"net/http"
	"net/http/cookiejar"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

func NewClient(ctx context.Context, cfg CLIConfig, opts ...rpc.ClientOption) (*ethclient.Client, error) {
	r, err := NewRPCClient(ctx, cfg, opts...)
	if err != nil {
		return nil, err
	}
	return ethclient.NewClient(r), nil
}

func NewRPCClient(ctx context.Context, cfg CLIConfig, opts ...rpc.ClientOption) (*rpc.Client, error) {
	if cfg.Cookies {
		jar, err := cookiejar.New(nil)
		if err != nil {
			return nil, err
		}
		opts = append(opts, rpc.WithHTTPClient(&http.Client{Jar: jar}))
	}
	if cfg.Headers != nil {
		opts = append(opts, rpc.WithHeaders(cfg.Headers))
	}

	return rpc.DialOptions(ctx, cfg.Addr, opts...)
}
