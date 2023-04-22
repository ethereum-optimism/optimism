package flags

import (
	"fmt"

	"github.com/urfave/cli"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

const envVarPrefix = "OP_PROPOSER"

var (
	// Required Flags
	L1EthRpcFlag = cli.StringFlag{
		Name:   "l1-eth-rpc",
		Usage:  "HTTP provider URL for L1",
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "L1_ETH_RPC"),
	}
	RollupRpcFlag = cli.StringFlag{
		Name:   "rollup-rpc",
		Usage:  "HTTP provider URL for the rollup node",
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "ROLLUP_RPC"),
	}
	L2OOAddressFlag = cli.StringFlag{
		Name:   "l2oo-address",
		Usage:  "Address of the L2OutputOracle contract",
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "L2OO_ADDRESS"),
	}
	PollIntervalFlag = cli.DurationFlag{
		Name: "poll-interval",
		Usage: "Delay between querying L2 for more transactions and " +
			"creating a new batch",
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "POLL_INTERVAL"),
	}
	// Optional flags
	AllowNonFinalizedFlag = cli.BoolFlag{
		Name:   "allow-non-finalized",
		Usage:  "Allow the proposer to submit proposals for L2 blocks derived from non-finalized L1 blocks.",
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "ALLOW_NON_FINALIZED"),
	}
	// Legacy Flags
	L2OutputHDPathFlag = txmgr.L2OutputHDPathFlag
)

var requiredFlags = []cli.Flag{
	L1EthRpcFlag,
	RollupRpcFlag,
	L2OOAddressFlag,
	PollIntervalFlag,
}

var optionalFlags = []cli.Flag{
	AllowNonFinalizedFlag,
}

func init() {
	requiredFlags = append(requiredFlags, oprpc.CLIFlags(envVarPrefix)...)

	optionalFlags = append(optionalFlags, oplog.CLIFlags(envVarPrefix)...)
	optionalFlags = append(optionalFlags, opmetrics.CLIFlags(envVarPrefix)...)
	optionalFlags = append(optionalFlags, oppprof.CLIFlags(envVarPrefix)...)
	optionalFlags = append(optionalFlags, txmgr.CLIFlags(envVarPrefix)...)

	Flags = append(requiredFlags, optionalFlags...)
}

// Flags contains the list of configuration options available to the binary.
var Flags []cli.Flag

func CheckRequired(ctx *cli.Context) error {
	for _, f := range requiredFlags {
		if !ctx.GlobalIsSet(f.GetName()) {
			return fmt.Errorf("flag %s is required", f.GetName())
		}
	}
	return nil
}
