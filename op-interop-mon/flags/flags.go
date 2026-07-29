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

	DependencySetFlag = &cli.StringFlag{
		Name:      "dependency-set",
		Usage:     "Path to the interop dependency-set JSON file (sources the chain set and message expiry window)",
		EnvVars:   prefixEnvVars("DEPENDENCY_SET"),
		Required:  true,
		TakesFile: true,
	}

	// Optional Flags
	InteropFilterEndpointFlag = &cli.StringFlag{
		Name:     "interop-filter-endpoint",
		Usage:    "Optional op-interop-filter RPC endpoint to cross-check executing-message validity and observe failsafe state (read-only)",
		EnvVars:  prefixEnvVars("INTEROP_FILTER_ENDPOINT"),
		Required: false,
	}

	InteropFilterMinSafetyFlag = &cli.StringFlag{
		Name:     "interop-filter-min-safety",
		Usage:    "Minimum safety level requested from the interop-filter checkAccessList cross-check; the filter only supports cross-unsafe or unsafe",
		EnvVars:  prefixEnvVars("INTEROP_FILTER_MIN_SAFETY"),
		Required: false,
		Value:    "cross-unsafe",
	}

	SupernodeEndpointsFlag = &cli.StringSliceFlag{
		Name:     "supernode-endpoints",
		Usage:    "Optional op-supernode CL RPC endpoints to observe liveness, per-chain safe/finalized heads, and cross-safety violations (read-only)",
		EnvVars:  prefixEnvVars("SUPERNODE_ENDPOINTS"),
		Required: false,
	}
)

var requiredFlags = []cli.Flag{
	L2RpcsFlag,
	DependencySetFlag,
}

var optionalFlags = []cli.Flag{
	InteropFilterEndpointFlag,
	InteropFilterMinSafetyFlag,
	SupernodeEndpointsFlag,
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
