package flags

import (
	"fmt"

	"github.com/urfave/cli/v2"

	stconf "github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/config"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-sync-tester/config"
)

const EnvVarPrefix = "OP_SYNC_TESTER"

func prefixEnvVars(name string) []string {
	return opservice.PrefixEnvVar(EnvVarPrefix, name)
}

var (
	ConfigFlag = &cli.StringFlag{
		Name:    "config",
		Usage:   "Configuration file path",
		EnvVars: prefixEnvVars("CONFIG"),
		Value:   config.DefaultConfigYaml,
	}
)

var requiredFlags = []cli.Flag{ConfigFlag}

var optionalFlags = []cli.Flag{}

func init() {
	optionalFlags = append(optionalFlags, oprpc.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, oplog.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, opmetrics.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, oppprof.CLIFlags(EnvVarPrefix)...)

	Flags = append(Flags, requiredFlags...)
	Flags = append(Flags, optionalFlags...)
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

func ConfigFromCLI(ctx *cli.Context, version string) *config.Config {
	return &config.Config{
		Version:       version,
		LogConfig:     oplog.ReadCLIConfig(ctx),
		MetricsConfig: opmetrics.ReadCLIConfig(ctx),
		PprofConfig:   oppprof.ReadCLIConfig(ctx),
		RPC:           oprpc.ReadCLIConfig(ctx),
		SyncTesters:   &stconf.YamlLoader{Path: ctx.String(ConfigFlag.Name)},
	}
}
