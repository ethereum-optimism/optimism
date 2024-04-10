package plasma

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/urfave/cli/v2"
)

const (
	EnabledFlagName       = "plasma.enabled"
	DaServerAddressFlagName = "plasma.da-server"
	VerifyOnReadFlagName  = "plasma.verify-on-read"
)

func plasmaEnv(envPrefix, flagName string) []string {
	return []string{fmt.Sprintf("%s_PLASMA_%s", envPrefix, strings.ToUpper(flagName))}
}

func NewCLIFlags(envPrefix string, category string) []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:     EnabledFlagName,
			Usage:    "Enable plasma mode",
			Value:    false,
			EnvVars:  plasmaEnv(envPrefix, "enabled"),
			Category: category,
		},
		&cli.StringFlag{
			Name:     DaServerAddressFlagName,
			Usage:    "HTTP address of a DA Server",
			EnvVars:  plasmaEnv(envPrefix, "da-server"),
			Category: category,
		},
		&cli.BoolFlag{
			Name:     VerifyOnReadFlagName,
			Usage:    "Verify input data matches the commitments from the DA storage service",
			Value:    true,
			EnvVars:  plasmaEnv(envPrefix, "verify-on-read"),
			Category: category,
		},
	}
}

type CLIConfig struct {
	Enabled     bool
	DAServerURL string
	VerifyOnRead bool
}

func (c *CLIConfig) Check() error {
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

func (c *CLIConfig) NewDAClient() *DAClient {
	return &DAClient{url: c.DAServerURL, verify: c.VerifyOnRead}
}

func ReadCLIConfig(c *cli.Context) *CLIConfig {
	return &CLIConfig{
		Enabled:     c.Bool(EnabledFlagName),
		DAServerURL: c.String(DaServerAddressFlagName),
		VerifyOnRead: c.Bool(VerifyOnReadFlagName),
	}
}
