package txmgr

import (
	"context"
	"errors"
	"math/big"
	"time"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
	"github.com/ethereum-optimism/optimism/op-signer/client"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli"
)

const (
	// Duplicated L1 RPC flag
	L1RPCFlagName = "l1-eth-rpc"
	// Key Management Flags (also have op-signer client flags)
	MnemonicFlagName   = "mnemonic"
	HDPathFlagName     = "hd-path"
	PrivateKeyFlagName = "private-key"
	// Legacy TxMgr Flags
	NumConfirmationsFlagName          = "num-confirmations"
	SafeAbortNonceTooLowCountFlagName = "safe-abort-nonce-too-low-count"
	ResubmissionTimeoutFlagName       = "resubmission-timeout"
)

var (
	SequencerHDPathFlag = cli.StringFlag{
		Name: "sequencer-hd-path",
		Usage: "DEPRECATED: The HD path used to derive the sequencer wallet from the " +
			"mnemonic. The mnemonic flag must also be set.",
		EnvVar: "OP_BATCHER_SEQUENCER_HD_PATH",
	}
	L2OutputHDPathFlag = cli.StringFlag{
		Name: "l2-output-hd-path",
		Usage: "DEPRECATED:The HD path used to derive the l2output wallet from the " +
			"mnemonic. The mnemonic flag must also be set.",
		EnvVar: "OP_PROPOSER_L2_OUTPUT_HD_PATH",
	}
)

func CLIFlags(envPrefix string) []cli.Flag {
	return append([]cli.Flag{
		cli.StringFlag{
			Name:   MnemonicFlagName,
			Usage:  "The mnemonic used to derive the wallets for either the service",
			EnvVar: opservice.PrefixEnvVar(envPrefix, "MNEMONIC"),
		},
		cli.StringFlag{
			Name:   HDPathFlagName,
			Usage:  "The HD path used to derive the sequencer wallet from the mnemonic. The mnemonic flag must also be set.",
			EnvVar: opservice.PrefixEnvVar(envPrefix, "HD_PATH"),
		},
		SequencerHDPathFlag,
		L2OutputHDPathFlag,
		cli.StringFlag{
			Name:   "private-key",
			Usage:  "The private key to use with the service. Must not be used with mnemonic.",
			EnvVar: opservice.PrefixEnvVar(envPrefix, "PRIVATE_KEY"),
		},
		cli.Uint64Flag{
			Name:   NumConfirmationsFlagName,
			Usage:  "Number of confirmations which we will wait after sending a transaction",
			Value:  10,
			EnvVar: opservice.PrefixEnvVar(envPrefix, "NUM_CONFIRMATIONS"),
		},
		cli.Uint64Flag{
			Name:   "safe-abort-nonce-too-low-count",
			Usage:  "Number of ErrNonceTooLow observations required to give up on a tx at a particular nonce without receiving confirmation",
			Value:  3,
			EnvVar: opservice.PrefixEnvVar(envPrefix, "SAFE_ABORT_NONCE_TOO_LOW_COUNT"),
		},
		cli.DurationFlag{
			Name:   "resubmission-timeout",
			Usage:  "Duration we will wait before resubmitting a transaction to L1",
			Value:  30 * time.Second,
			EnvVar: opservice.PrefixEnvVar(envPrefix, "RESUBMISSION_TIMEOUT"),
		},
	}, client.CLIFlags(envPrefix)...)
}

type CLIConfig struct {
	L1RPCURL                  string
	Mnemonic                  string
	HDPath                    string
	SequencerHDPath           string
	L2OutputHDPath            string
	PrivateKey                string
	SignerCLIConfig           client.CLIConfig
	NumConfirmations          uint64
	SafeAbortNonceTooLowCount uint64
	ResubmissionTimeout       time.Duration
	ReceiptQueryInterval      time.Duration
}

func (m CLIConfig) Check() error {
	if m.L1RPCURL == "" {
		return errors.New("must provide a L1 RPC url")
	}
	if m.NumConfirmations == 0 {
		return errors.New("num confirmations must not be 0")
	}
	if err := m.SignerCLIConfig.Check(); err != nil {
		return err
	}
	return nil
}

