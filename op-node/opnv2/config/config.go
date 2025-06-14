package config

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-node/opnv2/flags"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	p2pcli "github.com/ethereum-optimism/optimism/op-node/p2p/cli"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
)

var (
	ErrMissingDependencySet   = errors.New("must specify a dependency set source")
	ErrMissingRollupConfigSet = errors.New("must specify a rollup config set source")
	ErrMissingDatadir         = errors.New("must specify datadir")
)

type Config struct {
	Version string

	LogConfig     oplog.CLIConfig
	MetricsConfig opmetrics.CLIConfig
	PprofConfig   oppprof.CLIConfig
	RPC           oprpc.CLIConfig

	L1     config.L1EndpointSetup
	Beacon config.L1BeaconEndpointSetup

	L1ConfDepth uint64

	L2 L2ELsSetup

	// TODO: sequencer config / state-persistence omitted for now
	// TODO: conductor config omitted for now
	// TODO: no p2p signer setup for now

	P2P p2p.SetupP2P

	RollupConfigSetSource depset.RollupConfigSetSourceV2
	DependencySetSource   depset.DependencySetSource

	// SynchronousProcessors disables background-workers,
	// requiring manual triggers for the backend to process anything.
	SynchronousProcessors bool

	Datadir string

	// Cancel to request a premature shutdown of the node itself, e.g. when halting. This may be nil.
	Cancel context.CancelCauseFunc
}

func (c *Config) Check() error {
	var result error
	result = errors.Join(result, c.MetricsConfig.Check())
	result = errors.Join(result, c.PprofConfig.Check())
	result = errors.Join(result, c.RPC.Check())
	if c.DependencySetSource == nil {
		result = errors.Join(result, ErrMissingDependencySet)
	}
	if c.RollupConfigSetSource == nil {
		result = errors.Join(result, ErrMissingRollupConfigSet)
	}
	if c.Datadir == "" {
		result = errors.Join(result, ErrMissingDatadir)
	}
	if c.P2P != nil { // p2p is optional
		if err := c.P2P.Check(); err != nil {
			result = errors.Join(result, fmt.Errorf("p2p config error: %w", err))
		}
	}
	if c.L1 == nil {
		result = errors.New("missing L1 Eth RPC endpoint config")
	} else if err := c.L1.Check(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to validate L1 EL RPC config: %w", err))
	}
	if c.Beacon == nil { // used to not be required in op-node V1
		result = errors.Join(result, errors.New("missing L1 beacon endpoint config"))
	} else if err := c.Beacon.Check(); err != nil {
		result = errors.Join(result, fmt.Errorf("misconfigured L1 Beacon API endpoint: %w", err))
	}
	if c.L2 == nil {
		result = errors.New("missing L2 Execution Layer endpoints config")
	} else if err := c.L2.Check(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to validate L2 engine/read-EL RPC config: %w", err))
	}
	return result
}

func FromCLI(ctx *cli.Context, version string) (*Config, error) {
	out := &Config{
		Version:     version,
		LogConfig:   oplog.ReadCLIConfig(ctx),
		PprofConfig: oppprof.ReadCLIConfig(ctx),
		// op-node does not use opmetrics.ReadCLIConfig, or oprpc.ReadCLIConfig,
		// need to investigate if there are differences.
		MetricsConfig: opmetrics.CLIConfig{
			Enabled:    ctx.Bool(flags.MetricsEnabledFlag.Name),
			ListenAddr: ctx.String(flags.MetricsAddrFlag.Name),
			ListenPort: ctx.Int(flags.MetricsPortFlag.Name),
		},
		RPC: oprpc.CLIConfig{
			ListenAddr:  ctx.String(flags.RPCListenAddr.Name),
			ListenPort:  ctx.Int(flags.RPCListenPort.Name),
			EnableAdmin: ctx.Bool(flags.RPCEnableAdmin.Name),
		},
		L1:          L1EndpointConfigFromCLI(ctx),
		Beacon:      BeaconEndpointConfigFromCLI(ctx),
		L1ConfDepth: ctx.Uint64(flags.L1VerifierConfs.Name),
		L2:          L2EndpointsConfigFromCLI(ctx),

		SynchronousProcessors: false,
		Datadir:               ctx.Path(flags.DataDirFlag.Name),
		Cancel:                nil,

		// Loaded below
		P2P: nil,
	}

	// TODO: make scoring params adapt per topic, so 2 second and 1 second chains can have adjusted scoring each
	p2pConfig, err := p2pcli.NewConfig(ctx, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to load p2p config: %w", err)
	}
	out.P2P = p2pConfig

	depSetSource, cfgSetSource, err := ChainConfigsFromCLI(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load config set source: %w", err)
	}
	out.DependencySetSource = depSetSource
	out.RollupConfigSetSource = cfgSetSource

	return out, nil
}

