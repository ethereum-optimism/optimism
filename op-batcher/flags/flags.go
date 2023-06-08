package flags

import (
	"fmt"
	"time"

	"github.com/urfave/cli"

	"github.com/ethereum-optimism/optimism/op-batcher/compressor"
	"github.com/ethereum-optimism/optimism/op-batcher/rpc"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

const EnvVarPrefix = "OP_BATCHER"

var (
	// Required flags
	L1EthRpcFlag = cli.StringFlag{
		Name:   "l1-eth-rpc",
		Usage:  "HTTP provider URL for L1",
		EnvVar: opservice.PrefixEnvVar(EnvVarPrefix, "L1_ETH_RPC"),
	}
	L2EthRpcFlag = cli.StringFlag{
		Name:   "l2-eth-rpc",
		Usage:  "HTTP provider URL for L2 execution engine",
		EnvVar: opservice.PrefixEnvVar(EnvVarPrefix, "L2_ETH_RPC"),
	}
	RollupRpcFlag = cli.StringFlag{
		Name:   "rollup-rpc",
		Usage:  "HTTP provider URL for Rollup node",
		EnvVar: opservice.PrefixEnvVar(EnvVarPrefix, "ROLLUP_RPC"),
	}
	// Optional flags
	SubSafetyMarginFlag = cli.Uint64Flag{
		Name: "sub-safety-margin",
		Usage: "The batcher tx submission safety margin (in #L1-blocks) to subtract " +
			"from a channel's timeout and sequencing window, to guarantee safe inclusion " +
			"of a channel on L1.",
		Value:  10,
		EnvVar: opservice.PrefixEnvVar(EnvVarPrefix, "SUB_SAFETY_MARGIN"),
	}
	PollIntervalFlag = cli.DurationFlag{
		Name:   "poll-interval",
		Usage:  "How frequently to poll L2 for new blocks",
		Value:  6 * time.Second,
		EnvVar: opservice.PrefixEnvVar(EnvVarPrefix, "POLL_INTERVAL"),
	}
	MaxPendingTransactionsFlag = cli.Uint64Flag{
		Name:   "max-pending-tx",
		Usage:  "The maximum number of pending transactions. 0 for no limit.",
		Value:  1,
		EnvVar: opservice.PrefixEnvVar(EnvVarPrefix, "MAX_PENDING_TX"),
	}
	MaxChannelDurationFlag = cli.Uint64Flag{
		Name:   "max-channel-duration",
		Usage:  "The maximum duration of L1-blocks to keep a channel open. 0 to disable.",
		Value:  0,
		EnvVar: opservice.PrefixEnvVar(EnvVarPrefix, "MAX_CHANNEL_DURATION"),
	}
	MaxL1TxSizeBytesFlag = cli.Uint64Flag{
		Name:   "max-l1-tx-size-bytes",
		Usage:  "The maximum size of a batch tx submitted to L1.",
		Value:  120_000,
		EnvVar: opservice.PrefixEnvVar(EnvVarPrefix, "MAX_L1_TX_SIZE_BYTES"),
	}
	StoppedFlag = cli.BoolFlag{
		Name:   "stopped",
		Usage:  "Initialize the batcher in a stopped state. The batcher can be started using the admin_startBatcher RPC",
		EnvVar: opservice.PrefixEnvVar(EnvVarPrefix, "STOPPED"),
	}
	// Legacy Flags
	SequencerHDPathFlag = txmgr.SequencerHDPathFlag
)

var requiredFlags = []cli.Flag{
	L1EthRpcFlag,
	L2EthRpcFlag,
	RollupRpcFlag,
}

var optionalFlags = []cli.Flag{
	SubSafetyMarginFlag,
	PollIntervalFlag,
	MaxPendingTransactionsFlag,
	MaxChannelDurationFlag,
	MaxL1TxSizeBytesFlag,
	StoppedFlag,
	SequencerHDPathFlag,
}

func init() {
	optionalFlags = append(optionalFlags, oprpc.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, oplog.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, opmetrics.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, oppprof.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, rpc.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, txmgr.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, compressor.CLIFlags(EnvVarPrefix)...)

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
