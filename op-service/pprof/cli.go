package pprof

import (
	"errors"
	"math"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/urfave/cli/v2"
)

const (
	EnabledFlagName    = "pprof.enabled"
	ListenAddrFlagName = "pprof.addr"
	PortFlagName       = "pprof.port"
)

func CLIFlags(envPrefix string) []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:    EnabledFlagName,
			Usage:   "Enable the pprof server",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "PPROF_ENABLED"),
		},
		&cli.StringFlag{
			Name:    ListenAddrFlagName,
			Usage:   "pprof listening address",
			Value:   "0.0.0.0",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "PPROF_ADDR"),
		},
		&cli.IntFlag{
			Name:    PortFlagName,
			Usage:   "pprof listening port",
			Value:   6060,
			EnvVars: opservice.PrefixEnvVar(envPrefix, "PPROF_PORT"),
		},
	}
}

type CLIConfig struct {
	Enabled    bool
	ListenAddr string
	ListenPort int
}

func (m CLIConfig) Check() error {
	if !m.Enabled {
		return nil
	}

	if m.ListenPort < 0 || m.ListenPort > math.MaxUint16 {
		return errors.New("invalid pprof port")
	}

	return nil
}

func ReadCLIConfig(ctx *cli.Context) CLIConfig {
	return CLIConfig{
		Enabled:    ctx.Bool(EnabledFlagName),
		ListenAddr: ctx.String(ListenAddrFlagName),
		ListenPort: ctx.Int(PortFlagName),
	}
}