func BeaconEndpointConfigFromCLI(ctx *cli.Context) config.L1BeaconEndpointSetup {
	return &config.L1BeaconEndpointConfig{
		BeaconAddr:             ctx.String(flags.BeaconAddr.Name),
		BeaconHeader:           ctx.String(flags.BeaconHeader.Name),
		BeaconFallbackAddrs:    ctx.StringSlice(flags.BeaconFallbackAddrs.Name),
		BeaconCheckIgnore:      ctx.Bool(flags.BeaconCheckIgnore.Name),
		BeaconFetchAllSidecars: ctx.Bool(flags.BeaconFetchAllSidecars.Name),
	}
}

func DefaultBeaconEndpointConfig() *config.L1BeaconEndpointConfig {
	return &config.L1BeaconEndpointConfig{
		BeaconAddr:             flags.BeaconAddr.Value,
		BeaconHeader:           flags.BeaconHeader.Value,
		BeaconFallbackAddrs:    []string{},
		BeaconCheckIgnore:      flags.BeaconCheckIgnore.Value,
		BeaconFetchAllSidecars: flags.BeaconFetchAllSidecars.Value,
	}
}

func L1EndpointConfigFromCLI(ctx *cli.Context) *config.L1EndpointConfig {
	return &config.L1EndpointConfig{
		L1NodeAddr:       ctx.String(flags.L1NodeAddr.Name),
		L1TrustRPC:       ctx.Bool(flags.L1TrustRPC.Name),
		L1RPCKind:        sources.RPCProviderKind(strings.ToLower(ctx.String(flags.L1RPCProviderKind.Name))),
		RateLimit:        ctx.Float64(flags.L1RPCRateLimit.Name),
		BatchSize:        ctx.Int(flags.L1RPCMaxBatchSize.Name),
		HttpPollInterval: ctx.Duration(flags.L1HTTPPollInterval.Name),
		MaxConcurrency:   ctx.Int(flags.L1RPCMaxConcurrency.Name),
		CacheSize:        ctx.Uint(flags.L1CacheSize.Name),
	}
}

func DefaultL1EndpointConfig() *config.L1EndpointConfig {
	return &config.L1EndpointConfig{
		L1NodeAddr:       flags.L1NodeAddr.Value,
		L1TrustRPC:       flags.L1TrustRPC.Value,
		L1RPCKind:        *flags.L1RPCProviderKind.Value.(*sources.RPCProviderKind),
		RateLimit:        flags.L1RPCRateLimit.Value,
		BatchSize:        flags.L1RPCMaxBatchSize.Value,
		MaxConcurrency:   flags.L1RPCMaxConcurrency.Value,
		HttpPollInterval: flags.L1HTTPPollInterval.Value,
		CacheSize:        flags.L1CacheSize.Value,
	}
}

// DefaultConfig creates a new config using default values whenever possible.
// Required options with no suitable default are passed as parameters.
func DefaultConfig() *Config {
	depSet, err := depset.NewStaticConfigDependencySet(make(map[eth.ChainID]*depset.StaticConfigDependency))
	if err != nil {
		panic(err)
	}
	rollupConfigSet := depset.NewStaticRollupConfigSetV2(map[eth.ChainID]*rollup.Config{})
	return &Config{
		Version:               "dev",
		LogConfig:             oplog.DefaultCLIConfig(),
		MetricsConfig:         opmetrics.DefaultCLIConfig(),
		PprofConfig:           oppprof.DefaultCLIConfig(),
		RPC:                   oprpc.DefaultCLIConfig(),
		L1:                    DefaultL1EndpointConfig(),
		Beacon:                DefaultBeaconEndpointConfig(),
		P2P:                   nil, // disabled
		DependencySetSource:   depSet,
		RollupConfigSetSource: rollupConfigSet,
		SynchronousProcessors: false,
		Datadir:               flags.DataDirFlag.Value,
		Cancel:                func(cause error) {},
	}
}
