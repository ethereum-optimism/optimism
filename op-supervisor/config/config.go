package config

import (
	"errors"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
)

var (
	ErrMissingL2RPC   = errors.New("must specify at least one L2 RPC")
	ErrMissingDatadir = errors.New("must specify datadir")
)

type Config struct {
	Version string

	LogConfig     oplog.CLIConfig
	MetricsConfig opmetrics.CLIConfig
	PprofConfig   oppprof.CLIConfig
	RPC           oprpc.CLIConfig

	// MockRun runs the service with a mock backend
	MockRun bool

	L2RPCs  []string
	Datadir string
}

func (c *Config) Check() error {
	var result error
	result = errors.Join(result, c.MetricsConfig.Check())
	result = errors.Join(result, c.PprofConfig.Check())
	result = errors.Join(result, c.RPC.Check())
	if len(c.L2RPCs) == 0 {
		result = errors.Join(result, ErrMissingL2RPC)
	}
	if c.Datadir == "" {
		result = errors.Join(result, ErrMissingDatadir)
	}
	return result
}

// NewConfig creates a new config using default values whenever possible.
// Required options with no suitable default are passed as parameters.
func NewConfig(l2RPCs []string, datadir string) *Config {
	return &Config{
		LogConfig:     oplog.DefaultCLIConfig(),
		MetricsConfig: opmetrics.DefaultCLIConfig(),
		PprofConfig:   oppprof.DefaultCLIConfig(),
		RPC:           oprpc.DefaultCLIConfig(),
		MockRun:       false,
		L2RPCs:        l2RPCs,
		Datadir:       datadir,
	}
}
