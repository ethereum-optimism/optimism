package flags

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/HashKeyChain/verse/op-faucet/config"
	fconf "github.com/HashKeyChain/verse/op-faucet/faucet/backend/config"
	opservice "github.com/HashKeyChain/verse/op-service"
	oplog "github.com/HashKeyChain/verse/op-service/log"
	opmetrics "github.com/HashKeyChain/verse/op-service/metrics"
	"github.com/HashKeyChain/verse/op-service/oppprof"
	oprpc "github.com/HashKeyChain/verse/op-service/rpc"
)

const EnvVarPrefix = "OP_FAUCET"

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

var requiredFlags = []cli.Flag{}

var optionalFlags = []cli.Flag{
	ConfigFlag,
}

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
		Faucets:       &fconf.YamlLoader{Path: ctx.String(ConfigFlag.Name)},
	}
}
