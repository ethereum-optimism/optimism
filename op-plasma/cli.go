package plasma

import (
	"fmt"
	"net/url"

	"github.com/urfave/cli/v2"
)

const (
	EnabledFlagName         = "plasma.enabled"
	DaServerAddressFlagName = "plasma.da-server"
	VerifyOnReadFlagName    = "plasma.verify-on-read"
)

func plasmaEnv(envprefix, v string) []string {
	return []string{envprefix + "_PLASMA_" + v}
}

func CLIFlags(envPrefix string, category string) []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:     EnabledFlagName,
			Usage:    "Enable plasma mode",
			Value:    false,
			EnvVars:  plasmaEnv(envPrefix, "ENABLED"),
			Category: category,
		},
		&cli.StringFlag{
			Name:     DaServerAddressFlagName,
			Usage:    "HTTP address of a DA Server",
			EnvVars:  plasmaEnv(envPrefix, "DA_SERVER"),
			Category: category,
		},
		&cli.BoolFlag{
			Name:     VerifyOnReadFlagName,
			Usage:    "Verify input data matches the commitments from the DA storage service",
			Value:    true,
			EnvVars:  plasmaEnv(envPrefix, "VERIFY_ON_READ"),
			Category: category,
		},
	}
}

type CLIConfig struct {
	Enabled      bool
	DAServerURL  string
	VerifyOnRead bool
}

func (c CLIConfig) Check() error {
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

func (c CLIConfig) NewDAClient() *DAClient {
	return &DAClient{url: c.DAServerURL, verify: c.VerifyOnRead}
}

func ReadCLIConfig(c *cli.Context) CLIConfig {
	return CLIConfig{
		Enabled:      c.Bool(EnabledFlagName),
		DAServerURL:  c.String(DaServerAddressFlagName),
		VerifyOnRead: c.Bool(VerifyOnReadFlagName),
	}
}
