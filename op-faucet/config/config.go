package config

import (
	"errors"

	fconf "github.com/ethereum-optimism/optimism/op-faucet/faucet/backend/config"
	"github.com/ethereum-optimism/optimism/op-service/log/logcli"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
)

const (
	DefaultConfigYaml = "config.yaml"
)

type Config struct {
	Version string

	LogConfig     logcli.CLIConfig
	MetricsConfig opmetrics.CLIConfig
	PprofConfig   oppprof.CLIConfig
	RPC           oprpc.CLIConfig

	Faucets fconf.Loader
}

func (c *Config) Check() error {
	var result error
	result = errors.Join(result, c.MetricsConfig.Check())
	result = errors.Join(result, c.PprofConfig.Check())
	result = errors.Join(result, c.RPC.Check())
	return result
}

func DefaultCLIConfig() *Config {
	return &Config{
		Version:       "dev",
		LogConfig:     logcli.DefaultCLIConfig(),
		MetricsConfig: opmetrics.DefaultCLIConfig(),
		PprofConfig:   oppprof.DefaultCLIConfig(),
		RPC:           oprpc.DefaultCLIConfig(),
		Faucets:       &fconf.YamlLoader{Path: DefaultConfigYaml},
	}
}
