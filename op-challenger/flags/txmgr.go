package flags

import (
	"time"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	txmgr "github.com/ethereum-optimism/optimism/op-service/txmgr"
	client "github.com/ethereum-optimism/optimism/op-signer/client"
	cli "github.com/urfave/cli"
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

func TxManagerCLIFlags(envPrefix string) []cli.Flag {
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
		// cli.StringFlag{
		// 	Name:   "private-key",
		// 	Usage:  "The private key to use with the service. Must not be used with mnemonic.",
		// 	EnvVar: opservice.PrefixEnvVar(envPrefix, "PRIVATE_KEY"),
		// },
		cli.Uint64Flag{
			Name:   NumConfirmationsFlagName,
			Usage:  "Number of confirmations which we will wait after sending a transaction",
			Value:  10,
			EnvVar: opservice.PrefixEnvVar(envPrefix, "NUM_CONFIRMATIONS"),
		},
		cli.Uint64Flag{
			Name:   SafeAbortNonceTooLowCountFlagName,
			Usage:  "Number of ErrNonceTooLow observations required to give up on a tx at a particular nonce without receiving confirmation",
			Value:  3,
			EnvVar: opservice.PrefixEnvVar(envPrefix, "SAFE_ABORT_NONCE_TOO_LOW_COUNT"),
		},
		cli.DurationFlag{
			Name:   ResubmissionTimeoutFlagName,
			Usage:  "Duration we will wait before resubmitting a transaction to L1",
			Value:  48 * time.Second,
			EnvVar: opservice.PrefixEnvVar(envPrefix, "RESUBMISSION_TIMEOUT"),
		},
		cli.DurationFlag{
			Name:   NetworkTimeoutFlagName,
			Usage:  "Timeout for all network operations",
			Value:  2 * time.Second,
			EnvVar: opservice.PrefixEnvVar(envPrefix, "NETWORK_TIMEOUT"),
		},
		cli.DurationFlag{
			Name:   TxSendTimeoutFlagName,
			Usage:  "Timeout for sending transactions. If 0 it is disabled.",
			Value:  0,
			EnvVar: opservice.PrefixEnvVar(envPrefix, "TXMGR_TX_SEND_TIMEOUT"),
		},
		cli.DurationFlag{
			Name:   TxNotInMempoolTimeoutFlagName,
			Usage:  "Timeout for aborting a tx send if the tx does not make it to the mempool.",
			Value:  2 * time.Minute,
			EnvVar: opservice.PrefixEnvVar(envPrefix, "TXMGR_TX_NOT_IN_MEMPOOL_TIMEOUT"),
		},
		cli.DurationFlag{
			Name:   ReceiptQueryIntervalFlagName,
			Usage:  "Frequency to poll for receipts",
			Value:  12 * time.Second,
			EnvVar: opservice.PrefixEnvVar(envPrefix, "TXMGR_RECEIPT_QUERY_INTERVAL"),
		},
	}, client.CLIFlags(envPrefix)...)
}

// type TxManagerCLIConfig struct {
// 	L1RPCURL                  string
// 	Mnemonic                  string
// 	HDPath                    string
// 	SequencerHDPath           string
// 	L2OutputHDPath            string
// 	PrivateKey                string
// 	SignerCLIConfig           client.CLIConfig
// 	NumConfirmations          uint64
// 	SafeAbortNonceTooLowCount uint64
// 	ResubmissionTimeout       time.Duration
// 	ReceiptQueryInterval      time.Duration
// 	NetworkTimeout            time.Duration
// 	TxSendTimeout             time.Duration
// 	TxNotInMempoolTimeout     time.Duration
// }

// func (m TxManagerCLIConfig) Check() error {
// 	if m.L1RPCURL == "" {
// 		return errors.New("must provide a L1 RPC url")
// 	}
// 	if m.NumConfirmations == 0 {
// 		return errors.New("NumConfirmations must not be 0")
// 	}
// 	if m.NetworkTimeout == 0 {
// 		return errors.New("must provide NetworkTimeout")
// 	}
// 	if m.ResubmissionTimeout == 0 {
// 		return errors.New("must provide ResubmissionTimeout")
// 	}
// 	if m.ReceiptQueryInterval == 0 {
// 		return errors.New("must provide ReceiptQueryInterval")
// 	}
// 	if m.TxNotInMempoolTimeout == 0 {
// 		return errors.New("must provide TxNotInMempoolTimeout")
// 	}
// 	if m.SafeAbortNonceTooLowCount == 0 {
// 		return errors.New("SafeAbortNonceTooLowCount must not be 0")
// 	}
// 	if err := m.SignerCLIConfig.Check(); err != nil {
// 		return err
// 	}
// 	return nil
// }

func ReadTxManagerCLIConfig(ctx *cli.Context) txmgr.CLIConfig {
	return txmgr.CLIConfig{
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
		ReceiptQueryInterval:      ctx.GlobalDuration(ReceiptQueryIntervalFlagName),
		NetworkTimeout:            ctx.GlobalDuration(NetworkTimeoutFlagName),
		TxSendTimeout:             ctx.GlobalDuration(TxSendTimeoutFlagName),
		TxNotInMempoolTimeout:     ctx.GlobalDuration(TxNotInMempoolTimeoutFlagName),
	}
}

// func NewTxManagerConfig(cfg TxManagerCLIConfig, l log.Logger) (txmgr.Config, error) {
// 	if err := cfg.Check(); err != nil {
// 		return txmgr.Config{}, fmt.Errorf("invalid config: %w", err)
// 	}

// 	ctx, cancel := context.WithTimeout(context.Background(), cfg.NetworkTimeout)
// 	defer cancel()
// 	l1, err := ethclient.DialContext(ctx, cfg.L1RPCURL)
// 	if err != nil {
// 		return txmgr.Config{}, fmt.Errorf("could not dial eth client: %w", err)
// 	}

// 	ctx, cancel = context.WithTimeout(context.Background(), cfg.NetworkTimeout)
// 	defer cancel()
// 	chainID, err := l1.ChainID(ctx)
// 	if err != nil {
// 		return txmgr.Config{}, fmt.Errorf("could not dial fetch L1 chain ID: %w", err)
// 	}

// 	// Allow backwards compatible ways of specifying the HD path
// 	hdPath := cfg.HDPath
// 	if hdPath == "" && cfg.SequencerHDPath != "" {
// 		hdPath = cfg.SequencerHDPath
// 	} else if hdPath == "" && cfg.L2OutputHDPath != "" {
// 		hdPath = cfg.L2OutputHDPath
// 	}

// 	signerFactory, from, err := opcrypto.SignerFactoryFromConfig(l, cfg.PrivateKey, cfg.Mnemonic, hdPath, cfg.SignerCLIConfig)
// 	if err != nil {
// 		return txmgr.Config{}, fmt.Errorf("could not init signer: %w", err)
// 	}

// 	return txmgr.Config{
// 		ResubmissionTimeout:       cfg.ResubmissionTimeout,
// 		ChainID:                   chainID,
// 		NetworkTimeout:            cfg.NetworkTimeout,
// 		ReceiptQueryInterval:      cfg.ReceiptQueryInterval,
// 		NumConfirmations:          cfg.NumConfirmations,
// 		SafeAbortNonceTooLowCount: cfg.SafeAbortNonceTooLowCount,
// 		Signer:                    signerFactory(chainID),
// 		From:                      from,
// 	}, nil
// }
