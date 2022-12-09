package app

import (
	"github.com/urfave/cli"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
)

const (
	ClientEndpointFlagName = "endpoint"
)

func CLIFlags(envPrefix string) []cli.Flag {
	flags := []cli.Flag{}
	flags = append(flags, oprpc.CLIFlags(envPrefix)...)
	flags = append(flags, oplog.CLIFlags(envPrefix)...)
	flags = append(flags, opmetrics.CLIFlags(envPrefix)...)
	flags = append(flags, oppprof.CLIFlags(envPrefix)...)
	return flags
}

func ClientSignCLIFlags(envPrefix string) []cli.Flag {
	flags := []cli.Flag{
		cli.StringFlag{
			Name:   ClientEndpointFlagName,
			Usage:  "Signer endpoint the client will connect to",
			Value:  "http://localhost:8080",
			EnvVar: opservice.PrefixEnvVar(envPrefix, "CLIENT_ENDPOINT"),
		},
	}
	return flags
}

type Config struct {
	ClientEndpoint string

	RPCConfig     oprpc.CLIConfig
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
	cfg := Config{
		ClientEndpoint: ctx.String(ClientEndpointFlagName),
		RPCConfig:      oprpc.ReadCLIConfig(ctx),
		LogConfig:      oplog.ReadCLIConfig(ctx),
		MetricsConfig:  opmetrics.ReadCLIConfig(ctx),
		PprofConfig:    oppprof.ReadCLIConfig(ctx),
	}
	return cfg
}
