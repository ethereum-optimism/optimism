package supervisor

import (
	"errors"

	"github.com/urfave/cli/v2"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-supervisor/flags"
)

type CLIConfig struct {
	Version string

	LogConfig     oplog.CLIConfig
	MetricsConfig opmetrics.CLIConfig
	PprofConfig   oppprof.CLIConfig
	RPC           oprpc.CLIConfig

	// MockRun runs the service with a mock backend
	MockRun bool

	L2RPCs []string
}

func CLIConfigFromCLI(ctx *cli.Context, version string) *CLIConfig {
	return &CLIConfig{
		Version:       version,
		LogConfig:     oplog.ReadCLIConfig(ctx),
		MetricsConfig: opmetrics.ReadCLIConfig(ctx),
		PprofConfig:   oppprof.ReadCLIConfig(ctx),
		RPC:           oprpc.ReadCLIConfig(ctx),
		MockRun:       ctx.Bool(flags.MockRunFlag.Name),
		L2RPCs:        ctx.StringSlice(flags.L2RPCsFlag.Name),
	}
}

func (c *CLIConfig) Check() error {
	var result error
	result = errors.Join(result, c.MetricsConfig.Check())
	result = errors.Join(result, c.PprofConfig.Check())
	result = errors.Join(result, c.RPC.Check())
	return result
}

func DefaultCLIConfig() *CLIConfig {
	return &CLIConfig{
		Version:       "",
		LogConfig:     oplog.DefaultCLIConfig(),
		MetricsConfig: opmetrics.DefaultCLIConfig(),
		PprofConfig:   oppprof.DefaultCLIConfig(),
		RPC:           oprpc.DefaultCLIConfig(),
		MockRun:       false,
		L2RPCs:        flags.L2RPCsFlag.Value.Value(),
	}
}
