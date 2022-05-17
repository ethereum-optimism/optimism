package flags

import (
	"github.com/urfave/cli"
)

const envVarPrefix = "OUTPUT_SUBMITTER_"

func prefixEnvVar(name string) string {
	return envVarPrefix + name
}

var (
	/* Required Flags */

	L1EthRpcFlag = cli.StringFlag{
		Name:     "l1-eth-rpc",
		Usage:    "HTTP provider URL for L1",
		Required: true,
		EnvVar:   "L1_ETH_RPC",
	}
	L2EthRpcFlag = cli.StringFlag{
		Name:     "l2-eth-rpc",
		Usage:    "HTTP provider URL for L2",
		Required: true,
		EnvVar:   "L2_ETH_RPC",
	}
	RollupRpcFlag = cli.StringFlag{
		Name:     "rollup-rpc",
		Usage:    "HTTP provider URL for the rollup node",
		Required: true,
		EnvVar:   "ROLLUP_RPC",
	}
	L2OOAddressFlag = cli.StringFlag{
		Name:     "l2oo-address",
		Usage:    "Address of the L2OutputOracle contract",
		Required: true,
		EnvVar:   "L2OO_ADDRESS",
	}
	PollIntervalFlag = cli.DurationFlag{
		Name: "poll-interval",
		Usage: "Delay between querying L2 for more transactions and " +
			"creating a new batch",
		Required: true,
		EnvVar:   prefixEnvVar("POLL_INTERVAL"),
	}
	NumConfirmationsFlag = cli.Uint64Flag{
		Name: "num-confirmations",
		Usage: "Number of confirmations which we will wait after " +
			"appending a new batch",
		Required: true,
		EnvVar:   prefixEnvVar("NUM_CONFIRMATIONS"),
	}
	SafeAbortNonceTooLowCountFlag = cli.Uint64Flag{
		Name: "safe-abort-nonce-too-low-count",
		Usage: "Number of ErrNonceTooLow observations required to " +
			"give up on a tx at a particular nonce without receiving " +
			"confirmation",
		Required: true,
		EnvVar:   prefixEnvVar("SAFE_ABORT_NONCE_TOO_LOW_COUNT"),
	}
	ResubmissionTimeoutFlag = cli.DurationFlag{
		Name: "resubmission-timeout",
		Usage: "Duration we will wait before resubmitting a " +
			"transaction to L1",
		Required: true,
		EnvVar:   prefixEnvVar("RESUBMISSION_TIMEOUT"),
	}
	MnemonicFlag = cli.StringFlag{
		Name: "mnemonic",
		Usage: "The mnemonic used to derive the wallets for either the " +
			"sequencer or the l2output",
		Required: true,
		EnvVar:   prefixEnvVar("MNEMONIC"),
	}
	L2OutputHDPathFlag = cli.StringFlag{
		Name: "l2-output-hd-path",
		Usage: "The HD path used to derive the l2output wallet from the " +
			"mnemonic. The mnemonic flag must also be set.",
		Required: true,
		EnvVar:   prefixEnvVar("L2_OUTPUT_HD_PATH"),
	}

	/* Optional Flags */

	LogLevelFlag = cli.StringFlag{
		Name:   "log-level",
		Usage:  "The lowest log level that will be output",
		Value:  "info",
		EnvVar: prefixEnvVar("LOG_LEVEL"),
	}
	LogTerminalFlag = cli.BoolFlag{
		Name: "log-terminal",
		Usage: "If true, outputs logs in terminal format, otherwise prints " +
			"in JSON format.",
		EnvVar: prefixEnvVar("LOG_TERMINAL"),
	}
)

var requiredFlags = []cli.Flag{
	L1EthRpcFlag,
	L2EthRpcFlag,
	RollupRpcFlag,
	L2OOAddressFlag,
	PollIntervalFlag,
	NumConfirmationsFlag,
	SafeAbortNonceTooLowCountFlag,
	ResubmissionTimeoutFlag,
	MnemonicFlag,
	L2OutputHDPathFlag,
}

var optionalFlags = []cli.Flag{
	LogLevelFlag,
	LogTerminalFlag,
}

// Flags contains the list of configuration options available to the binary.
var Flags = append(requiredFlags, optionalFlags...)
