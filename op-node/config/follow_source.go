package config

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/client"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum/log"
)

type L2FollowSourceEndpointSetup interface {
	// Setup a RPC client to a L2 execution engine to follow.
	Setup(ctx context.Context, log log.Logger, rollupCfg *rollup.Config, metrics opmetrics.RPCMetricer) (client.RPC, *sources.L2ClientConfig, error)
	Check() error
}

type L2FollowSourceConfig struct {
	L2RPCAddr        string
	L2RPCCallTimeout time.Duration
}

var _ L2FollowSourceEndpointSetup = (*L2FollowSourceConfig)(nil)

func (cfg *L2FollowSourceConfig) Check() error {
	if cfg.L2RPCAddr == "" {
		return errors.New("empty L2 RPC Address")
	}
	return nil
}

func (cfg *L2FollowSourceConfig) Setup(ctx context.Context, log log.Logger, rollupCfg *rollup.Config, metrics opmetrics.RPCMetricer) (client.RPC, *sources.L2ClientConfig, error) {
	if err := cfg.Check(); err != nil {
		return nil, nil, err
	}
	opts := []client.RPCOption{
		client.WithDialAttempts(10),
		client.WithCallTimeout(cfg.L2RPCCallTimeout),
		client.WithRPCRecorder(metrics.NewRecorder("follow-source-api")),
	}
	l2Node, err := client.NewRPC(ctx, log, cfg.L2RPCAddr, opts...)
	if err != nil {
		return nil, nil, err
	}

	return l2Node, sources.L2ClientDefaultConfig(rollupCfg, true), nil
}
