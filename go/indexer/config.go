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

	// ChainID identifies the chain being indexed.
	ChainID int64

	// L1EthRpc is the HTTP provider URL for L1.
	L1EthRpc string

	// L2EthRpc is the HTTP provider URL for L1.
	L2EthRpc string

	// L1AddressManagerAddress is the address of the address manager for L1.
	L1AddressManagerAddress string

	// L2GenesisBlockHash is the l2 genesis block hash.
	L2GenesisBlockHash string

	// PollInterval is the delay between querying L2 for more transaction
	// and creating a new batch.
	PollInterval time.Duration

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

	// LogTerminal if true, prints to stdout in terminal format, otherwise
	// prints using JSON. If SentryEnable is true this flag is ignored, and logs
	// are printed using JSON.
	LogTerminal bool

	// SentryEnable if true, logs any error messages to sentry. SentryDsn
	// must also be set if SentryEnable is true.
	SentryEnable bool

	// SentryDsn is the sentry Data Source Name.
	SentryDsn string

	// SentryTraceRate the frequency with which Sentry should flush buffered
	// events.
	SentryTraceRate time.Duration

	// StartBlockNumber is the block number to start indexing from.
	StartBlockNumber uint64

	// StartBlockHash is the block hash to start indexing from.
	StartBlockHash string

	// ConfDepth is the number of confirmations after which headers are
	// considered confirmed.
	ConfDepth uint64

	// MaxHeaderBatchSize is the maximum number of headers to request as a
	// batch.
	MaxHeaderBatchSize uint64

	// RESTHostname is the hostname at which the REST server is running.
	RESTHostname string

	// RESTPort is the port at which the REST server is running.
	RESTPort uint64

	// MetricsServerEnable if true, will create a metrics client and log to
	// Prometheus.
	MetricsServerEnable bool

	// MetricsHostname is the hostname at which the metrics server is running.
	MetricsHostname string

	// MetricsPort is the port at which the metrics server is running.
	MetricsPort uint64

	// DisableIndexer enables/disables the indexer.
	DisableIndexer bool
}

// NewConfig parses the Config from the provided flags or environment variables.
// This method fails if ValidateConfig deems the configuration to be malformed.
func NewConfig(ctx *cli.Context) (Config, error) {
	cfg := Config{
		/* Required Flags */
		BuildEnv:                ctx.GlobalString(flags.BuildEnvFlag.Name),
		EthNetworkName:          ctx.GlobalString(flags.EthNetworkNameFlag.Name),
		ChainID:                 ctx.GlobalInt64(flags.ChainIDFlag.Name),
		L1EthRpc:                ctx.GlobalString(flags.L1EthRPCFlag.Name),
		L2EthRpc:                ctx.GlobalString(flags.L2EthRPCFlag.Name),
		L1AddressManagerAddress: ctx.GlobalString(flags.L1AddressManagerAddressFlag.Name),
		L2GenesisBlockHash:      ctx.GlobalString(flags.L2GenesisBlockHashFlag.Name),
		DBHost:                  ctx.GlobalString(flags.DBHostFlag.Name),
		DBPort:                  ctx.GlobalUint64(flags.DBPortFlag.Name),
		DBUser:                  ctx.GlobalString(flags.DBUserFlag.Name),
		DBPassword:              ctx.GlobalString(flags.DBPasswordFlag.Name),
		DBName:                  ctx.GlobalString(flags.DBNameFlag.Name),
		/* Optional Flags */
		DisableIndexer:      ctx.GlobalBool(flags.DisableIndexer.Name),
		LogLevel:            ctx.GlobalString(flags.LogLevelFlag.Name),
		LogTerminal:         ctx.GlobalBool(flags.LogTerminalFlag.Name),
		SentryEnable:        ctx.GlobalBool(flags.SentryEnableFlag.Name),
		SentryDsn:           ctx.GlobalString(flags.SentryDsnFlag.Name),
		SentryTraceRate:     ctx.GlobalDuration(flags.SentryTraceRateFlag.Name),
		StartBlockNumber:    ctx.GlobalUint64(flags.StartBlockNumberFlag.Name),
		StartBlockHash:      ctx.GlobalString(flags.StartBlockHashFlag.Name),
		ConfDepth:           ctx.GlobalUint64(flags.ConfDepthFlag.Name),
		MaxHeaderBatchSize:  ctx.GlobalUint64(flags.MaxHeaderBatchSizeFlag.Name),
		MetricsServerEnable: ctx.GlobalBool(flags.MetricsServerEnableFlag.Name),
		RESTHostname:        ctx.GlobalString(flags.RESTHostnameFlag.Name),
		RESTPort:            ctx.GlobalUint64(flags.RESTPortFlag.Name),
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
