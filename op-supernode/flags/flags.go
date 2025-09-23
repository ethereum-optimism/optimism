package flags

import (
	"fmt"

	"github.com/urfave/cli/v2"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
)

const EnvVarPrefix = "OP_SUPERNODE"

func prefixEnvVars(name string) []string {
	return opservice.PrefixEnvVar(EnvVarPrefix, name)
}

var (
	SampleFlag = &cli.StringFlag{
		Name:     "sample",
		Usage:    "A sample string configuration for op-supernode",
		EnvVars:  prefixEnvVars("SAMPLE"),
		Required: true,
	}
	ChainsFlag = &cli.Uint64SliceFlag{
		Name:    "chains",
		Usage:   "List of chain IDs to run (repeatable or comma-separated)",
		EnvVars: prefixEnvVars("CHAINS"),
		Value:   cli.NewUint64Slice(),
	}
	DataDirFlag = &cli.StringFlag{
		Name:     "data-dir",
		Usage:    "Data directory for op-supernode",
		EnvVars:  prefixEnvVars("DATA_DIR"),
		Value:    "./datadir",
		Required: false,
	}
)

var requiredFlags = []cli.Flag{
	SampleFlag,
	ChainsFlag,
}

var optionalFlags []cli.Flag

func init() {
	optionalFlags = append(optionalFlags, oprpc.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, oplog.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, opmetrics.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, oppprof.CLIFlags(EnvVarPrefix)...)

	Flags = append(requiredFlags, optionalFlags...)
}

var Flags []cli.Flag

func CheckRequired(ctx *cli.Context) error {
	for _, f := range requiredFlags {
		if !ctx.IsSet(f.Names()[0]) {
			return fmt.Errorf("flag %s is required", f.Names()[0])
		}
	}
	return nil
}
