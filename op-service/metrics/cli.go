package metrics

import (
	"errors"
	"math"

	opservice "github.com/ethereum-optimism/optimism/op-service"

	"github.com/urfave/cli/v2"
)

const (
	EnabledFlagName    = "metrics.enabled"
	ListenAddrFlagName = "metrics.addr"
	PortFlagName       = "metrics.port"
	defaultListenAddr  = "0.0.0.0"
	defaultListenPort  = 7300
)

func DefaultCLIConfig() CLIConfig {
	return CLIConfig{
		Enabled:    false,
		ListenAddr: defaultListenAddr,
		ListenPort: defaultListenPort,
	}
}

type FlagValue string

func (fv FlagValue) Set(value string) error {
	*(*string)(&fv) = value
	return nil
}

func (fv FlagValue) String() string {
	return string(fv)
}
func CLIFlags(envPrefix string) []cli.Flag {
	return []cli.Flag{
		&cli.GenericFlag{
			Name:    EnabledFlagName,
			Usage:   "Enable the metrics server",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "METRICS_ENABLED"),
		},
		&cli.GenericFlag{
			Name:    ListenAddrFlagName,
			Usage:   "Metrics listening address",
			Value:   FlagValue(defaultListenAddr), // TODO(CLI-4159): Switch to 127.0.0.1
			EnvVars: opservice.PrefixEnvVar(envPrefix, "METRICS_ADDR"),
		},
		&cli.GenericFlag{
			Name:    PortFlagName,
			Usage:   "Metrics listening port",
			Value:   FlagValue(defaultListenAddr),
			EnvVars: opservice.PrefixEnvVar(envPrefix, "METRICS_PORT"),
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
		return errors.New("invalid metrics port")
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
