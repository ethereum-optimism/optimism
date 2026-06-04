package config

import (
	"errors"
	"time"

	"github.com/urfave/cli/v2"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-supernode/flags"
)

type CLIConfig struct {
	Chains                     []uint64
	DataDir                    string
	L1NodeAddr                 string
	L1HTTPPollInterval         time.Duration
	L1BeaconAddr               string
	L1BeaconFallbackAddrs      []string
	RPCConfig                  oprpc.CLIConfig
	LogConfig                  oplog.CLIConfig
	MetricsConfig              opmetrics.CLIConfig
	PprofConfig                oppprof.CLIConfig
	RawCtx                     *cli.Context
	InteropActivationTimestamp *uint64
	// InteropLogBackfillDepth is the duration (e.g. 168h) to extend initiating-message log ingestion
	// backward from the tip before interop message validation runs. Set to zero to disable.
	InteropLogBackfillDepth time.Duration
	// DependencySetPath is the path to a JSON dependency-set file shared by every chain
	// managed by the supernode. Empty means fall back to per-chain registry lookup.
	DependencySetPath string
}

func (c *CLIConfig) Check() error {
	if err := c.RPCConfig.Check(); err != nil {
		return err
	}
	if err := c.MetricsConfig.Check(); err != nil {
		return err
	}
	if err := c.PprofConfig.Check(); err != nil {
		return err
	}
	if c.L1NodeAddr == "" {
		return errors.New("l1 node address is required")
	}
	if c.L1HTTPPollInterval < 0 {
		return errors.New("l1 http poll interval must not be negative")
	}
	if c.InteropLogBackfillDepth < 0 {
		return errors.New("interop.log-backfill-depth must be >= 0")
	}
	// Note: InteropLogBackfillDepth > 0 also requires a resolved interop
	// activation timestamp, but that can be satisfied either by the CLI
	// override (InteropActivationTimestamp) or by rollup configs loaded
	// later during supernode construction. The pairing check runs after
	// resolution, in supernode.New, so that rollup-derived activation
	// counts as configured.
	return nil
}

func NewConfig(ctx *cli.Context) *CLIConfig {
	cfg := &CLIConfig{
		Chains:                  ctx.Uint64Slice(flags.ChainsFlag.Name),
		DataDir:                 ctx.String(flags.DataDirFlag.Name),
		L1NodeAddr:              ctx.String(flags.L1NodeAddr.Name),
		L1HTTPPollInterval:      ctx.Duration(flags.L1HTTPPollInterval.Name),
		L1BeaconAddr:            ctx.String(flags.L1BeaconAddr.Name),
		L1BeaconFallbackAddrs:   ctx.StringSlice(flags.L1BeaconFallbackAddrs.Name),
		RPCConfig:               oprpc.ReadCLIConfig(ctx),
		LogConfig:               oplog.ReadCLIConfig(ctx),
		MetricsConfig:           opmetrics.ReadCLIConfig(ctx),
		PprofConfig:             oppprof.ReadCLIConfig(ctx),
		RawCtx:                  ctx,
		InteropLogBackfillDepth: ctx.Duration("interop.log-backfill-depth"),
		DependencySetPath:       ctx.Path(flags.DependencySet.Name),
	}
	if ctx.IsSet("interop.activation-timestamp") {
		ts := ctx.Uint64("interop.activation-timestamp")
		cfg.InteropActivationTimestamp = &ts
	}
	return cfg
}
