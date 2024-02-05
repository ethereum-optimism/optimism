package flags

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-dispute-mon/config"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
)

const (
	envVarPrefix = "OP_DISPUTE_MON"
)

func prefixEnvVars(name string) []string {
	return opservice.PrefixEnvVar(envVarPrefix, name)
}

var (
	// Required Flags
	L1EthRpcFlag = &cli.StringFlag{
		Name:    "l1-eth-rpc",
		Usage:   "HTTP provider URL for L1.",
		EnvVars: prefixEnvVars("L1_ETH_RPC"),
	}
	// Optional Flags
)

// requiredFlags are checked by [CheckRequired]
var requiredFlags = []cli.Flag{
	L1EthRpcFlag,
}

// optionalFlags is a list of unchecked cli flags
var optionalFlags = []cli.Flag{}

func init() {
	optionalFlags = append(optionalFlags, oplog.CLIFlags(envVarPrefix)...)
	optionalFlags = append(optionalFlags, opmetrics.CLIFlags(envVarPrefix)...)
	optionalFlags = append(optionalFlags, oppprof.CLIFlags(envVarPrefix)...)

	Flags = append(requiredFlags, optionalFlags...)
}

// Flags contains the list of configuration options available to the binary.
var Flags []cli.Flag

func CheckRequired(ctx *cli.Context) error {
	for _, f := range requiredFlags {
		if !ctx.IsSet(f.Names()[0]) {
			return fmt.Errorf("flag %s is required", f.Names()[0])
		}
	}
	return nil
}

// NewConfigFromCLI parses the Config from the provided flags or environment variables.
func NewConfigFromCLI(ctx *cli.Context) (*config.Config, error) {
	if err := CheckRequired(ctx); err != nil {
		return nil, err
	}

	metricsConfig := opmetrics.ReadCLIConfig(ctx)
	pprofConfig := oppprof.ReadCLIConfig(ctx)

	return &config.Config{
		L1EthRpc:      ctx.String(L1EthRpcFlag.Name),
		MetricsConfig: metricsConfig,
		PprofConfig:   pprofConfig,
	}, nil
}
