package flags

import (
	"github.com/urfave/cli"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
)

const envVarPrefix = "OP_BATCHER"

var (
	/* Required Flags */

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
	MinL1TxSizeBytesFlag = cli.Uint64Flag{
		Name:     "min-l1-tx-size-bytes",
		Usage:    "The minimum size of a batch tx submitted to L1.",
		Required: true,
		EnvVar:   opservice.PrefixEnvVar(envVarPrefix, "MIN_L1_TX_SIZE_BYTES"),
	}
	MaxL1TxSizeBytesFlag = cli.Uint64Flag{
		Name:     "max-l1-tx-size-bytes",
		Usage:    "The maximum size of a batch tx submitted to L1.",
		Required: true,
		EnvVar:   opservice.PrefixEnvVar(envVarPrefix, "MAX_L1_TX_SIZE_BYTES"),
	}
	ChannelTimeoutFlag = cli.Uint64Flag{
		Name:     "channel-timeout",
		Usage:    "The maximum amount of time to attempt completing an opened channel, as opposed to submitting L2 blocks into a new channel.",
		Required: true,
		EnvVar:   opservice.PrefixEnvVar(envVarPrefix, "CHANNEL_TIMEOUT"),
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
	SequencerBatchInboxAddressFlag = cli.StringFlag{
		Name:     "sequencer-batch-inbox-address",
		Usage:    "L1 Address to receive batch transactions",
		Required: true,
		EnvVar:   opservice.PrefixEnvVar(envVarPrefix, "SEQUENCER_BATCH_INBOX_ADDRESS"),
	}
)

var requiredFlags = []cli.Flag{
	L1EthRpcFlag,
	L2EthRpcFlag,
	RollupRpcFlag,
	MinL1TxSizeBytesFlag,
	MaxL1TxSizeBytesFlag,
	ChannelTimeoutFlag,
	PollIntervalFlag,
	NumConfirmationsFlag,
	SafeAbortNonceTooLowCountFlag,
	ResubmissionTimeoutFlag,
	SequencerBatchInboxAddressFlag,
}

var optionalFlags = []cli.Flag{
	MnemonicFlag,
	SequencerHDPathFlag,
	PrivateKeyFlag,
}

func init() {
	requiredFlags = append(requiredFlags, oprpc.CLIFlags(envVarPrefix)...)

	optionalFlags = append(optionalFlags, oplog.CLIFlags(envVarPrefix)...)
	optionalFlags = append(optionalFlags, opmetrics.CLIFlags(envVarPrefix)...)
	optionalFlags = append(optionalFlags, oppprof.CLIFlags(envVarPrefix)...)

	Flags = append(requiredFlags, optionalFlags...)
}

// Flags contains the list of configuration options available to the binary.
var Flags []cli.Flag
