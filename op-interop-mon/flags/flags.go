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
	SupervisorEndpointsFlag = &cli.StringSliceFlag{
		Name:     "supervisor-endpoints",
		Usage:    "The RPC endpoints for the supervisors to call admin_setFailsafeEnabled",
		EnvVars:  prefixEnvVars("SUPERVISOR_ENDPOINTS"),
		Required: false,
	}

	TriggerFailsafeFlag = &cli.BoolFlag{
		Name:     "trigger-failsafe",
		Usage:    "Enable automatic failsafe triggering when invalid messages are detected",
		EnvVars:  prefixEnvVars("TRIGGER_FAILSAFE"),
		Required: false,
		Value:    true,
	}
)

var requiredFlags = []cli.Flag{
	L2RpcsFlag,
}

var optionalFlags = []cli.Flag{
	SupervisorEndpointsFlag,
	TriggerFailsafeFlag,
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
