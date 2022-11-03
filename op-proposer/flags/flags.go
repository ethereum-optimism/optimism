package flags

import (
	"github.com/urfave/cli"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
)

const envVarPrefix = "OP_PROPOSER"

var (
	/* Required Flags */

	L1EthRpcFlag = cli.StringFlag{
		Name:     "l1-eth-rpc",
		Usage:    "HTTP provider URL for L1",
		Required: true,
		EnvVar:   opservice.PrefixEnvVar(envVarPrefix, "L1_ETH_RPC"),
	}
	RollupRpcFlag = cli.StringFlag{
		Name:     "rollup-rpc",
		Usage:    "HTTP provider URL for the rollup node",
		Required: true,
		EnvVar:   opservice.PrefixEnvVar(envVarPrefix, "ROLLUP_RPC"),
	}
	L2OOAddressFlag = cli.StringFlag{
		Name:     "l2oo-address",
		Usage:    "Address of the L2OutputOracle contract",
		Required: true,
		EnvVar:   opservice.PrefixEnvVar(envVarPrefix, "L2OO_ADDRESS"),
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

	MnemonicFlag = cli.StringFlag{
		Name: "mnemonic",
		Usage: "The mnemonic used to derive the wallets for either the " +
			"sequencer or the l2output",
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "MNEMONIC"),
	}
	L2OutputHDPathFlag = cli.StringFlag{
		Name: "l2-output-hd-path",
		Usage: "The HD path used to derive the l2output wallet from the " +
			"mnemonic. The mnemonic flag must also be set.",
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "L2_OUTPUT_HD_PATH"),
	}
	PrivateKeyFlag = cli.StringFlag{
		Name:   "private-key",
		Usage:  "The private key to use with the l2output wallet. Must not be used with mnemonic.",
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "PRIVATE_KEY"),
	}
	AllowNonFinalizedFlag = cli.BoolFlag{
		Name:   "allow-non-finalized",
		Usage:  "Allow the proposer to submit proposals for L2 blocks derived from non-finalized L1 blocks.",
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "ALLOW_NON_FINALIZED"),
	}
)

var requiredFlags = []cli.Flag{
	L1EthRpcFlag,
	RollupRpcFlag,
	L2OOAddressFlag,
	PollIntervalFlag,
	NumConfirmationsFlag,
	SafeAbortNonceTooLowCountFlag,
	ResubmissionTimeoutFlag,
}

var optionalFlags = []cli.Flag{
	MnemonicFlag,
	L2OutputHDPathFlag,
	PrivateKeyFlag,
	AllowNonFinalizedFlag,
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
