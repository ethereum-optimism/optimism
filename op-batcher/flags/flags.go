package flags

import (
	"time"

	"github.com/urfave/cli"

	"github.com/ethereum-optimism/optimism/op-batcher/rpc"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	opsigner "github.com/ethereum-optimism/optimism/op-signer/client"
)

const envVarPrefix = "OP_BATCHER"

var (
	/* Required flags */

	L1EthRpcFlag = cli.StringFlag{
		Name:     "l1-eth-rpc",
		Usage:    "HTTP provider URL for L1",
		Required: true,
		EnvVar:   opservice.PrefixEnvVar(envVarPrefix, "L1_ETH_RPC"),
	}
	L2EthRpcFlag = cli.StringFlag{
		Name:     "l2-eth-rpc",
		Usage:    "HTTP provider URL for L2 execution engine",
		Required: true,
		EnvVar:   opservice.PrefixEnvVar(envVarPrefix, "L2_ETH_RPC"),
	}
	RollupRpcFlag = cli.StringFlag{
		Name:     "rollup-rpc",
		Usage:    "HTTP provider URL for Rollup node",
		Required: true,
		EnvVar:   opservice.PrefixEnvVar(envVarPrefix, "ROLLUP_RPC"),
	}
	SubSafetyMarginFlag = cli.Uint64Flag{
		Name: "sub-safety-margin",
		Usage: "The batcher tx submission safety margin (in #L1-blocks) to subtract " +
			"from a channel's timeout and sequencing window, to guarantee safe inclusion " +
			"of a channel on L1.",
		Required: true,
		EnvVar:   opservice.PrefixEnvVar(envVarPrefix, "SUB_SAFETY_MARGIN"),
	}
	PollIntervalFlag = cli.DurationFlag{
		Name: "poll-interval",
		Usage: "Delay between querying L2 for more transactions and " +
			"creating a new batch",
		Required: true,
		EnvVar:   opservice.PrefixEnvVar(envVarPrefix, "POLL_INTERVAL"),
	}
	NumConfirmationsFlag = cli.Uint64Flag{
		Name: "num-confirmations",
		Usage: "Number of confirmations which we will wait after " +
			"appending a new batch",
		Required: true,
		EnvVar:   opservice.PrefixEnvVar(envVarPrefix, "NUM_CONFIRMATIONS"),
	}
	SafeAbortNonceTooLowCountFlag = cli.Uint64Flag{
		Name: "safe-abort-nonce-too-low-count",
		Usage: "Number of ErrNonceTooLow observations required to " +
			"give up on a tx at a particular nonce without receiving " +
			"confirmation",
		Required: true,
		EnvVar:   opservice.PrefixEnvVar(envVarPrefix, "SAFE_ABORT_NONCE_TOO_LOW_COUNT"),
	}
	ResubmissionTimeoutFlag = cli.DurationFlag{
		Name: "resubmission-timeout",
		Usage: "Duration we will wait before resubmitting a " +
			"transaction to L1",
		Required: true,
		EnvVar:   opservice.PrefixEnvVar(envVarPrefix, "RESUBMISSION_TIMEOUT"),
	}

	/* Optional flags */
	OfflineGasEstimationFlag = cli.BoolFlag{
		Name:   "offline-gas-estimation",
		Usage:  "Whether to use offline gas estimation",
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "OFFLINE_GAS_ESTIMATION"),
	}
	TxManagerTimeoutFlag = cli.DurationFlag{
		Name:   "tx-manager-timeout",
		Usage:  "Maximum duration to wait for L1 transactions, including resubmissions",
		Value:  10 * time.Minute,
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "TX_MANAGER_TIMEOUT"),
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
	MnemonicFlag = cli.StringFlag{
		Name: "mnemonic",
		Usage: "The mnemonic used to derive the wallets for either the " +
			"sequencer or the l2output",
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "MNEMONIC"),
	}
	SequencerHDPathFlag = cli.StringFlag{
		Name: "sequencer-hd-path",
		Usage: "The HD path used to derive the sequencer wallet from the " +
			"mnemonic. The mnemonic flag must also be set.",
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "SEQUENCER_HD_PATH"),
	}
	PrivateKeyFlag = cli.StringFlag{
		Name:   "private-key",
		Usage:  "The private key to use with the l2output wallet. Must not be used with mnemonic.",
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "PRIVATE_KEY"),
	}
)

var requiredFlags = []cli.Flag{
	L1EthRpcFlag,
	L2EthRpcFlag,
	RollupRpcFlag,
	SubSafetyMarginFlag,
	PollIntervalFlag,
	NumConfirmationsFlag,
	SafeAbortNonceTooLowCountFlag,
	ResubmissionTimeoutFlag,
}

var optionalFlags = []cli.Flag{
	OfflineGasEstimationFlag,
	TxManagerTimeoutFlag,
	MaxChannelDurationFlag,
	MaxL1TxSizeBytesFlag,
	TargetL1TxSizeBytesFlag,
	TargetNumFramesFlag,
	ApproxComprRatioFlag,
	StoppedFlag,
	MnemonicFlag,
	SequencerHDPathFlag,
	PrivateKeyFlag,
}

func init() {
	requiredFlags = append(requiredFlags, oprpc.CLIFlags(envVarPrefix)...)

	optionalFlags = append(optionalFlags, oplog.CLIFlags(envVarPrefix)...)
	optionalFlags = append(optionalFlags, opmetrics.CLIFlags(envVarPrefix)...)
	optionalFlags = append(optionalFlags, oppprof.CLIFlags(envVarPrefix)...)
	optionalFlags = append(optionalFlags, opsigner.CLIFlags(envVarPrefix)...)
	optionalFlags = append(optionalFlags, rpc.CLIFlags(envVarPrefix)...)

	Flags = append(requiredFlags, optionalFlags...)
}

// Flags contains the list of configuration options available to the binary.
var Flags []cli.Flag
