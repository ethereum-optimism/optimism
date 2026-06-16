package monitor

import (
	"errors"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-interop-mon/flags"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
)

type CLIConfig struct {
	L2Rpcs            []string
	DependencySetPath string
	PollInterval      time.Duration

	// InteropFilterEndpoint, when set, enables a read-only cross-check against the op-interop-filter.
	InteropFilterEndpoint  string
	InteropFilterMinSafety string

	// SupernodeEndpoints, when set, enable read-only observation of op-supernode liveness,
	// per-chain heads, and cross-safety violations.
	SupernodeEndpoints []string

	RPCConfig     oprpc.CLIConfig
	LogConfig     oplog.CLIConfig
	MetricsConfig opmetrics.CLIConfig
	PprofConfig   oppprof.CLIConfig
}

func (c *CLIConfig) Check() error {
	if err := c.RPCConfig.Check(); err != nil {
		return err
	}
	if err := c.MetricsConfig.Check(); err != nil {
		return err
	}
	if err := c.PprofConfig.Check(); err != nil {
		return err
	}

	if len(c.L2Rpcs) == 0 {
		return errors.New("l2 rpcs are required")
	}

	if c.DependencySetPath == "" {
		return errors.New("dependency-set is required")
	}

	return nil
}

func NewConfig(ctx *cli.Context) *CLIConfig {
	return &CLIConfig{
		// Required Flags
		L2Rpcs:            ctx.StringSlice(flags.L2RpcsFlag.Name),
		DependencySetPath: ctx.String(flags.DependencySetFlag.Name),

		// Optional Flags
		InteropFilterEndpoint:  ctx.String(flags.InteropFilterEndpointFlag.Name),
		InteropFilterMinSafety: ctx.String(flags.InteropFilterMinSafetyFlag.Name),
		SupernodeEndpoints:     ctx.StringSlice(flags.SupernodeEndpointsFlag.Name),
		RPCConfig:              oprpc.ReadCLIConfig(ctx),
		LogConfig:              oplog.ReadCLIConfig(ctx),
		MetricsConfig:          opmetrics.ReadCLIConfig(ctx),
		PprofConfig:            oppprof.ReadCLIConfig(ctx),
	}
}
