package flags

import "github.com/urfave/cli"

const envVarPrefix = "TELEPORTR_"

func prefixEnvVar(name string) string {
	return envVarPrefix + name
}

var (
	/* Required Flags */

	BuildEnvFlag = cli.StringFlag{
		Name: "build-env",
		Usage: "Build environment for which the binary is produced, " +
			"e.g. production or development",
		Required: true,
		EnvVar:   "BUILD_ENV",
	}
	EthNetworkNameFlag = cli.StringFlag{
		Name:     "eth-network-name",
		Usage:    "Ethereum network name",
		Required: true,
		EnvVar:   "ETH_NETWORK_NAME",
	}
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
	DepositAddressFlag = cli.StringFlag{
		Name:     "deposit-address",
		Usage:    "Address of the TeleportrDeposit contract",
		Required: true,
		EnvVar:   prefixEnvVar("DEPOSIT_ADDRESS"),
	}
	DepositDeployBlockNumberFlag = cli.Uint64Flag{
		Name:     "deposit-deploy-block-number",
		Usage:    "Deployment block number of the TeleportrDeposit contract",
		Required: true,
		EnvVar:   prefixEnvVar("DEPOSIT_DEPLOY_BLOCK_NUMBER"),
	}
	DisburserAddressFlag = cli.StringFlag{
		Name:     "disburser-address",
		Usage:    "Address of the TeleportrDisburser contract",
		Required: true,
		EnvVar:   prefixEnvVar("DISBURSER_ADDRESS"),
	}
	MaxL2TxSizeFlag = cli.Uint64Flag{
		Name: "max-l2-tx-size",
		Usage: "Maximum size in bytes of any L2 transaction that gets " +
			"sent for disbursement",
		Required: true,
		EnvVar:   prefixEnvVar("MAX_L2_TX_SIZE"),
	}
	NumDepositConfirmationsFlag = cli.Uint64Flag{
		Name: "num-deposit-confirmations",
		Usage: "Number of confirmations before deposits are considered " +
			"confirmed",
		Required: true,
		EnvVar:   prefixEnvVar("NUM_DEPOSIT_CONFIRMATIONS"),
	}
	FilterQueryMaxBlocksFlag = cli.Uint64Flag{
		Name:     "filter-query-max-blocks",
		Usage:    "Maximum range of a filter query in blocks",
		Required: true,
		EnvVar:   prefixEnvVar("FILTER_QUERY_MAX_BLOCKS"),
	}
	PollIntervalFlag = cli.DurationFlag{
		Name: "poll-interval",
		Usage: "Delay between querying L1 for more transactions and " +
			"creating a new disbursement batch",
		Required: true,
		EnvVar:   prefixEnvVar("POLL_INTERVAL"),
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
			"transaction to L2",
		Required: true,
		EnvVar:   prefixEnvVar("RESUBMISSION_TIMEOUT"),
	}
	PostgresHostFlag = cli.StringFlag{
		Name:     "postgres-host",
		Usage:    "Host of the teleportr postgres instance",
		Required: true,
		EnvVar:   prefixEnvVar("POSTGRES_HOST"),
	}
	PostgresPortFlag = cli.Uint64Flag{
		Name:     "postgres-port",
		Usage:    "Port of the teleportr postgres instance",
		Required: true,
		EnvVar:   prefixEnvVar("POSTGRES_PORT"),
	}
	PostgresUserFlag = cli.StringFlag{
		Name:     "postgres-user",
		Usage:    "Username of the teleportr postgres instance",
		Required: true,
		EnvVar:   prefixEnvVar("POSTGRES_USER"),
	}
	PostgresPasswordFlag = cli.StringFlag{
		Name:     "postgres-password",
		Usage:    "Password of the teleportr postgres instance",
		Required: true,
		EnvVar:   prefixEnvVar("POSTGRES_PASSWORD"),
	}
	PostgresDBNameFlag = cli.StringFlag{
		Name:     "postgres-db-name",
		Usage:    "Database name of the teleportr postgres instance",
		Required: true,
		EnvVar:   prefixEnvVar("POSTGRES_DB_NAME"),
	}

	/* Optional Flags */

	PostgresEnableSSLFlag = cli.BoolFlag{
		Name: "postgres-enable-ssl",
		Usage: "Whether or not to enable SSL on connections to " +
			"teleportr postgres instance",
		EnvVar: prefixEnvVar("POSTGRES_ENABLE_SSL"),
	}
	LogLevelFlag = cli.StringFlag{
		Name:   "log-level",
		Usage:  "The lowest log level that will be output",
		Value:  "info",
		EnvVar: prefixEnvVar("LOG_LEVEL"),
	}
	LogTerminalFlag = cli.BoolFlag{
		Name: "log-terminal",
		Usage: "If true, outputs logs in terminal format, otherwise prints " +
			"in JSON format. If SENTRY_ENABLE is set to true, this flag is " +
			"ignored and logs are printed using JSON",
		EnvVar: prefixEnvVar("LOG_TERMINAL"),
	}
	DisburserPrivateKeyFlag = cli.StringFlag{
		Name:   "disburser-private-key",
		Usage:  "The private key to use for sending to the disburser contract",
		EnvVar: prefixEnvVar("DISBURSER_PRIVATE_KEY"),
	}
	MnemonicFlag = cli.StringFlag{
		Name:   "mnemonic",
		Usage:  "The mnemonic used to derive the wallet for the disburser",
		EnvVar: prefixEnvVar("MNEMONIC"),
	}
	DisburserHDPathFlag = cli.StringFlag{
		Name: "disburser-hd-path",
		Usage: "The HD path used to derive the disburser wallet from the " +
			"mnemonic. The mnemonic flag must also be set.",
		EnvVar: prefixEnvVar("DISBURSER_HD_PATH"),
	}
	MetricsServerEnableFlag = cli.BoolFlag{
		Name:   "metrics-server-enable",
		Usage:  "Whether or not to run the embedded metrics server",
		EnvVar: prefixEnvVar("METRICS_SERVER_ENABLE"),
	}
	MetricsHostnameFlag = cli.StringFlag{
		Name:   "metrics-hostname",
		Usage:  "The hostname of the metrics server",
		Value:  "127.0.0.1",
		EnvVar: prefixEnvVar("METRICS_HOSTNAME"),
	}
	MetricsPortFlag = cli.Uint64Flag{
		Name:   "metrics-port",
		Usage:  "The port of the metrics server",
		Value:  7300,
		EnvVar: prefixEnvVar("METRICS_PORT"),
	}
	HTTP2DisableFlag = cli.BoolFlag{
		Name:   "http2-disable",
		Usage:  "Whether or not to disable HTTP/2 support.",
		EnvVar: prefixEnvVar("HTTP2_DISABLE"),
	}
)

var requiredFlags = []cli.Flag{
	BuildEnvFlag,
	EthNetworkNameFlag,
	L1EthRpcFlag,
	L2EthRpcFlag,
	DepositAddressFlag,
	DepositDeployBlockNumberFlag,
	DisburserAddressFlag,
	MaxL2TxSizeFlag,
	NumDepositConfirmationsFlag,
	FilterQueryMaxBlocksFlag,
	PollIntervalFlag,
	SafeAbortNonceTooLowCountFlag,
	ResubmissionTimeoutFlag,
	PostgresHostFlag,
	PostgresPortFlag,
	PostgresUserFlag,
	PostgresPasswordFlag,
	PostgresDBNameFlag,
}

var optionalFlags = []cli.Flag{
	LogLevelFlag,
	LogTerminalFlag,
	PostgresEnableSSLFlag,
	DisburserPrivateKeyFlag,
	MnemonicFlag,
	DisburserHDPathFlag,
	MetricsServerEnableFlag,
	MetricsHostnameFlag,
	MetricsPortFlag,
	HTTP2DisableFlag,
}

// Flags contains the list of configuration options available to the binary.
var Flags = append(requiredFlags, optionalFlags...)
