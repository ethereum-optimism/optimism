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
	Sample        string
	Chains        []uint64
	DataDir       string
	VNFlags       flags.VNFlagMap
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
	if c.Sample == "" {
		return errors.New("sample is required")
	}
	return nil
}

func NewConfig(ctx *cli.Context, vnFlags flags.VNFlagMap) *CLIConfig {
	return &CLIConfig{
		Sample:        ctx.String(flags.SampleFlag.Name),
		Chains:        ctx.Uint64Slice(flags.ChainsFlag.Name),
		VNFlags:       vnFlags,
		RPCConfig:     oprpc.ReadCLIConfig(ctx),
		LogConfig:     oplog.ReadCLIConfig(ctx),
		MetricsConfig: opmetrics.ReadCLIConfig(ctx),
		PprofConfig:   oppprof.ReadCLIConfig(ctx),
		RawCtx:        ctx,
		DataDir:       ctx.String(flags.DataDirFlag.Name),
	}
}
