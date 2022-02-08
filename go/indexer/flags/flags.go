package flags

import (
	"time"

	"github.com/urfave/cli"
)

const envVarPrefix = "INDEXER_"

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
	L1StandardBridgeAddressFlag = cli.StringFlag{
		Name:     "l1-standard-bridge-address",
		Usage:    "Address of the L1 Standard Bridge",
		Required: true,
		EnvVar:   "L1_STANDARD_BRIDGE_ADDRESS",
	}
	L2StandardBridgeAddressFlag = cli.StringFlag{
		Name:     "l2-standard-bridge-address",
		Usage:    "Address of the L2 Standard Bridge",
		Required: true,
		EnvVar:   "L2_STANDARD_BRIDGE_ADDRESS",
	}
	L2GenesisBlockHashFlag = cli.StringFlag{
		Name:     "l2-genesis-block-hash",
		Usage:    "Genesis block hash of the L2 chain",
		Required: true,
		EnvVar:   "L2_GENESIS_BLOCK_HASH",
	}
	NumConfirmationsFlag = cli.Uint64Flag{
		Name:     "num-confirmations",
		Usage:    "Number of confirmations to wait before finalizing L1 deposit",
		Required: true,
		EnvVar:   prefixEnvVar("NUM_CONFIRMATIONS"),
	}
	DBHostFlag = cli.StringFlag{
		Name:     "db-host",
		Usage:    "Hostname of the database connection",
		Required: true,
		EnvVar:   prefixEnvVar("DB_HOST"),
	}
	DBPortFlag = cli.Uint64Flag{
		Name:     "db-port",
		Usage:    "Port of the database connection",
		Required: true,
		EnvVar:   prefixEnvVar("DB_PORT"),
	}
	DBUserFlag = cli.StringFlag{
		Name:     "db-user",
		Usage:    "Username of the database connection",
		Required: true,
		EnvVar:   prefixEnvVar("DB_USER"),
	}
	DBPasswordFlag = cli.StringFlag{
		Name:     "db-password",
		Usage:    "Password of the database connection",
		Required: true,
		EnvVar:   prefixEnvVar("DB_PASSWORD"),
	}
	DBNameFlag = cli.StringFlag{
		Name:     "db-name",
		Usage:    "Database name of the database connection",
		Required: true,
		EnvVar:   prefixEnvVar("DB_NAME"),
	}

	/* Optional Flags */

	LogLevelFlag = cli.StringFlag{
		Name:   "log-level",
		Usage:  "The lowest log level that will be output",
		Value:  "info",
		EnvVar: prefixEnvVar("LOG_LEVEL"),
	}
	SentryEnableFlag = cli.BoolFlag{
		Name:   "sentry-enable",
		Usage:  "Whether or not to enable Sentry. If true, sentry-dsn must also be set",
		EnvVar: prefixEnvVar("SENTRY_ENABLE"),
	}
	SentryDsnFlag = cli.StringFlag{
		Name:   "sentry-dsn",
		Usage:  "Sentry data source name",
		EnvVar: prefixEnvVar("SENTRY_DSN"),
	}
	SentryTraceRateFlag = cli.DurationFlag{
		Name:   "sentry-trace-rate",
		Usage:  "Sentry trace rate",
		Value:  50 * time.Millisecond,
		EnvVar: prefixEnvVar("SENTRY_TRACE_RATE"),
	}
	StartBlockNumberFlag = cli.Uint64Flag{
		Name:   "start-block-number",
		Usage:  "The block number to start indexing from. Must be use together with start block hash",
		Value:  0,
		EnvVar: prefixEnvVar("START_BLOCK_NUMBER"),
	}
	StartBlockHashFlag = cli.StringFlag{
		Name:   "start-block-hash",
		Usage:  "The block hash to start indexing from. Must be use together with start block number",
		Value:  "0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3",
		EnvVar: prefixEnvVar("START_BLOCK_HASH"),
	}
	ConfDepthFlag = cli.Uint64Flag{
		Name:   "conf-depth",
		Usage:  "The number of confirmations after which headers are considered confirmed",
		Value:  20,
		EnvVar: prefixEnvVar("CONF_DEPTH"),
	}
	MaxHeaderBatchSizeFlag = cli.Uint64Flag{
		Name:   "max-header-batch-size",
		Usage:  "The maximum number of headers to request as a batch",
		Value:  100,
		EnvVar: prefixEnvVar("MAX_HEADER_BATCH_SIZE"),
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
)

var requiredFlags = []cli.Flag{
	BuildEnvFlag,
	EthNetworkNameFlag,
	L1EthRpcFlag,
	L2EthRpcFlag,
	L1StandardBridgeAddressFlag,
	L2StandardBridgeAddressFlag,
	L2GenesisBlockHashFlag,
	NumConfirmationsFlag,
	DBHostFlag,
	DBPortFlag,
	DBUserFlag,
	DBPasswordFlag,
	DBNameFlag,
}

var optionalFlags = []cli.Flag{
	LogLevelFlag,
	SentryEnableFlag,
	SentryDsnFlag,
	SentryTraceRateFlag,
	ConfDepthFlag,
	MaxHeaderBatchSizeFlag,
	StartBlockNumberFlag,
	StartBlockHashFlag,
	MetricsServerEnableFlag,
	MetricsHostnameFlag,
	MetricsPortFlag,
}

// Flags contains the list of configuration options available to the binary.
var Flags = append(requiredFlags, optionalFlags...)
