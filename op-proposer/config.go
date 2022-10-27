package op_proposer

import (
	"time"

	"github.com/urfave/cli"

	"github.com/ethereum-optimism/optimism/op-proposer/flags"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
)

type Config struct {
	/* Required Params */

	// L1EthRpc is the HTTP provider URL for L1.
	L1EthRpc string

	// RollupRpc is the HTTP provider URL for the rollup node.
	RollupRpc string

	// L2OOAddress is the L2OutputOracle contract address.
	L2OOAddress string

	// PollInterval is the delay between querying L2 for more transaction
	// and creating a new batch.
	PollInterval time.Duration

	// NumConfirmations is the number of confirmations which we will wait after
	// appending new batches.
	NumConfirmations uint64

	// SafeAbortNonceTooLowCount is the number of ErrNonceTooLowObservations
	// required to give up on a tx at a particular nonce without receiving
	// confirmation.
	SafeAbortNonceTooLowCount uint64

	// ResubmissionTimeout is time we will wait before resubmitting a
	// transaction.
	ResubmissionTimeout time.Duration

	// Mnemonic is the HD seed used to derive the wallet private keys for both
	// the sequence and proposer. Must be used in conjunction with
	// SequencerHDPath and ProposerHDPath.
	Mnemonic string

	// L2OutputHDPath is the derivation path used to obtain the private key for
	// the l2output transactions.
	L2OutputHDPath string

	// PrivateKey is the private key used for l2output transactions.
	PrivateKey string

	RPCConfig oprpc.CLIConfig

	/* Optional Params */

	// AllowNonFinalized can be set to true to propose outputs
	// for L2 blocks derived from non-finalized L1 data.
	AllowNonFinalized bool

	LogConfig oplog.CLIConfig

	MetricsConfig opmetrics.CLIConfig

	PprofConfig oppprof.CLIConfig
}

func (c Config) Check() error {
	if err := c.RPCConfig.Check(); err != nil {
		return err
	}
	if err := c.LogConfig.Check(); err != nil {
		return err
	}
	if err := c.MetricsConfig.Check(); err != nil {
		return err
	}
	if err := c.PprofConfig.Check(); err != nil {
		return err
	}
	return nil
}

// NewConfig parses the Config from the provided flags or environment variables.
func NewConfig(ctx *cli.Context) Config {
	return Config{
		/* Required Flags */
		L1EthRpc:                  ctx.GlobalString(flags.L1EthRpcFlag.Name),
		RollupRpc:                 ctx.GlobalString(flags.RollupRpcFlag.Name),
		L2OOAddress:               ctx.GlobalString(flags.L2OOAddressFlag.Name),
		PollInterval:              ctx.GlobalDuration(flags.PollIntervalFlag.Name),
		NumConfirmations:          ctx.GlobalUint64(flags.NumConfirmationsFlag.Name),
		SafeAbortNonceTooLowCount: ctx.GlobalUint64(flags.SafeAbortNonceTooLowCountFlag.Name),
		ResubmissionTimeout:       ctx.GlobalDuration(flags.ResubmissionTimeoutFlag.Name),
		Mnemonic:                  ctx.GlobalString(flags.MnemonicFlag.Name),
		L2OutputHDPath:            ctx.GlobalString(flags.L2OutputHDPathFlag.Name),
		PrivateKey:                ctx.GlobalString(flags.PrivateKeyFlag.Name),
		AllowNonFinalized:         ctx.GlobalBool(flags.AllowNonFinalizedFlag.Name),
		RPCConfig:                 oprpc.ReadCLIConfig(ctx),
		LogConfig:                 oplog.ReadCLIConfig(ctx),
		MetricsConfig:             opmetrics.ReadCLIConfig(ctx),
		PprofConfig:               oppprof.ReadCLIConfig(ctx),
	}
}
