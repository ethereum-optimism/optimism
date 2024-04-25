package flags

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-batcher/compressor"
	plasma "github.com/ethereum-optimism/optimism/op-plasma"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	openum "github.com/ethereum-optimism/optimism/op-service/enum"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

const EnvVarPrefix = "OP_BATCHER"

func prefixEnvVars(name string) []string {
	return opservice.PrefixEnvVar(EnvVarPrefix, name)
}

var (
	// Required flags
	L1EthRpcFlag = &cli.StringFlag{
		Name:    "l1-eth-rpc",
		Usage:   "HTTP provider URL for L1",
		EnvVars: prefixEnvVars("L1_ETH_RPC"),
	}
	L2EthRpcFlag = &cli.StringFlag{
		Name:    "l2-eth-rpc",
		Usage:   "HTTP provider URL for L2 execution engine. A comma-separated list enables the active L2 endpoint provider. Such a list needs to match the number of rollup-rpcs provided.",
		EnvVars: prefixEnvVars("L2_ETH_RPC"),
	}
	RollupRpcFlag = &cli.StringFlag{
		Name:    "rollup-rpc",
		Usage:   "HTTP provider URL for Rollup node. A comma-separated list enables the active L2 endpoint provider. Such a list needs to match the number of l2-eth-rpcs provided.",
		EnvVars: prefixEnvVars("ROLLUP_RPC"),
	}
	// Optional flags
	SubSafetyMarginFlag = &cli.Uint64Flag{
		Name: "sub-safety-margin",
		Usage: "The batcher tx submission safety margin (in #L1-blocks) to subtract " +
			"from a channel's timeout and sequencing window, to guarantee safe inclusion " +
			"of a channel on L1.",
		Value:   10,
		EnvVars: prefixEnvVars("SUB_SAFETY_MARGIN"),
	}
	PollIntervalFlag = &cli.DurationFlag{
		Name:    "poll-interval",
		Usage:   "How frequently to poll L2 for new blocks",
		Value:   6 * time.Second,
		EnvVars: prefixEnvVars("POLL_INTERVAL"),
	}
	MaxPendingTransactionsFlag = &cli.Uint64Flag{
		Name:    "max-pending-tx",
		Usage:   "The maximum number of pending transactions. 0 for no limit.",
		Value:   1,
		EnvVars: prefixEnvVars("MAX_PENDING_TX"),
	}
	MaxChannelDurationFlag = &cli.Uint64Flag{
		Name:    "max-channel-duration",
		Usage:   "The maximum duration of L1-blocks to keep a channel open. 0 to disable.",
		Value:   0,
		EnvVars: prefixEnvVars("MAX_CHANNEL_DURATION"),
	}
	MaxL1TxSizeBytesFlag = &cli.Uint64Flag{
		Name:    "max-l1-tx-size-bytes",
		Usage:   "The maximum size of a batch tx submitted to L1. Ignored for blobs, where max blob size will be used.",
		Value:   120_000, // will be overwritten to max for blob da-type
		EnvVars: prefixEnvVars("MAX_L1_TX_SIZE_BYTES"),
	}
	TargetNumFramesFlag = &cli.IntFlag{
		Name:    "target-num-frames",
		Usage:   "The target number of frames to create per channel. Controls number of blobs per blob tx, if using Blob DA.",
		Value:   1,
		EnvVars: prefixEnvVars("TARGET_NUM_FRAMES"),
	}
	ApproxComprRatioFlag = &cli.Float64Flag{
		Name:    "approx-compr-ratio",
		Usage:   "The approximate compression ratio (<= 1.0). Only relevant for ratio compressor.",
		Value:   0.6,
		EnvVars: prefixEnvVars("APPROX_COMPR_RATIO"),
	}
	CompressorFlag = &cli.StringFlag{
		Name:    "compressor",
		Usage:   "The type of compressor. Valid options: " + strings.Join(compressor.KindKeys, ", "),
		EnvVars: prefixEnvVars("COMPRESSOR"),
		Value:   compressor.ShadowKind,
		Action: func(_ *cli.Context, s string) error {
			if !slices.Contains(compressor.KindKeys, s) {
				return fmt.Errorf("unsupported compressor: %s", s)
			}
			return nil
		},
	}
	StoppedFlag = &cli.BoolFlag{
		Name:    "stopped",
		Usage:   "Initialize the batcher in a stopped state. The batcher can be started using the admin_startBatcher RPC",
		EnvVars: prefixEnvVars("STOPPED"),
	}
	BatchTypeFlag = &cli.UintFlag{
		Name:        "batch-type",
		Usage:       "The batch type. 0 for SingularBatch and 1 for SpanBatch.",
		Value:       0,
		EnvVars:     prefixEnvVars("BATCH_TYPE"),
		DefaultText: "singular",
	}
	DataAvailabilityTypeFlag = &cli.GenericFlag{
		Name: "data-availability-type",
		Usage: "The data availability type to use for submitting batches to the L1. Valid options: " +
			openum.EnumString(DataAvailabilityTypes),
		Value: func() *DataAvailabilityType {
			out := CalldataType
			return &out
		}(),
		EnvVars: prefixEnvVars("DATA_AVAILABILITY_TYPE"),
	}
	ActiveSequencerCheckDurationFlag = &cli.DurationFlag{
		Name:    "active-sequencer-check-duration",
		Usage:   "The duration between checks to determine the active sequencer endpoint. ",
		Value:   2 * time.Minute,
		EnvVars: prefixEnvVars("ACTIVE_SEQUENCER_CHECK_DURATION"),
	}
	CheckRecentTxsDepthFlag = &cli.IntFlag{
		Name: "check-recent-txs-depth",
		Usage: "Indicates how many blocks back the batcher should look during startup for a recent batch tx on L1. This can " +
			"speed up waiting for node sync. It should be set to the verifier confirmation depth of the sequencer (e.g. 4).",
		Value:   0,
		EnvVars: prefixEnvVars("CHECK_RECENT_TXS_DEPTH"),
	}
	WaitNodeSyncFlag = &cli.BoolFlag{
		Name: "wait-node-sync",
		Usage: "Indicates if, during startup, the batcher should wait for a recent batcher tx on L1 to " +
			"finalize (via more block confirmations). This should help avoid duplicate batcher txs.",
		Value:   false,
		EnvVars: prefixEnvVars("WAIT_NODE_SYNC"),
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
	WaitNodeSyncFlag,
	CheckRecentTxsDepthFlag,
	SubSafetyMarginFlag,
	PollIntervalFlag,
	MaxPendingTransactionsFlag,
	MaxChannelDurationFlag,
	MaxL1TxSizeBytesFlag,
	TargetNumFramesFlag,
	ApproxComprRatioFlag,
	CompressorFlag,
	StoppedFlag,
	SequencerHDPathFlag,
	BatchTypeFlag,
	DataAvailabilityTypeFlag,
	ActiveSequencerCheckDurationFlag,
}

func init() {
	optionalFlags = append(optionalFlags, oprpc.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, oplog.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, opmetrics.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, oppprof.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, txmgr.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, plasma.CLIFlags(EnvVarPrefix, "")...)

	Flags = append(requiredFlags, optionalFlags...)
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
