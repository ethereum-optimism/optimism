package rpc

import (
	"errors"
	"math"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/urfave/cli/v2"
)

const (
	ListenAddrFlagName  = "rpc.addr"
	PortFlagName        = "rpc.port"
	EnableAdminFlagName = "rpc.enable-admin"
)

var ErrInvalidPort = errors.New("invalid RPC port")

func CLIFlags(envPrefix string) []cli.Flag {
	return CLIFlagsWithCategory(envPrefix, "", CLIConfig{
		ListenAddr:  "0.0.0.0", // TODO(#16487): Switch to 127.0.0.1
		ListenPort:  8545,
		EnableAdmin: false,
	})
}

func CLIFlagsWithCategory(envPrefix string, category string, cfg CLIConfig) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     ListenAddrFlagName,
			Usage:    "rpc listening address",
			Value:    cfg.ListenAddr,
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "RPC_ADDR"),
			Category: category,
		},
		&cli.IntFlag{
			Name:     PortFlagName,
			Usage:    "rpc listening port",
			Value:    cfg.ListenPort,
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "RPC_PORT"),
			Category: category,
		},
		&cli.BoolFlag{
			Name:     EnableAdminFlagName,
			Usage:    "Enable the admin API",
			Value:    cfg.EnableAdmin,
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "RPC_ENABLE_ADMIN"),
			Category: category,
		},
	}
}

type CLIConfig struct {
	ListenAddr  string
	ListenPort  int
	EnableAdmin bool
}

func DefaultCLIConfig() CLIConfig {
	return CLIConfig{
		ListenAddr:  "0.0.0.0",
		ListenPort:  8545,
		EnableAdmin: false,
	}
}

func (c CLIConfig) Check() error {
	if c.ListenPort < 0 || c.ListenPort > math.MaxUint16 {
		return ErrInvalidPort
	}

	return nil
}

func ReadCLIConfig(ctx *cli.Context) CLIConfig {
	return CLIConfig{
		ListenAddr:  ctx.String(ListenAddrFlagName),
		ListenPort:  ctx.Int(PortFlagName),
		EnableAdmin: ctx.Bool(EnableAdminFlagName),
	}
}
