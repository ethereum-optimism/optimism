package txmgr

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
	"github.com/ethereum-optimism/optimism/op-signer/client"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

const (
	// Duplicated L1 RPC flag
	L1RPCFlagName = "l1-eth-rpc"
	// Key Management Flags (also have op-signer client flags)
	MnemonicFlagName   = "mnemonic"
	HDPathFlagName     = "hd-path"
	PrivateKeyFlagName = "private-key"
	// TxMgr Flags (new + legacy + some shared flags)
	NumConfirmationsFlagName          = "num-confirmations"
	SafeAbortNonceTooLowCountFlagName = "safe-abort-nonce-too-low-count"
	ResubmissionTimeoutFlagName       = "resubmission-timeout"
	NetworkTimeoutFlagName            = "network-timeout"
	TxSendTimeoutFlagName             = "txmgr.send-timeout"
	TxNotInMempoolTimeoutFlagName     = "txmgr.not-in-mempool-timeout"
	ReceiptQueryIntervalFlagName      = "txmgr.receipt-query-interval"
)

var (
	SequencerHDPathFlag = &cli.StringFlag{
		Name: "sequencer-hd-path",
		Usage: "DEPRECATED: The HD path used to derive the sequencer wallet from the " +
			"mnemonic. The mnemonic flag must also be set.",
		EnvVars: []string{"OP_BATCHER_SEQUENCER_HD_PATH"},
	}
	L2OutputHDPathFlag = &cli.StringFlag{
		Name: "l2-output-hd-path",
		Usage: "DEPRECATED:The HD path used to derive the l2output wallet from the " +
			"mnemonic. The mnemonic flag must also be set.",
		EnvVars: []string{"OP_PROPOSER_L2_OUTPUT_HD_PATH"},
	}
)

var (
	defaultNumConfirmations          = uint64(10)
	defaultSafeAbortNonceTooLowCount = uint64(3)
	defaultResubmissionTimeout       = 48 * time.Second
	defaultNetworkTimeout            = 10 * time.Second
	defaultTxSendTimeout             = 0 * time.Second
	defaultTxNotInMempoolTimeout     = 2 * time.Minute
	defaultReceiptQueryInterval      = 12 * time.Second
)

