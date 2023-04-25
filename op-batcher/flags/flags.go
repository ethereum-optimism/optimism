package flags

import (
	"fmt"
	"time"

	"github.com/urfave/cli"

	"github.com/ethereum-optimism/optimism/op-batcher/rpc"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

const envVarPrefix = "OP_BATCHER"

var (
	// Required flags
	L1EthRpcFlag = cli.StringFlag{
		Name:   "l1-eth-rpc",
		Usage:  "HTTP provider URL for L1",
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "L1_ETH_RPC"),
	}
	L2EthRpcFlag = cli.StringFlag{
		Name:   "l2-eth-rpc",
		Usage:  "HTTP provider URL for L2 execution engine",
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "L2_ETH_RPC"),
	}
	RollupRpcFlag = cli.StringFlag{
		Name:   "rollup-rpc",
		Usage:  "HTTP provider URL for Rollup node",
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "ROLLUP_RPC"),
	}
	// Optional flags
	SubSafetyMarginFlag = cli.Uint64Flag{
		Name: "sub-safety-margin",
		Usage: "The batcher tx submission safety margin (in #L1-blocks) to subtract " +
			"from a channel's timeout and sequencing window, to guarantee safe inclusion " +
			"of a channel on L1.",
		Value:  10,
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "SUB_SAFETY_MARGIN"),
	}
	PollIntervalFlag = cli.DurationFlag{
		Name:   "poll-interval",
		Usage:  "How frequently to poll L2 for new blocks",
		Value:  6 * time.Second,
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "POLL_INTERVAL"),
	}
	MaxPendingTransactionsFlag = cli.Uint64Flag{
		Name:   "max-pending-tx",
		Usage:  "The maximum number of pending transactions. 0 for no limit.",
		Value:  1,
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "MAX_PENDING_TX"),
	}
	MaxChannelDurationFlag = cli.Uint64Flag{
		Name:   "max-channel-duration",
		Usage:  "The maximum duration of L1-blocks to keep a channel open. 0 to disable.",
		Value:  0,
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "MAX_CHANNEL_DURATION"),
	}
	MaxL1TxSizeBytesFlag = cli.Uint64Flag{
		Name:   "max-l1-tx-size-bytes",
		Usage:  "The maximum size of a batch tx submitted to L1.",
		Value:  120_000,
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "MAX_L1_TX_SIZE_BYTES"),
	}
	TargetL1TxSizeBytesFlag = cli.Uint64Flag{
		Name:   "target-l1-tx-size-bytes",
		Usage:  "The target size of a batch tx submitted to L1.",
		Value:  100_000,
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "TARGET_L1_TX_SIZE_BYTES"),
	}
	TargetNumFramesFlag = cli.IntFlag{
		Name:   "target-num-frames",
		Usage:  "The target number of frames to create per channel",
		Value:  1,
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "TARGET_NUM_FRAMES"),
	}
	ApproxComprRatioFlag = cli.Float64Flag{
		Name:   "approx-compr-ratio",
		Usage:  "The approximate compression ratio (<= 1.0)",
		Value:  0.4,
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "APPROX_COMPR_RATIO"),
	}
	StoppedFlag = cli.BoolFlag{
		Name:   "stopped",
		Usage:  "Initialize the batcher in a stopped state. The batcher can be started using the admin_startBatcher RPC",
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "STOPPED"),
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
	TargetL1TxSizeBytesFlag,
	TargetNumFramesFlag,
	ApproxComprRatioFlag,
	StoppedFlag,
}

func init() {
	optionalFlags = append(optionalFlags, oprpc.CLIFlags(envVarPrefix)...)
	optionalFlags = append(optionalFlags, oplog.CLIFlags(envVarPrefix)...)
	optionalFlags = append(optionalFlags, opmetrics.CLIFlags(envVarPrefix)...)
	optionalFlags = append(optionalFlags, oppprof.CLIFlags(envVarPrefix)...)
	optionalFlags = append(optionalFlags, rpc.CLIFlags(envVarPrefix)...)
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
