package server

import (
	"github.com/urfave/cli/v2"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-test/server/flags"
)

type CLIConfig struct {
	Version string

	Log oplog.CLIConfig
	RPC oprpc.CLIConfig

	Config string
}

func ReadCLIConfig(ctx *cli.Context, version string) *CLIConfig {
	return &CLIConfig{
		Version: version,
		Log:     oplog.ReadCLIConfig(ctx),
		RPC:     oprpc.ReadCLIConfig(ctx),
		Config:  ctx.String(flags.ConfigFlag.Name),
	}
}
