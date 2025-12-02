package inspect

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

// Config holds the configuration for the inspect service
type Config struct {
	EnclaveID           string
	FixTraefik          bool
	ConductorConfigPath string
	EnvironmentPath     string
}

func NewConfig(ctx *cli.Context) (*Config, error) {
	if ctx.NArg() != 1 {
		return nil, fmt.Errorf("expected exactly one argument (enclave-id), got %d", ctx.NArg())
	}

	cfg := &Config{
		EnclaveID:           ctx.Args().Get(0),
		FixTraefik:          ctx.Bool("fix-traefik"),
		ConductorConfigPath: ctx.String("conductor-config-path"),
		EnvironmentPath:     ctx.String("environment-path"),
	}

	if cfg.EnclaveID == "" {
		return nil, fmt.Errorf("enclave-id is required")
	}

	return cfg, nil
}
