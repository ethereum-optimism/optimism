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
	CTCAddressFlag = cli.StringFlag{
		Name:     "ctc-address",
		Usage:    "Address of the CTC contract",
		Required: true,
		EnvVar:   "CTC_ADDRESS",
	}
	SCCAddressFlag = cli.StringFlag{
		Name:     "scc-address",
		Usage:    "Address of the SCC contract",
		Required: true,
		EnvVar:   "SCC_ADDRESS",
	}
	NumConfirmationsFlag = cli.Uint64Flag{
		Name:     "num-confirmations",
		Usage:    "Number of confirmations to wait before finalizing L1 deposit",
		Required: true,
		EnvVar:   prefixEnvVar("NUM_CONFIRMATIONS"),
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
	BlockOffsetFlag = cli.Uint64Flag{
		Name:   "block-offset",
		Usage:  "The offset between the CTC contract start and the L2 geth blocks",
		Value:  1,
		EnvVar: prefixEnvVar("BLOCK_OFFSET"),
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
	CTCAddressFlag,
	SCCAddressFlag,
	NumConfirmationsFlag,
}

var optionalFlags = []cli.Flag{
	LogLevelFlag,
	SentryEnableFlag,
	SentryDsnFlag,
	SentryTraceRateFlag,
	BlockOffsetFlag,
	MetricsServerEnableFlag,
	MetricsHostnameFlag,
	MetricsPortFlag,
}

// Flags contains the list of configuration options available to the binary.
var Flags = append(requiredFlags, optionalFlags...)