func ReadCLIConfig(ctx *cli.Context) CLIConfig {
	return CLIConfig{
		L1RPCURL:                  ctx.GlobalString(L1RPCFlagName),
		Mnemonic:                  ctx.GlobalString(MnemonicFlagName),
		HDPath:                    ctx.GlobalString(HDPathFlagName),
		SequencerHDPath:           ctx.GlobalString(SequencerHDPathFlag.Name),
		L2OutputHDPath:            ctx.GlobalString(L2OutputHDPathFlag.Name),
		PrivateKey:                ctx.GlobalString(PrivateKeyFlagName),
		SignerCLIConfig:           client.ReadCLIConfig(ctx),
		NumConfirmations:          ctx.GlobalUint64(NumConfirmationsFlagName),
		SafeAbortNonceTooLowCount: ctx.GlobalUint64(SafeAbortNonceTooLowCountFlagName),
		ResubmissionTimeout:       ctx.GlobalDuration(ResubmissionTimeoutFlagName),
	}
}

func NewConfig(cfg CLIConfig, l log.Logger) (Config, error) {
	if err := cfg.Check(); err != nil {
		return Config{}, err
	}

	networkTimeout := 2 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), networkTimeout)
	defer cancel()
	l1, err := ethclient.DialContext(ctx, cfg.L1RPCURL)
	if err != nil {
		return Config{}, err
	}

	ctx, cancel = context.WithTimeout(context.Background(), networkTimeout)
	defer cancel()
	chainID, err := l1.ChainID(ctx)
	if err != nil {
		return Config{}, err
	}
	hdPath := cfg.HDPath
	if hdPath == "" && cfg.SequencerHDPath != "" {
		hdPath = cfg.SequencerHDPath
	} else if hdPath == "" && cfg.L2OutputHDPath != "" {
		hdPath = cfg.L2OutputHDPath
	}

	signerFactory, from, err := opcrypto.SignerFactoryFromConfig(l, cfg.PrivateKey, cfg.Mnemonic, hdPath, cfg.SignerCLIConfig)
	if err != nil {
		return Config{}, err
	}

	receiptQueryInterval := 30 * time.Second
	if cfg.ReceiptQueryInterval != 0 {
		receiptQueryInterval = cfg.ReceiptQueryInterval
	}

	return Config{
		Backend:                   l1,
		ResubmissionTimeout:       cfg.ResubmissionTimeout,
		ChainID:                   chainID,
		NetworkTimeout:            networkTimeout,
		ReceiptQueryInterval:      receiptQueryInterval,
		NumConfirmations:          cfg.NumConfirmations,
		SafeAbortNonceTooLowCount: cfg.SafeAbortNonceTooLowCount,
		Signer:                    signerFactory(chainID),
		From:                      from,
	}, nil
}

// Config houses parameters for altering the behavior of a SimpleTxManager.
type Config struct {
	Backend ETHBackend
	// ResubmissionTimeout is the interval at which, if no previously
	// published transaction has been mined, the new tx with a bumped gas
	// price will be published. Only one publication at MaxGasPrice will be
	// attempted.
	ResubmissionTimeout time.Duration

	// ChainID is the chain ID of the L1 chain.
	ChainID *big.Int

	// NetworkTimeout is the allowed duration for a single network request.
	// This is intended to be used for network requests that can be replayed.
	//
	// If not set, this will default to 2 seconds.
	NetworkTimeout time.Duration

	// RequireQueryInterval is the interval at which the tx manager will
	// query the backend to check for confirmations after a tx at a
	// specific gas price has been published.
	ReceiptQueryInterval time.Duration

	// NumConfirmations specifies how many blocks are need to consider a
	// transaction confirmed.
	NumConfirmations uint64

	// SafeAbortNonceTooLowCount specifies how many ErrNonceTooLow observations
	// are required to give up on a tx at a particular nonce without receiving
	// confirmation.
	SafeAbortNonceTooLowCount uint64

	// Signer is used to sign transactions when the gas price is increased.
	Signer opcrypto.SignerFn
	From   common.Address
}
