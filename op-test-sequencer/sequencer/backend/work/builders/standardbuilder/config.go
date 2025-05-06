package standardbuilder

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/closer"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
)

type Config struct {
	// L1 execution-layer RPC endpoint
	L1EL endpoint.MustRPC `yaml:"l1EL"`

	// L2 execution-layer RPC endpoint
	L2EL endpoint.MustRPC `yaml:"l2EL"`
	// L2 consensus-layer RPC endpoint
	L2CL endpoint.MustRPC `yaml:"l2CL"`
}

func (c *Config) Start(ctx context.Context, id seqtypes.BuilderID, opts *work.ServiceOpts) (work.Builder, error) {
	onClose := closer.CloseFn(func() {
		opts.Log.Info("Closed")
	})
	cancelCloseEarly, closeEarly := onClose.Maybe()
	defer closeEarly()

	l2CLRPCClient, err := client.NewRPC(ctx, opts.Log, c.L2CL.Value.RPC(), client.WithLazyDial())
	if err != nil {
		return nil, err
	}
	onClose.Stack(l2CLRPCClient.Close)
	l2ELRPCClient, err := client.NewRPC(ctx, opts.Log, c.L2EL.Value.RPC(), client.WithLazyDial())
	if err != nil {
		return nil, err
	}
	onClose.Stack(l2ELRPCClient.Close)
	l1ELRPCClient, err := client.NewRPC(ctx, opts.Log, c.L1EL.Value.RPC(), client.WithLazyDial())
	if err != nil {
		return nil, err
	}
	onClose.Stack(l1ELRPCClient.Close)

	rolCl := sources.NewRollupClient(l2CLRPCClient)
	cfg, err := retry.Do(ctx, 10, retry.Exponential(), func() (*rollup.Config, error) {
		opts.Log.Info("Fetching rollup-config for block-building")
		cfg, err := rolCl.RollupConfig(ctx)
		if err != nil {
			opts.Log.Warn("Failed to fetch rollup-config", "err", err)
		}
		return cfg, err
	})
	if err != nil {
		return nil, err
	}

	l1Cl, err := sources.NewL1Client(l1ELRPCClient, opts.Log, nil,
		sources.L1ClientSimpleConfig(false, sources.RPCKindStandard, 10))
	if err != nil {
		return nil, err
	}
	l2Cl, err := sources.NewL2Client(l2ELRPCClient, opts.Log, nil,
		sources.L2ClientSimpleConfig(cfg, false, 10, 10))
	if err != nil {
		return nil, err
	}
	fb := derive.NewFetchingAttributesBuilder(cfg, l1Cl, l2Cl)

	fb.TestSkipL1OriginCheck()

	cl := sources.NewOPStackClient(l2CLRPCClient)
	cancelCloseEarly()
	return &Builder{
		id:       id,
		log:      opts.Log,
		m:        opts.Metrics,
		registry: opts.Jobs,
		l1:       l1Cl,
		l2:       l2Cl,
		attrPrep: fb,
		cl:       cl,
		onClose:  onClose,
	}, nil
}
