package teleportr

import (
	"time"

	"github.com/ethereum-optimism/optimism/go/teleportr/flags"
	"github.com/urfave/cli"
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

	// DepositAddress is the TeleportrDeposit contract adddress.
	DepositAddress string

	// DepositDeployBlockNumber is the deployment block number of the
	// TeleportrDeposit contract.
	DepositDeployBlockNumber uint64

	// FilterQueryMaxBlocks is the maximum range of a filter query in blocks.
	FilterQueryMaxBlocks uint64

	// DisburserAddress is the TeleportrDisburser contract address.
	DisburserAddress string

	// MaxL2TxSize is the maximum size in bytes of any L2 transactions generated
	// for teleportr disbursements.
	MaxL2TxSize uint64

	// NumDepositConfirmations is the number of confirmations required before a
	// deposit is considered confirmed.
	NumDepositConfirmations uint64

	// PollInterval is the delay between querying L2 for more transaction
	// and creating a new batch.
	PollInterval time.Duration

	// SafeAbortNonceTooLowCount is the number of ErrNonceTooLowObservations
	// required to give up on a tx at a particular nonce without receiving
	// confirmation.
	SafeAbortNonceTooLowCount uint64

	// ResubmissionTimeout is time we will wait before resubmitting a
	// transaction.
	ResubmissionTimeout time.Duration

	// PostgresHost is the host of the teleportr postgres instance.
	PostgresHost string

	// PostgresPort is the port of the teleportr postgres instance.
	PostgresPort uint16

	// PostgresUser is the username for the teleportr postgres instance.
	PostgresUser string

	// PostgresPassword is the password for the teleportr postgres instance.
	PostgresPassword string

	// PostgresDBName is the database name of the teleportr postgres instance.
	PostgresDBName string

	// PostgresEnableSSL determines whether or not to enable SSL on connections
	// to the teleportr postgres instance.
	PostgresEnableSSL bool

	/* Optional Params */

	// LogLevel is the lowest log level that will be output.
	LogLevel string

	// LogTerminal if true, prints to stdout in terminal format, otherwise
	// prints using JSON. If SentryEnable is true this flag is ignored, and logs
	// are printed using JSON.
	LogTerminal bool

	// DisburserPrivKey the private key of the wallet used to submit
	// transactions to the TeleportrDisburser contract.
	DisburserPrivKey string

	// Mnemonic is the HD seed used to derive the wallet private key for
	// submitting to the TeleportrDisburser. Must be used in conjunction with
	// DisburserHDPath.
	Mnemonic string

	// DisburserHDPath is the derivation path used to obtain the private key for
	// the disburser transactions.
	DisburserHDPath string

	// MetricsServerEnable if true, will create a metrics client and log to
	// Prometheus.
	MetricsServerEnable bool

	// MetricsHostname is the hostname at which the metrics server is running.
	MetricsHostname string

	// MetricsPort is the port at which the metrics server is running.
	MetricsPort uint64

	// DisableHTTP2 disables HTTP2 support.
	DisableHTTP2 bool
}

func NewConfig(ctx *cli.Context) (Config, error) {
	return Config{
		/* Required Flags */
		BuildEnv:                  ctx.GlobalString(flags.BuildEnvFlag.Name),
		EthNetworkName:            ctx.GlobalString(flags.EthNetworkNameFlag.Name),
		L1EthRpc:                  ctx.GlobalString(flags.L1EthRpcFlag.Name),
		L2EthRpc:                  ctx.GlobalString(flags.L2EthRpcFlag.Name),
		DepositAddress:            ctx.GlobalString(flags.DepositAddressFlag.Name),
		DepositDeployBlockNumber:  ctx.GlobalUint64(flags.DepositDeployBlockNumberFlag.Name),
		DisburserAddress:          ctx.GlobalString(flags.DisburserAddressFlag.Name),
		MaxL2TxSize:               ctx.GlobalUint64(flags.MaxL2TxSizeFlag.Name),
		NumDepositConfirmations:   ctx.GlobalUint64(flags.NumDepositConfirmationsFlag.Name),
		FilterQueryMaxBlocks:      ctx.GlobalUint64(flags.FilterQueryMaxBlocksFlag.Name),
		PollInterval:              ctx.GlobalDuration(flags.PollIntervalFlag.Name),
		SafeAbortNonceTooLowCount: ctx.GlobalUint64(flags.SafeAbortNonceTooLowCountFlag.Name),
		ResubmissionTimeout:       ctx.GlobalDuration(flags.ResubmissionTimeoutFlag.Name),
		PostgresHost:              ctx.GlobalString(flags.PostgresHostFlag.Name),
		PostgresPort:              uint16(ctx.GlobalUint64(flags.PostgresPortFlag.Name)),
		PostgresUser:              ctx.GlobalString(flags.PostgresUserFlag.Name),
		PostgresPassword:          ctx.GlobalString(flags.PostgresPasswordFlag.Name),
		PostgresDBName:            ctx.GlobalString(flags.PostgresDBNameFlag.Name),
		PostgresEnableSSL:         ctx.GlobalBool(flags.PostgresEnableSSLFlag.Name),
		/* Optional flags */
		LogLevel:            ctx.GlobalString(flags.LogLevelFlag.Name),
		LogTerminal:         ctx.GlobalBool(flags.LogTerminalFlag.Name),
		DisburserPrivKey:    ctx.GlobalString(flags.DisburserPrivateKeyFlag.Name),
		Mnemonic:            ctx.GlobalString(flags.MnemonicFlag.Name),
		DisburserHDPath:     ctx.GlobalString(flags.DisburserHDPathFlag.Name),
		MetricsServerEnable: ctx.GlobalBool(flags.MetricsServerEnableFlag.Name),
		MetricsHostname:     ctx.GlobalString(flags.MetricsHostnameFlag.Name),
		MetricsPort:         ctx.GlobalUint64(flags.MetricsPortFlag.Name),
		DisableHTTP2:        ctx.GlobalBool(flags.HTTP2DisableFlag.Name),
	}, nil
}
