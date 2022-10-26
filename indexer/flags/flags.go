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
		EnvVar:   prefixEnvVar("BUILD_ENV"),
	}
	ChainIDFlag = cli.StringFlag{
		Name:     "chain-id",
		Usage:    "Ethereum chain ID",
		Required: true,
		EnvVar:   prefixEnvVar("CHAIN_ID"),
	}
	L1EthRPCFlag = cli.StringFlag{
		Name:     "l1-eth-rpc",
		Usage:    "HTTP provider URL for L1",
		Required: true,
		EnvVar:   prefixEnvVar("L1_ETH_RPC"),
	}
	L2EthRPCFlag = cli.StringFlag{
		Name:     "l2-eth-rpc",
		Usage:    "HTTP provider URL for L2",
		Required: true,
		EnvVar:   prefixEnvVar("L2_ETH_RPC"),
	}
	L1AddressManagerAddressFlag = cli.StringFlag{
		Name:     "l1-address-manager-address",
		Usage:    "Address of the L1 address manager",
		Required: true,
		EnvVar:   prefixEnvVar("L1_ADDRESS_MANAGER_ADDRESS"),
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

	/* Bedrock Flags */
	BedrockFlag = cli.BoolFlag{
		Name:   "bedrock",
		Usage:  "Whether or not this indexer should operate in Bedrock mode",
		EnvVar: prefixEnvVar("BEDROCK"),
	}
	BedrockL1StandardBridgeAddress = cli.BoolFlag{
		Name:   "bedrock.l1-standard-bridge-address",
		Usage:  "Address of the L1 standard bridge",
		EnvVar: prefixEnvVar("BEDROCK_L1_STANDARD_BRIDGE"),
	}
	BedrockOptimismPortalAddress = cli.BoolFlag{
		Name:   "bedrock.portal-address",
		Usage:  "Address of the portal",
		EnvVar: prefixEnvVar("BEDROCK_OPTIMISM_PORTAL"),
	}

	/* Optional Flags */

	DisableIndexer = cli.BoolFlag{
		Name:     "disable-indexer",
		Usage:    "Whether or not to enable the indexer on this instance",
		Required: false,
		EnvVar:   prefixEnvVar("DISABLE_INDEXER"),
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
	L1StartBlockNumberFlag = cli.Uint64Flag{
		Name:   "start-block-number",
		Usage:  "The block number to start indexing from. Must be use together with start block hash",
		Value:  0,
		EnvVar: prefixEnvVar("START_BLOCK_NUMBER"),
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
		Value:  2000,
		EnvVar: prefixEnvVar("MAX_HEADER_BATCH_SIZE"),
	}
	RESTHostnameFlag = cli.StringFlag{
		Name:   "rest-hostname",
		Usage:  "The hostname of the REST server",
		Value:  "127.0.0.1",
		EnvVar: prefixEnvVar("REST_HOSTNAME"),
	}
	RESTPortFlag = cli.Uint64Flag{
		Name:   "rest-port",
		Usage:  "The port of the REST server",
		Value:  8080,
		EnvVar: prefixEnvVar("REST_PORT"),
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
	ChainIDFlag,
	L1EthRPCFlag,
	L2EthRPCFlag,
	L1AddressManagerAddressFlag,
	DBHostFlag,
	DBPortFlag,
	DBUserFlag,
	DBPasswordFlag,
	DBNameFlag,
}

var optionalFlags = []cli.Flag{
	BedrockFlag,
	BedrockL1StandardBridgeAddress,
	BedrockOptimismPortalAddress,
	DisableIndexer,
	LogLevelFlag,
	LogTerminalFlag,
	SentryEnableFlag,
	SentryDsnFlag,
	SentryTraceRateFlag,
	ConfDepthFlag,
	MaxHeaderBatchSizeFlag,
	L1StartBlockNumberFlag,
	RESTHostnameFlag,
	RESTPortFlag,
	MetricsServerEnableFlag,
	MetricsHostnameFlag,
	MetricsPortFlag,
}

// Flags contains the list of configuration options available to the binary.
var Flags = append(requiredFlags, optionalFlags...)
