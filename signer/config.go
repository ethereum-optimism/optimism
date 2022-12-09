package app

import (
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/urfave/cli"
)

func CLIFlags(envPrefix string) []cli.Flag {
	flags := []cli.Flag{}
	flags = append(flags, oprpc.CLIFlags(envPrefix)...)
	flags = append(flags, oplog.CLIFlags(envPrefix)...)
	flags = append(flags, opmetrics.CLIFlags(envPrefix)...)
	flags = append(flags, oppprof.CLIFlags(envPrefix)...)
	return flags
}

type Config struct {
	RPCConfig oprpc.CLIConfig

	LogConfig     oplog.CLIConfig
	MetricsConfig opmetrics.CLIConfig
	PprofConfig   oppprof.CLIConfig
}

func (c Config) Check() error {
	if err := c.RPCConfig.Check(); err != nil {
		return err
	}
	if err := c.LogConfig.Check(); err != nil {
		return err
	}
	if err := c.MetricsConfig.Check(); err != nil {
		return err
	}
	if err := c.PprofConfig.Check(); err != nil {
		return err
	}
	return nil
}

func NewConfig(ctx *cli.Context) Config {
	return Config{
		RPCConfig:     oprpc.ReadCLIConfig(ctx),
		LogConfig:     oplog.ReadCLIConfig(ctx),
		MetricsConfig: opmetrics.ReadCLIConfig(ctx),
		PprofConfig:   oppprof.ReadCLIConfig(ctx),
	}
}
