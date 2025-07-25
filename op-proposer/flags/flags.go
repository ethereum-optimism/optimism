package flags

import (
	"fmt"
	"time"

	"github.com/urfave/cli/v2"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

const EnvVarPrefix = "OP_PROPOSER"

func prefixEnvVars(name string) []string {
	return opservice.PrefixEnvVar(EnvVarPrefix, name)
}

var (
	// Required Flags

	// L1 Ethereum RPC URL (required).
	L1EthRpcFlag = &cli.StringFlag{
		Name:    "l1-eth-rpc",
		Usage:   "HTTP provider URL for L1",
		EnvVars: prefixEnvVars("L1_ETH_RPC"),
	}

	// Optional flags

	// Rollup RPC URL (optional).
	//
	// MAY be a comma-separated list of URLs.
	RollupRpcFlag = &cli.StringFlag{
		Name:    "rollup-rpc",
		Usage:   "HTTP provider URL for the rollup node. A comma-separated list enables the active rollup provider.",
		EnvVars: prefixEnvVars("ROLLUP_RPC"),
	}

	// Supervisor RPC URLs (optional).
	//
	// MAY be a comma-separated list of URLs.
	SupervisorRpcsFlag = &cli.StringSliceFlag{
		Name:    "supervisor-rpcs",
		Usage:   "HTTP provider URLs for the supervisor nodes. Multiple URLs can be provided to automatically fail over.",
		EnvVars: prefixEnvVars("SUPERVISOR_RPCS"),
	}

	// L2 output orgcle contract address (optional).
	L2OOAddressFlag = &cli.StringFlag{
		Name:    "l2oo-address",
		Usage:   "Address of the L2OutputOracle contract",
		EnvVars: prefixEnvVars("L2OO_ADDRESS"),
	}

	// Poll interval for output root proposals (optional).
	PollIntervalFlag = &cli.DurationFlag{
		Name:    "poll-interval",
		Usage:   "Delay between periodic checks on whether it is time to load an output root and propose it.",
		Value:   12 * time.Second,
		EnvVars: prefixEnvVars("POLL_INTERVAL"),
	}

	// Allow proposals for non-finalized L1 blocks (optional).
	AllowNonFinalizedFlag = &cli.BoolFlag{
		Name:    "allow-non-finalized",
		Usage:   "Allow the proposer to submit proposals for L2 blocks derived from non-finalized L1 blocks.",
		EnvVars: prefixEnvVars("ALLOW_NON_FINALIZED"),
	}

	// Dispute game factory contract address (optional).
	DisputeGameFactoryAddressFlag = &cli.StringFlag{
		Name:    "game-factory-address",
		Usage:   "Address of the DisputeGameFactory contract",
		EnvVars: prefixEnvVars("GAME_FACTORY_ADDRESS"),
	}

	// Interval between L2 proposals (only when DisputeGameFactoryAddress is set) (optional).
	ProposalIntervalFlag = &cli.DurationFlag{
		Name:    "proposal-interval",
		Usage:   "Interval between submitting L2 output proposals when the dispute game factory address is set",
		EnvVars: prefixEnvVars("PROPOSAL_INTERVAL"),
	}

	// Dispute game type (only when DisputeGameFactoryAddress is set) (optional).
	DisputeGameTypeFlag = &cli.UintFlag{
		Name:    "game-type",
		Usage:   "Dispute game type to create via the configured DisputeGameFactory",
		Value:   0,
		EnvVars: prefixEnvVars("GAME_TYPE"),
	}

	// Time between checks for active sequencer (optional).
	ActiveSequencerCheckDurationFlag = &cli.DurationFlag{
		Name:    "active-sequencer-check-duration",
		Usage:   "The duration between checks to determine the active sequencer endpoint.",
		Value:   2 * time.Minute,
		EnvVars: prefixEnvVars("ACTIVE_SEQUENCER_CHECK_DURATION"),
	}

	// Wait for node sync before starting (optional).
	WaitNodeSyncFlag = &cli.BoolFlag{
		Name: "wait-node-sync",
		Usage: "Indicates if, during startup, the proposer should wait for the rollup node to sync to " +
			"the current L1 tip before proceeding with its driver loop.",
		Value:   false,
		EnvVars: prefixEnvVars("WAIT_NODE_SYNC"),
	}

	// Legacy Flags
	L2OutputHDPathFlag = txmgr.L2OutputHDPathFlag
)

var requiredFlags = []cli.Flag{
	L1EthRpcFlag,
}

var optionalFlags = []cli.Flag{
	RollupRpcFlag,
	SupervisorRpcsFlag,
	L2OOAddressFlag,
	PollIntervalFlag,
	AllowNonFinalizedFlag,
	L2OutputHDPathFlag,
	DisputeGameFactoryAddressFlag,
	ProposalIntervalFlag,
	DisputeGameTypeFlag,
	ActiveSequencerCheckDurationFlag,
	WaitNodeSyncFlag,
}

// Appends RPC, logger, metrics, profile, and tx manager flags to local flags.
func init() {
	optionalFlags = append(optionalFlags, oprpc.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, oplog.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, opmetrics.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, oppprof.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, txmgr.CLIFlags(EnvVarPrefix)...)

	Flags = append(requiredFlags, optionalFlags...)
}

// Flags contains the list of configuration options available to the binary.
var Flags []cli.Flag

// Checks for required flags.
func CheckRequired(ctx *cli.Context) error {
	for _, f := range requiredFlags {
		if !ctx.IsSet(f.Names()[0]) {
			return fmt.Errorf("flag %s is required", f.Names()[0])
		}
	}
	return nil
}
