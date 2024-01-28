package main

import (
	"errors"

	"github.com/urfave/cli/v2"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

type RunConfig struct {
	// ClientCommand is the command to create the client sub-process with.
	// If left empty, the service assumes the parent-process is the client.
	ClientCommand string

	// HostCommand is the command to create the host sub-process with.
	// If left empty, the service assumes the parent-process is the host.
	HostCommand string

	LogHints          bool
	LogPreimageKeys   bool
	LogPreimageValues bool
}

type Config struct {
	LogCfg oplog.CLIConfig

	RunConfig
}

func (c *Config) Check() error {
	if c.HostCommand == "" && c.ClientCommand == "" {
		return errors.New("parent-process cannot be both host and client")
	}
	return nil
}

func ReadCLIConfig(ctx *cli.Context) *Config {
	return &Config{
		LogCfg: oplog.ReadCLIConfig(ctx),
		RunConfig: RunConfig{
			ClientCommand:     ctx.String(ClientCommandFlag.Name),
			HostCommand:       ctx.String(HostCommandFlag.Name),
			LogHints:          ctx.Bool(InfoHintsFlag.Name),
			LogPreimageKeys:   ctx.Bool(InfoPreimageKeysFlag.Name),
			LogPreimageValues: ctx.Bool(InfoPreimageValuesFlag.Name),
		},
	}
}
