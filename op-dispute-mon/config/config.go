package config

import (
	"errors"
	"fmt"

	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
)

var (
	ErrMissingL1EthRPC = errors.New("missing l1 eth rpc url")
)

// Config is a well typed config that is parsed from the CLI params.
// It also contains config options for auxiliary services.
type Config struct {
	L1EthRpc string // L1 RPC Url

	MetricsConfig opmetrics.CLIConfig
	PprofConfig   oppprof.CLIConfig
}

func NewConfig(l1EthRpc string) Config {
	return Config{
		L1EthRpc:      l1EthRpc,
		MetricsConfig: opmetrics.DefaultCLIConfig(),
		PprofConfig:   oppprof.DefaultCLIConfig(),
	}
}

func (c Config) Check() error {
	if c.L1EthRpc == "" {
		return ErrMissingL1EthRPC
	}
	if err := c.MetricsConfig.Check(); err != nil {
		return fmt.Errorf("metrics config: %w", err)
	}
	if err := c.PprofConfig.Check(); err != nil {
		return fmt.Errorf("pprof config: %w", err)
	}
	return nil
}
