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

const EnvVarPrefix = "OP_INTEROP_MON"

func prefixEnvVars(name string) []string {
	return opservice.PrefixEnvVar(EnvVarPrefix, name)
}

var (
	// Required Flags
	L2RpcsFlag = &cli.StringSliceFlag{
		Name:     "l2-rpcs",
		Usage:    "The RPC URLs for the L2 chains to monitor",
		EnvVars:  prefixEnvVars("L2_RPCS"),
		Required: true,
	}

	// Optional Flags
	SupervisorEndpointFlag = &cli.StringFlag{
		Name:     "supervisor-endpoint",
		Usage:    "The RPC endpoint for the supervisor to call admin_setFailsafeEnabled",
		EnvVars:  prefixEnvVars("SUPERVISOR_ENDPOINT"),
		Required: false,
	}
)

var requiredFlags = []cli.Flag{
	L2RpcsFlag,
}

var optionalFlags = []cli.Flag{
	SupervisorEndpointFlag,
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
		if !ctx.IsSet(f.Names()[0]) {
			return fmt.Errorf("flag %s is required", f.Names()[0])
		}
	}
	return nil
}
