package indexer

import (
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli"

	"github.com/ethereum-optimism/optimism/go/indexer/flags"
)

var (
	// ErrSentryDSNNotSet signals that not Data Source Name was provided
	// with which to configure Sentry logging.
	ErrSentryDSNNotSet = errors.New("sentry-dsn must be set if use-sentry " +
		"is true")
)

type Config struct {
	/* Required Params */

	// BuildEnv identifies the environment this binary is intended for, i.e.
	// production, development, etc.
	BuildEnv string

	// EthNetworkName identifies the intended Ethereum network.
	EthNetworkName string

	// L1EthRpc is the HTTP provider URL for L1.
	L1EthRpc string

	// L2EthRpc is the HTTP provider URL for L1.
	L2EthRpc string

	// CTCAddress is the CTC contract address.
	CTCAddress string

	// SCCAddress is the SCC contract address.
	SCCAddress string

	// PollInterval is the delay between querying L2 for more transaction
	// and creating a new batch.
	PollInterval time.Duration

	// NumConfirmations is the number of confirmations which we will wait after
	// appending new batches.
	NumConfirmations uint64

	// Hostname of the database connection.
	DBHost string

	// Port of the database connection.
	DBPort uint64

	// Username of the database connection.
	DBUser string

	// Password of the database connection.
	DBPassword string

	// Database name of the database connection.
	DBName string

	/* Optional Params */

	// LogLevel is the lowest log level that will be output.
	LogLevel string

	// SentryEnable if true, logs any error messages to sentry. SentryDsn
	// must also be set if SentryEnable is true.
	SentryEnable bool

	// SentryDsn is the sentry Data Source Name.
	SentryDsn string

	// SentryTraceRate the frequency with which Sentry should flush buffered
	// events.
	SentryTraceRate time.Duration

	// BlockOffset is the offset between the CTC contract start and the L2 geth
	// blocks.
	BlockOffset uint64
	// MetricsServerEnable if true, will create a metrics client and log to
	// Prometheus.
	MetricsServerEnable bool

	// MetricsHostname is the hostname at which the metrics server is running.
	MetricsHostname string

	// MetricsPort is the port at which the metrics server is running.
	MetricsPort uint64
}

// NewConfig parses the Config from the provided flags or environment variables.
// This method fails if ValidateConfig deems the configuration to be malformed.
func NewConfig(ctx *cli.Context) (Config, error) {
	cfg := Config{
		/* Required Flags */
		BuildEnv:         ctx.GlobalString(flags.BuildEnvFlag.Name),
		EthNetworkName:   ctx.GlobalString(flags.EthNetworkNameFlag.Name),
		L1EthRpc:         ctx.GlobalString(flags.L1EthRpcFlag.Name),
		L2EthRpc:         ctx.GlobalString(flags.L2EthRpcFlag.Name),
		CTCAddress:       ctx.GlobalString(flags.CTCAddressFlag.Name),
		SCCAddress:       ctx.GlobalString(flags.SCCAddressFlag.Name),
		NumConfirmations: ctx.GlobalUint64(flags.NumConfirmationsFlag.Name),
		DBHost:           ctx.GlobalString(flags.DBHost.Name),
		DBPort:           ctx.GlobalUint64(flags.DBHost.Name),
		DBUser:           ctx.GlobalString(flags.DBHost.Name),
		DBPassword:       ctx.GlobalString(flags.DBHost.Name),
		DBName:           ctx.GlobalString(flags.DBHost.Name),
		/* Optional Flags */
		SentryEnable:        ctx.GlobalBool(flags.SentryEnableFlag.Name),
		SentryDsn:           ctx.GlobalString(flags.SentryDsnFlag.Name),
		SentryTraceRate:     ctx.GlobalDuration(flags.SentryTraceRateFlag.Name),
		BlockOffset:         ctx.GlobalUint64(flags.BlockOffsetFlag.Name),
		MetricsServerEnable: ctx.GlobalBool(flags.MetricsServerEnableFlag.Name),
		MetricsHostname:     ctx.GlobalString(flags.MetricsHostnameFlag.Name),
		MetricsPort:         ctx.GlobalUint64(flags.MetricsPortFlag.Name),
	}

	err := ValidateConfig(&cfg)
	if err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// ValidateConfig ensures additional constraints on the parsed configuration to
// ensure that it is well-formed.
func ValidateConfig(cfg *Config) error {
	// Sanity check log level.
	if cfg.LogLevel == "" {
		cfg.LogLevel = "debug"
	}

	_, err := log.LvlFromString(cfg.LogLevel)
	if err != nil {
		return err
	}

	// Ensure the Sentry Data Source Name is set when using Sentry.
	if cfg.SentryEnable && cfg.SentryDsn == "" {
		return ErrSentryDSNNotSet
	}

	return nil
}
