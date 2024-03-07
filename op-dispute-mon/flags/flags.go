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
	FactoryAddressFlag = &cli.StringFlag{
		Name:    "game-factory-address",
		Usage:   "Address of the fault game factory contract.",
		EnvVars: prefixEnvVars("GAME_FACTORY_ADDRESS"),
	}
	// Optional Flags
	RollupRpcFlag = &cli.StringFlag{
		Name:    "rollup-rpc",
		Usage:   "HTTP provider URL for the rollup node",
		EnvVars: prefixEnvVars("ROLLUP_RPC"),
	}
	MonitorIntervalFlag = &cli.DurationFlag{
		Name:    "monitor-interval",
		Usage:   "The interval at which the dispute monitor will check for new games to monitor.",
		EnvVars: prefixEnvVars("MONITOR_INTERVAL"),
		Value:   config.DefaultMonitorInterval,
	}
	GameWindowFlag = &cli.DurationFlag{
		Name: "game-window",
		Usage: "The time window which the monitor will consider games to report on. " +
			"This should include a bond claim buffer for games outside the maximum game duration.",
		EnvVars: prefixEnvVars("GAME_WINDOW"),
		Value:   config.DefaultGameWindow,
	}
)

// requiredFlags are checked by [CheckRequired]
var requiredFlags = []cli.Flag{
	L1EthRpcFlag,
	FactoryAddressFlag,
}

// optionalFlags is a list of unchecked cli flags
var optionalFlags = []cli.Flag{
	RollupRpcFlag,
	MonitorIntervalFlag,
	GameWindowFlag,
}

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
	gameFactoryAddress, err := opservice.ParseAddress(ctx.String(FactoryAddressFlag.Name))
	if err != nil {
		return nil, err
	}

	metricsConfig := opmetrics.ReadCLIConfig(ctx)
	pprofConfig := oppprof.ReadCLIConfig(ctx)

	return &config.Config{
		L1EthRpc:           ctx.String(L1EthRpcFlag.Name),
		GameFactoryAddress: gameFactoryAddress,

		RollupRpc:       ctx.String(RollupRpcFlag.Name),
		MonitorInterval: ctx.Duration(MonitorIntervalFlag.Name),
		GameWindow:      ctx.Duration(GameWindowFlag.Name),

		MetricsConfig: metricsConfig,
		PprofConfig:   pprofConfig,
	}, nil
}
