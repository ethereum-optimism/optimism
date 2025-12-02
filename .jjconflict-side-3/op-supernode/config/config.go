package config

import (
	"errors"

	"github.com/urfave/cli/v2"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-supernode/flags"
)

type CLIConfig struct {
	Chains        []uint64
	DataDir       string
	L1NodeAddr    string
	L1BeaconAddr  string
	RPCConfig     oprpc.CLIConfig
	LogConfig     oplog.CLIConfig
	MetricsConfig opmetrics.CLIConfig
	PprofConfig   oppprof.CLIConfig
	RawCtx        *cli.Context
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
	return nil
}

func NewConfig(ctx *cli.Context) *CLIConfig {
	return &CLIConfig{
		Chains:        ctx.Uint64Slice(flags.ChainsFlag.Name),
		DataDir:       ctx.String(flags.DataDirFlag.Name),
		L1NodeAddr:    ctx.String(flags.L1NodeAddr.Name),
		L1BeaconAddr:  ctx.String(flags.L1BeaconAddr.Name),
		RPCConfig:     oprpc.ReadCLIConfig(ctx),
		LogConfig:     oplog.ReadCLIConfig(ctx),
		MetricsConfig: opmetrics.ReadCLIConfig(ctx),
		PprofConfig:   oppprof.ReadCLIConfig(ctx),
		RawCtx:        ctx,
	}
}
