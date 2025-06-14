package config

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/opnv2/flags"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/closer"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
)

type L2ELsSetup interface {
	Check() error
	Engines(ctx context.Context, logger log.Logger, m opmetrics.RPCMetricer) ([]client.RPC, error)
	ReadELs(ctx context.Context, logger log.Logger, m opmetrics.RPCMetricer) ([]client.RPC, error)
}

type L2ELsConfig struct {
	L2EngineAddrs      []string
	L2EngineJWTSecrets []string
	L2EngineRpcTimeout time.Duration
	L2ReadAddrs        []string
}

var _ L2ELsSetup = (*L2ELsConfig)(nil)

func (cfg *L2ELsConfig) Check() error {
	if cfg.L2EngineRpcTimeout <= 0 {
		return fmt.Errorf("timeout for L2 Engine RPC requests cannot be 0 or less, but got: %s", cfg.L2EngineRpcTimeout)
	}
	if len(cfg.L2EngineAddrs)+len(cfg.L2ReadAddrs) == 0 {
		return errors.New("need at least one L2 engine RPC or read RPC address, but got none")
	}
	if len(cfg.L2EngineAddrs) > 0 &&
		(len(cfg.L2EngineJWTSecrets) == 1 ||
			len(cfg.L2EngineJWTSecrets) == len(cfg.L2EngineAddrs)) {
		return fmt.Errorf("have %d execution engine RPCs, but %d engine JWT secrets, need matching number of secrets or shared secret",
			len(cfg.L2EngineAddrs), len(cfg.L2EngineJWTSecrets))
	}
	return nil
}

func (cfg *L2ELsConfig) Engines(ctx context.Context, logger log.Logger, m opmetrics.RPCMetricer) ([]client.RPC, error) {
	var out []client.RPC
	// If one of the RPCs fails to be set up, then close everything we have open so far.
	closeFn := closer.CloseFn(func() {
		for _, cl := range out {
			cl.Close()
		}
	})
	cancel, closeEarly := closeFn.Maybe()
	defer cancel()
	for i, endpoint := range cfg.L2EngineAddrs {
		rpcClient, err := client.NewRPC(ctx, logger, endpoint,
			client.WithLazyDial(), // we lazy-dial, if we have fallbacks we don't want to block on an unavailable RPC.
			client.WithCallTimeout(cfg.L2EngineRpcTimeout),
			client.WithRPCRecorder(m.NewRecorder(fmt.Sprintf("l2-engine-api-%d", i))),
		)
		if err != nil {
			closeEarly()
			return nil, fmt.Errorf("failed to open L2 execution-engine RPC: %w", err)
		}
		out = append(out, rpcClient)
	}
	return out, nil
}

func (cfg *L2ELsConfig) ReadELs(ctx context.Context, logger log.Logger, m opmetrics.RPCMetricer) ([]client.RPC, error) {
	var out []client.RPC
	// If one of the RPCs fails to be set up, then close everything we have open so far.
	closeFn := closer.CloseFn(func() {
		for _, cl := range out {
			cl.Close()
		}
	})
	cancel, closeEarly := closeFn.Maybe()
	defer cancel()
	for i, endpoint := range cfg.L2ReadAddrs {
		rpcClient, err := client.NewRPC(ctx, logger, endpoint,
			client.WithLazyDial(), // we lazy-dial, if we have fallbacks we don't want to block on an unavailable RPC.
			client.WithRPCRecorder(m.NewRecorder(fmt.Sprintf("l2-read-api-%d", i))),
		)
		if err != nil {
			closeEarly()
			return nil, fmt.Errorf("failed to open L2 read-only RPC: %w", err)
		}
		out = append(out, rpcClient)
	}
	return out, nil
}

func L2EndpointsConfigFromCLI(ctx *cli.Context) *L2ELsConfig {
	return &L2ELsConfig{
		L2EngineAddrs:      filterEmpty(ctx.StringSlice(flags.L2EngineAddrs.Name)),
		L2EngineJWTSecrets: filterEmpty(ctx.StringSlice(flags.L2EngineJWTSecrets.Name)),
		L2EngineRpcTimeout: ctx.Duration(flags.L2EngineRpcTimeout.Name),
		L2ReadAddrs:        filterEmpty(ctx.StringSlice(flags.L2ReadAddrs.Name)),
	}
}

// filterEmpty cleans empty entries from a string-slice flag,
// which has the potential to have empty strings.
func filterEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
