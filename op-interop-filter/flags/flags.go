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

const EnvVarPrefix = "OP_INTEROP_FILTER"

func prefixEnvVars(name string) []string {
	return opservice.PrefixEnvVar(EnvVarPrefix, name)
}

var (
	L2RPCsFlag = &cli.StringSliceFlag{
		Name:    "l2-rpcs",
		Usage:   "L2 RPC endpoints to connect to (chain ID is queried from each endpoint)",
		EnvVars: prefixEnvVars("L2_RPCS"),
	}
	DataDirFlag = &cli.StringFlag{
		Name:    "data-dir",
		Usage:   "Directory for LogsDB storage. If empty, uses in-memory storage",
		EnvVars: prefixEnvVars("DATA_DIR"),
		Value:   "",
	}
	BackfillDurationFlag = &cli.StringFlag{
		Name:    "backfill-duration",
		Usage:   "Duration to backfill on startup (e.g., 24h, 30m, 1h30m)",
		EnvVars: prefixEnvVars("BACKFILL_DURATION"),
		Value:   "24h",
	}
	JWTSecretFlag = &cli.StringFlag{
		Name: "rpc.jwt-secret",
		Usage: "Path to JWT secret key for RPC authentication. " +
			"Keys are 32 bytes, hex encoded in a file. " +
			"A new key will be generated if the file is empty.",
		EnvVars:   prefixEnvVars("RPC_JWT_SECRET"),
		Value:     "",
		TakesFile: true,
	}
)

var requiredFlags = []cli.Flag{
	L2RPCsFlag,
}

var optionalFlags = []cli.Flag{
	DataDirFlag,
	BackfillDurationFlag,
	JWTSecretFlag,
}

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
		name := f.Names()[0]
		if !ctx.IsSet(name) {
			return fmt.Errorf("flag %s is required", name)
		}
	}
	return nil
}
