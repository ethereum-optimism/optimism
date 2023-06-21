package rpc

import (
	"github.com/urfave/cli/v2"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
)

const (
	EnableAdminFlagName = "rpc.enable-admin"
)

func CLIFlags(envPrefix string) []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:    EnableAdminFlagName,
			Usage:   "Enable the admin API (experimental)",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "RPC_ENABLE_ADMIN"),
		},
	}
}

type CLIConfig struct {
	oprpc.CLIConfig
	EnableAdmin bool
}

func ReadCLIConfig(ctx *cli.Context) CLIConfig {
	return CLIConfig{
		CLIConfig:   oprpc.ReadCLIConfig(ctx),
		EnableAdmin: ctx.Bool(EnableAdminFlagName),
	}
}
