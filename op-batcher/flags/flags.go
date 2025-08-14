package flags

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	"github.com/ethereum-optimism/optimism/op-batcher/compressor"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	openum "github.com/ethereum-optimism/optimism/op-service/enum"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

const (
	EnvVarPrefix = "OP_BATCHER"

	// Throttling
	DefaultThrottleThreshold    = 1_600_000 // allows for 2x (6 blobs, 1 tx) channels at ~131KB per blob
	DefaultThrottleMaxThreshold = DefaultThrottleThreshold * 5
	DefaultPIDSampleTime        = 2 * time.Second
	DefaultPIDKp                = 0.33
	DefaultPIDKi                = 0.01
	DefaultPIDKd                = 0.05
	DefaultPIDIntegralMax       = 1000.0
	DefaultPIDOutputMax         = 1.0
)

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
	L2EthRpcFlag = &cli.StringSliceFlag{
		Name:    "l2-eth-rpc",
		Usage:   "HTTP provider URL for L2 execution engine. A comma-separated list enables the active L2 endpoint provider. Such a list needs to match the number of rollup-rpcs provided.",
		EnvVars: prefixEnvVars("L2_ETH_RPC"),
	}
	RollupRpcFlag = &cli.StringSliceFlag{
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
	MaxBlocksPerSpanBatch = &cli.IntFlag{
		Name:    "max-blocks-per-span-batch",
		Usage:   "Maximum number of blocks to add to a span batch. Default is 0 - no maximum.",
		EnvVars: prefixEnvVars("MAX_BLOCKS_PER_SPAN_BATCH"),
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
	CompressionAlgoFlag = &cli.GenericFlag{
		Name:    "compression-algo",
		Usage:   "The compression algorithm to use. Valid options: " + openum.EnumString(derive.CompressionAlgos),
		EnvVars: prefixEnvVars("COMPRESSION_ALGO"),
		Value: func() *derive.CompressionAlgo {
			out := derive.Zlib
			return &out
		}(),
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
		Usage:   "The duration between checks to determine the active sequencer endpoint.",
		Value:   5 * time.Second,
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
	ThrottleThresholdFlag = &cli.IntFlag{
		Name:    "throttle-threshold",
		Usage:   "The threshold on unsafe_da_bytes beyond which the batcher will start to throttle the block builder. Zero disables throttling.",
		Value:   1_000_000,
		EnvVars: prefixEnvVars("THROTTLE_THRESHOLD"),
	}
	ThrottleMaxThresholdFlag = &cli.Uint64Flag{
		Name:    "throttle-max-threshold",
		Usage:   "Threshold at which throttling has the maximum intensity (linear and quadratic controllers only)",
		Value:   DefaultThrottleMaxThreshold,
		EnvVars: prefixEnvVars("THROTTLE_MAX_THRESHOLD"),
	}
	ThrottleTxSizeFlag = &cli.IntFlag{
		Name:    "throttle-tx-size",
		Usage:   "The DA size of transactions to start throttling when we are over the throttle threshold",
		Value:   5000, // less than 1% of all transactions should be affected by this limit
		EnvVars: prefixEnvVars("THROTTLE_TX_SIZE"),
	}
	ThrottleBlockSizeFlag = &cli.IntFlag{
		Name:    "throttle-block-size",
		Usage:   "The total DA limit to start imposing on block building when we are over the throttle threshold",
		Value:   21_000, // at least 70 transactions per block of up to 300 compressed bytes each.
		EnvVars: prefixEnvVars("THROTTLE_BLOCK_SIZE"),
	}
	ThrottleAlwaysBlockSizeFlag = &cli.IntFlag{
		Name:    "throttle-always-block-size",
		Usage:   "The total DA limit to start imposing on block building at all times",
		Value:   130_000, // should be larger than the builder's max-l2-tx-size to prevent endlessly throttling some txs
		EnvVars: prefixEnvVars("THROTTLE_ALWAYS_BLOCK_SIZE"),
	}
	AdditionalThrottlingEndpointsFlag = &cli.StringSliceFlag{
		Name:    "additional-throttling-endpoints",
		Usage:   "Comma-separated list of endpoints to distribute throttling configuration to (in addition to the L2 endpoints specified with --l2-eth-rpc).",
		EnvVars: prefixEnvVars("ADDITIONAL_THROTTLING_ENDPOINTS"),
	}
	ThrottleControllerTypeFlag = &cli.StringFlag{
		Name:    "throttle-controller-type",
		Usage:   "Type of throttle controller to use: 'step', 'linear', 'quadratic' (default) or 'pid' (EXPERIMENTAL - use with caution)",
		Value:   "quadratic",
		EnvVars: prefixEnvVars("THROTTLE_CONTROLLER_TYPE"),
		Action: func(ctx *cli.Context, value string) error {
			validTypes := []string{"step", "linear", "quadratic", "pid"}
			for _, validType := range validTypes {
				if value == validType {
					return nil
				}
			}
			return fmt.Errorf("throttle-controller-type must be one of %v, got %s", validTypes, value)
		},
	}
	ThrottlePidKpFlag = &cli.Float64Flag{
		Name:    "throttle-pid-kp",
		Usage:   "EXPERIMENTAL: PID controller proportional gain. Only relevant if --throttle-controller-type is set to 'pid'",
		Value:   DefaultPIDKp,
		EnvVars: prefixEnvVars("THROTTLE_PID_KP"),
		Action: func(ctx *cli.Context, value float64) error {
			if value < 0 {
				return fmt.Errorf("throttle-pid-kp must be >= 0, got %f", value)
			}
			return nil
		},
	}
	ThrottlePidKiFlag = &cli.Float64Flag{
		Name:    "throttle-pid-ki",
		Usage:   "EXPERIMENTAL: PID controller integral gain. Only relevant if --throttle-controller-type is set to 'pid'",
		Value:   DefaultPIDKi,
		EnvVars: prefixEnvVars("THROTTLE_PID_KI"),
		Action: func(ctx *cli.Context, value float64) error {
			if value < 0 {
				return fmt.Errorf("throttle-pid-ki must be >= 0, got %f", value)
			}
			return nil
		},
	}
	ThrottlePidKdFlag = &cli.Float64Flag{
		Name:    "throttle-pid-kd",
		Usage:   "EXPERIMENTAL: PID controller derivative gain. Only relevant if --throttle-controller-type is set to 'pid'",
		Value:   DefaultPIDKd,
		EnvVars: prefixEnvVars("THROTTLE_PID_KD"),
		Action: func(ctx *cli.Context, value float64) error {
			if value < 0 {
				return fmt.Errorf("throttle-pid-kd must be >= 0, got %f", value)
			}
			return nil
		},
	}
	ThrottlePidIntegralMaxFlag = &cli.Float64Flag{
		Name:    "throttle-pid-integral-max",
		Usage:   "EXPERIMENTAL: PID controller maximum integral windup. Only relevant if --throttle-controller-type is set to 'pid'",
		Value:   DefaultPIDIntegralMax,
		EnvVars: prefixEnvVars("THROTTLE_PID_INTEGRAL_MAX"),
		Action: func(ctx *cli.Context, value float64) error {
			if value <= 0 {
				return fmt.Errorf("throttle-pid-integral-max must be > 0, got %f", value)
			}
			return nil
		},
	}
	ThrottlePidOutputMaxFlag = &cli.Float64Flag{
		Name:    "throttle-pid-output-max",
		Usage:   "EXPERIMENTAL: PID controller maximum output. Only relevant if --throttle-controller-type is set to 'pid'",
		Value:   DefaultPIDOutputMax,
		EnvVars: prefixEnvVars("THROTTLE_PID_OUTPUT_MAX"),
		Action: func(ctx *cli.Context, value float64) error {
			if value <= 0 || value > 1.0 {
				return fmt.Errorf("throttle-pid-output-max must be between 0 and 1, got %f", value)
			}
			return nil
		},
	}
	ThrottlePidSampleTimeFlag = &cli.DurationFlag{
		Name:    "throttle-pid-sample-time",
		Usage:   "EXPERIMENTAL: PID controller sample time interval, default is " + DefaultPIDSampleTime.String(),
		Value:   DefaultPIDSampleTime,
		EnvVars: prefixEnvVars("THROTTLE_PID_SAMPLE_TIME"),
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
	MaxBlocksPerSpanBatch,
	TargetNumFramesFlag,
	ApproxComprRatioFlag,
	CompressorFlag,
	StoppedFlag,
	SequencerHDPathFlag,
	BatchTypeFlag,
	DataAvailabilityTypeFlag,
	ActiveSequencerCheckDurationFlag,
	CompressionAlgoFlag,
	ThrottleThresholdFlag,
	ThrottleTxSizeFlag,
	ThrottleBlockSizeFlag,
	ThrottleAlwaysBlockSizeFlag,
	AdditionalThrottlingEndpointsFlag,
	ThrottleControllerTypeFlag,
	ThrottlePidKpFlag,
	ThrottlePidKiFlag,
	ThrottlePidKdFlag,
	ThrottlePidIntegralMaxFlag,
	ThrottlePidOutputMaxFlag,
	ThrottlePidSampleTimeFlag,
	ThrottleMaxThresholdFlag,
}

func init() {
	optionalFlags = append(optionalFlags, oprpc.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, oplog.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, opmetrics.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, oppprof.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, txmgr.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, altda.CLIFlags(EnvVarPrefix, "")...)

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