func CLIFlags(envPrefix string) []cli.Flag {
	prefixEnvVars := func(name string) []string {
		return opservice.PrefixEnvVar(envPrefix, name)
	}
	return append([]cli.Flag{
		&cli.StringFlag{
			Name:    MnemonicFlagName,
			Usage:   "The mnemonic used to derive the wallets for either the service",
			EnvVars: prefixEnvVars("MNEMONIC"),
		},
		&cli.StringFlag{
			Name:    HDPathFlagName,
			Usage:   "The HD path used to derive the sequencer wallet from the mnemonic. The mnemonic flag must also be set.",
			EnvVars: prefixEnvVars("HD_PATH"),
		},
		&cli.StringFlag{
			Name:    PrivateKeyFlagName,
			Usage:   "The private key to use with the service. Must not be used with mnemonic.",
			EnvVars: prefixEnvVars("PRIVATE_KEY"),
		},
		&cli.Uint64Flag{
			Name:    NumConfirmationsFlagName,
			Usage:   "Number of confirmations which we will wait after sending a transaction",
			Value:   defaultNumConfirmations,
			EnvVars: prefixEnvVars("NUM_CONFIRMATIONS"),
		},
		&cli.Uint64Flag{
			Name:    SafeAbortNonceTooLowCountFlagName,
			Usage:   "Number of ErrNonceTooLow observations required to give up on a tx at a particular nonce without receiving confirmation",
			Value:   defaultSafeAbortNonceTooLowCount,
			EnvVars: prefixEnvVars("SAFE_ABORT_NONCE_TOO_LOW_COUNT"),
		},
		&cli.DurationFlag{
			Name:    ResubmissionTimeoutFlagName,
			Usage:   "Duration we will wait before resubmitting a transaction to L1",
			Value:   defaultResubmissionTimeout,
			EnvVars: prefixEnvVars("RESUBMISSION_TIMEOUT"),
		},
		&cli.DurationFlag{
			Name:    NetworkTimeoutFlagName,
			Usage:   "Timeout for all network operations",
			Value:   defaultNetworkTimeout,
			EnvVars: prefixEnvVars("NETWORK_TIMEOUT"),
		},
		&cli.DurationFlag{
			Name:    TxSendTimeoutFlagName,
			Usage:   "Timeout for sending transactions. If 0 it is disabled.",
			Value:   defaultTxSendTimeout,
			EnvVars: prefixEnvVars("TXMGR_TX_SEND_TIMEOUT"),
		},
		&cli.DurationFlag{
			Name:    TxNotInMempoolTimeoutFlagName,
			Usage:   "Timeout for aborting a tx send if the tx does not make it to the mempool.",
			Value:   defaultTxNotInMempoolTimeout,
			EnvVars: prefixEnvVars("TXMGR_TX_NOT_IN_MEMPOOL_TIMEOUT"),
		},
		&cli.DurationFlag{
			Name:    ReceiptQueryIntervalFlagName,
			Usage:   "Frequency to poll for receipts",
			Value:   defaultReceiptQueryInterval,
			EnvVars: prefixEnvVars("TXMGR_RECEIPT_QUERY_INTERVAL"),
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
	NetworkTimeout            time.Duration
	TxSendTimeout             time.Duration
	TxNotInMempoolTimeout     time.Duration
}

func NewCLIConfig(l1RPCURL string) CLIConfig {
	return CLIConfig{
		L1RPCURL:                  l1RPCURL,
		NumConfirmations:          defaultNumConfirmations,
		SafeAbortNonceTooLowCount: defaultSafeAbortNonceTooLowCount,
		ResubmissionTimeout:       defaultResubmissionTimeout,
		NetworkTimeout:            defaultNetworkTimeout,
		TxSendTimeout:             defaultTxSendTimeout,
		TxNotInMempoolTimeout:     defaultTxNotInMempoolTimeout,
		ReceiptQueryInterval:      defaultReceiptQueryInterval,
		SignerCLIConfig:           client.NewCLIConfig(),
	}
}

func (m CLIConfig) Check() error {
	if m.L1RPCURL == "" {
		return errors.New("must provide a L1 RPC url")
	}
	if m.NumConfirmations == 0 {
		return errors.New("NumConfirmations must not be 0")
	}
	if m.NetworkTimeout == 0 {
		return errors.New("must provide NetworkTimeout")
	}
	if m.ResubmissionTimeout == 0 {
		return errors.New("must provide ResubmissionTimeout")
	}
	if m.ReceiptQueryInterval == 0 {
		return errors.New("must provide ReceiptQueryInterval")
	}
	if m.TxNotInMempoolTimeout == 0 {
		return errors.New("must provide TxNotInMempoolTimeout")
	}
	if m.SafeAbortNonceTooLowCount == 0 {
		return errors.New("SafeAbortNonceTooLowCount must not be 0")
	}
	if err := m.SignerCLIConfig.Check(); err != nil {
		return err
	}
	return nil
}

func ReadCLIConfig(ctx *cli.Context) CLIConfig {
	return CLIConfig{
		L1RPCURL:                  ctx.String(L1RPCFlagName),
		Mnemonic:                  ctx.String(MnemonicFlagName),
		HDPath:                    ctx.String(HDPathFlagName),
		SequencerHDPath:           ctx.String(SequencerHDPathFlag.Name),
		L2OutputHDPath:            ctx.String(L2OutputHDPathFlag.Name),
		PrivateKey:                ctx.String(PrivateKeyFlagName),
		SignerCLIConfig:           client.ReadCLIConfig(ctx),
		NumConfirmations:          ctx.Uint64(NumConfirmationsFlagName),
		SafeAbortNonceTooLowCount: ctx.Uint64(SafeAbortNonceTooLowCountFlagName),
		ResubmissionTimeout:       ctx.Duration(ResubmissionTimeoutFlagName),
		ReceiptQueryInterval:      ctx.Duration(ReceiptQueryIntervalFlagName),
		NetworkTimeout:            ctx.Duration(NetworkTimeoutFlagName),
		TxSendTimeout:             ctx.Duration(TxSendTimeoutFlagName),
		TxNotInMempoolTimeout:     ctx.Duration(TxNotInMempoolTimeoutFlagName),
	}
}

func NewConfig(cfg CLIConfig, l log.Logger) (Config, error) {
	if err := cfg.Check(); err != nil {
		return Config{}, fmt.Errorf("invalid config: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.NetworkTimeout)
	defer cancel()
	l1, err := ethclient.DialContext(ctx, cfg.L1RPCURL)
	if err != nil {
		return Config{}, fmt.Errorf("could not dial eth client: %w", err)
	}

	ctx, cancel = context.WithTimeout(context.Background(), cfg.NetworkTimeout)
	defer cancel()
	chainID, err := l1.ChainID(ctx)
	if err != nil {
		return Config{}, fmt.Errorf("could not dial fetch L1 chain ID: %w", err)
	}

	// Allow backwards compatible ways of specifying the HD path
	hdPath := cfg.HDPath
	if hdPath == "" && cfg.SequencerHDPath != "" {
		hdPath = cfg.SequencerHDPath
	} else if hdPath == "" && cfg.L2OutputHDPath != "" {
		hdPath = cfg.L2OutputHDPath
	}

	signerFactory, from, err := opcrypto.SignerFactoryFromConfig(l, cfg.PrivateKey, cfg.Mnemonic, hdPath, cfg.SignerCLIConfig)
	if err != nil {
		return Config{}, fmt.Errorf("could not init signer: %w", err)
	}

	return Config{
		Backend:                   l1,
		ResubmissionTimeout:       cfg.ResubmissionTimeout,
		ChainID:                   chainID,
		TxSendTimeout:             cfg.TxSendTimeout,
		TxNotInMempoolTimeout:     cfg.TxNotInMempoolTimeout,
		NetworkTimeout:            cfg.NetworkTimeout,
		ReceiptQueryInterval:      cfg.ReceiptQueryInterval,
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

	// TxSendTimeout is how long to wait for sending a transaction.
	// By default it is unbounded. If set, this is recommended to be at least 20 minutes.
	TxSendTimeout time.Duration

	// TxNotInMempoolTimeout is how long to wait before aborting a transaction send if the transaction does not
	// make it to the mempool. If the tx is in the mempool, TxSendTimeout is used instead.
	TxNotInMempoolTimeout time.Duration

	// NetworkTimeout is the allowed duration for a single network request.
	// This is intended to be used for network requests that can be replayed.
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
