package plasma

import (
	"fmt"
	"net/url"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/urfave/cli/v2"
)

const DaServerAddressFlagName = "da-server"

func CLIFlags(envPrefix string) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    DaServerAddressFlagName,
			Usage:   "HTTP address of a DaServer",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "DA_SERVER"),
		},
	}
}

type Config struct {
	Enabled     bool
	DAServerURL string
}

func (c Config) Check() error {
	if c.Enabled {
		if c.DAServerURL == "" {
			return fmt.Errorf("DA server URL is required when plasma da is enabled")
		}
		if _, err := url.Parse(c.DAServerURL); err != nil {
			return fmt.Errorf("DA server URL is invalid: %w", err)
		}
	}
	return nil
}

type CLIConfig struct {
	DAServer string
}

func (c *CLIConfig) Config(enabled bool) Config {
	return Config{
		Enabled:     enabled,
		DAServerURL: c.DAServer,
	}
}

func ReadCLIConfig(c *cli.Context) CLIConfig {
	return CLIConfig{
		DAServer: c.String(DaServerAddressFlagName),
	}
}
