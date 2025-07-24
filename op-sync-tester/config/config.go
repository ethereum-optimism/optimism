package config

import (
	"errors"

	stconf "github.com/HashKeyChain/verse/op-sync-tester/synctester/backend/config"

	oplog "github.com/HashKeyChain/verse/op-service/log"
	opmetrics "github.com/HashKeyChain/verse/op-service/metrics"
	"github.com/HashKeyChain/verse/op-service/oppprof"
	oprpc "github.com/HashKeyChain/verse/op-service/rpc"
)

const (
	DefaultConfigYaml = "config.yaml"
)

type Config struct {
	Version string

	LogConfig     oplog.CLIConfig
	MetricsConfig opmetrics.CLIConfig
	PprofConfig   oppprof.CLIConfig
	RPC           oprpc.CLIConfig

	SyncTesters stconf.Loader
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
		LogConfig:     oplog.DefaultCLIConfig(),
		MetricsConfig: opmetrics.DefaultCLIConfig(),
		PprofConfig:   oppprof.DefaultCLIConfig(),
		RPC:           oprpc.DefaultCLIConfig(),
		SyncTesters:   &stconf.YamlLoader{Path: DefaultConfigYaml},
	}
}
