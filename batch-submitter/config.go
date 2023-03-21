package batchsubmitter

import (
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli"

	"github.com/ethereum-optimism/optimism/batch-submitter/flags"
)

var (
	// ErrSequencerPrivKeyOrMnemonic signals that the user tried to set both
	// sequencer wallet derivation methods or neither of them.
	ErrSequencerPrivKeyOrMnemonic = errors.New("either sequencer-private-key " +
		"or mnemonic + sequencer-hd-path must be set")

	// ErrProposererPrivKeyOrMnemonic signals that the user tried to set
	// both proposer wallet derivation methods or neither of them.
	ErrProposerPrivKeyOrMnemonic = errors.New("either proposer-private-key " +
		"or mnemonic + proposer-hd-path must be set")

	// ErrSameSequencerAndProposerHDPath signals that the user specified the
	// same sequencer and proposer derivations paths, which otherwise would
	// lead to the two using the same wallet.
	ErrSameSequencerAndProposerHDPath = errors.New("sequencer-hd-path and " +
		"proposer-hd-path must be distinct when using mnemonic")

	// ErrSameSequencerAndProposerPrivKey signals that the user specified
	// the same sequencer and proposer private keys, which otherwise would
	// lead to the two using the same wallet.
	ErrSameSequencerAndProposerPrivKey = errors.New("sequencer-priv-key and " +
		"proposer-priv-key must be distinct")

	// ErrInvalidBatchType  signals that an unsupported batch type is being
	// configured. The default is "legacy" and the options are "legacy" or
	// "zlib"
	ErrInvalidBatchType = errors.New("invalid batch type")

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

	// MinL1TxSize is the minimum size in bytes of any L1 transactions generated
	// by the batch submitter.
	MinL1TxSize uint64

	// MaxL1TxSize is the maximum size in bytes of any L1 transactions generated
	// by the batch submitter.
	MaxL1TxSize uint64

	// MaxPlaintextL1TxSize is the maximum size in bytes of the plaintext tx
	// data encoded in batches.
	MaxPlaintextBatchSize uint64

	// MinStateRootElements is the minimum number of state root elements that
	// can be submitted in single proposer batch.
	MinStateRootElements uint64

	// MaxStateRootElements is the maximum number of state root elements that
	// can be submitted in single proposer batch.
	MaxStateRootElements uint64

	// MaxTxBatchCount is the maximum number of L2 transactions that can ever be
	// in a batch.
	MaxTxBatchCount uint64

	// MaxBatchSubmissionTime is the maximum amount of time that we will
	// wait before submitting an under-sized batch.
	MaxBatchSubmissionTime time.Duration

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

	// FinalityConfirmations is the number of confirmations that we should wait
	// before submitting state roots for CTC elements.
	FinalityConfirmations uint64

	// RunTxBatchSubmitter determines whether or not to run the tx batch
	// submitter.
	RunTxBatchSubmitter bool

	// RunStateBatchSubmitter determines whether or not to run the state batch
	// submitter.
	RunStateBatchSubmitter bool

	//SafeMinimumEtherBalance is the safe minimum amount of ether the batch
	//submitter key should hold before it starts to log errors.
	SafeMinimumEtherBalance uint64

	// ClearPendingTxs is a boolean to clear the pending transactions in the
	// mempool on startup.
	ClearPendingTxs bool

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

	// BlockOffset is the offset between the CTC contract start and the L2 geth
	// blocks.
	BlockOffset uint64

	// SequencerPrivateKey the private key of the wallet used to submit
	// transactions to the CTC contract.
	SequencerPrivateKey string

	// PropopserPrivateKey the private key of the wallet used to submit
	// transaction to the SCC contract.
	ProposerPrivateKey string

	// Mnemonic is the HD seed used to derive the wallet private keys for both
	// the sequence and proposer. Must be used in conjunction with
	// SequencerHDPath and ProposerHDPath.
	Mnemonic string

	// SequencerHDPath is the derivation path used to obtain the private key for
	// the sequencer transactions.
	SequencerHDPath string

	// ProposerHDPath is the derivation path used to obtain the private key for
	// the proposer transactions.
	ProposerHDPath string

	// SequencerBatchType represents the type of batch the sequencer submits.
	SequencerBatchType string

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

// NewConfig parses the Config from the provided flags or environment variables.
// This method fails if ValidateConfig deems the configuration to be malformed.
func NewConfig(ctx *cli.Context) (Config, error) {
	cfg := Config{
		/* Required Flags */
		BuildEnv:                  ctx.GlobalString(flags.BuildEnvFlag.Name),
		EthNetworkName:            ctx.GlobalString(flags.EthNetworkNameFlag.Name),
		L1EthRpc:                  ctx.GlobalString(flags.L1EthRpcFlag.Name),
		L2EthRpc:                  ctx.GlobalString(flags.L2EthRpcFlag.Name),
		CTCAddress:                ctx.GlobalString(flags.CTCAddressFlag.Name),
		SCCAddress:                ctx.GlobalString(flags.SCCAddressFlag.Name),
		MinL1TxSize:               ctx.GlobalUint64(flags.MinL1TxSizeFlag.Name),
		MaxL1TxSize:               ctx.GlobalUint64(flags.MaxL1TxSizeFlag.Name),
		MaxPlaintextBatchSize:     ctx.GlobalUint64(flags.MaxPlaintextBatchSizeFlag.Name),
		MinStateRootElements:      ctx.GlobalUint64(flags.MinStateRootElementsFlag.Name),
		MaxStateRootElements:      ctx.GlobalUint64(flags.MaxStateRootElementsFlag.Name),
		MaxBatchSubmissionTime:    ctx.GlobalDuration(flags.MaxBatchSubmissionTimeFlag.Name),
		PollInterval:              ctx.GlobalDuration(flags.PollIntervalFlag.Name),
		NumConfirmations:          ctx.GlobalUint64(flags.NumConfirmationsFlag.Name),
		SafeAbortNonceTooLowCount: ctx.GlobalUint64(flags.SafeAbortNonceTooLowCountFlag.Name),
		ResubmissionTimeout:       ctx.GlobalDuration(flags.ResubmissionTimeoutFlag.Name),
		FinalityConfirmations:     ctx.GlobalUint64(flags.FinalityConfirmationsFlag.Name),
		RunTxBatchSubmitter:       ctx.GlobalBool(flags.RunTxBatchSubmitterFlag.Name),
		RunStateBatchSubmitter:    ctx.GlobalBool(flags.RunStateBatchSubmitterFlag.Name),
		SafeMinimumEtherBalance:   ctx.GlobalUint64(flags.SafeMinimumEtherBalanceFlag.Name),
		ClearPendingTxs:           ctx.GlobalBool(flags.ClearPendingTxsFlag.Name),
		/* Optional Flags */
		LogLevel:            ctx.GlobalString(flags.LogLevelFlag.Name),
		LogTerminal:         ctx.GlobalBool(flags.LogTerminalFlag.Name),
		SentryEnable:        ctx.GlobalBool(flags.SentryEnableFlag.Name),
		SentryDsn:           ctx.GlobalString(flags.SentryDsnFlag.Name),
		SentryTraceRate:     ctx.GlobalDuration(flags.SentryTraceRateFlag.Name),
		BlockOffset:         ctx.GlobalUint64(flags.BlockOffsetFlag.Name),
		SequencerPrivateKey: ctx.GlobalString(flags.SequencerPrivateKeyFlag.Name),
		ProposerPrivateKey:  ctx.GlobalString(flags.ProposerPrivateKeyFlag.Name),
		Mnemonic:            ctx.GlobalString(flags.MnemonicFlag.Name),
		SequencerHDPath:     ctx.GlobalString(flags.SequencerHDPathFlag.Name),
		ProposerHDPath:      ctx.GlobalString(flags.ProposerHDPathFlag.Name),
		SequencerBatchType:  ctx.GlobalString(flags.SequencerBatchType.Name),
		MetricsServerEnable: ctx.GlobalBool(flags.MetricsServerEnableFlag.Name),
		MetricsHostname:     ctx.GlobalString(flags.MetricsHostnameFlag.Name),
		MetricsPort:         ctx.GlobalUint64(flags.MetricsPortFlag.Name),
		DisableHTTP2:        ctx.GlobalBool(flags.HTTP2DisableFlag.Name),
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
	_, err := log.LvlFromString(cfg.LogLevel)
	if err != nil {
		return err
	}

	// Enforce that either sequencer-private-key or mnemonic + sequencer-hd-path
	// is enabled, but not both or neither.
	usingSequencerPrivateKey := cfg.SequencerPrivateKey != ""
	usingSequencerHDPath := cfg.Mnemonic != "" && cfg.SequencerHDPath != ""
	if usingSequencerPrivateKey == usingSequencerHDPath {
		return ErrSequencerPrivKeyOrMnemonic
	}

	// Enforce that either proposer-private-key or mnemonic + proposer-hd-path
	// is enabled, but not both or neither.
	usingProposerPrivateKey := cfg.ProposerPrivateKey != ""
	usingProposerHDPath := cfg.Mnemonic != "" && cfg.ProposerHDPath != ""
	if usingProposerPrivateKey == usingProposerHDPath {
		return ErrProposerPrivKeyOrMnemonic
	}

	// If mnemonic is used, the sequencer-hd-path and proposer-hd-path must
	// differ to avoid resuing the same wallet for both.
	if cfg.Mnemonic != "" && cfg.SequencerHDPath == cfg.ProposerHDPath {
		return ErrSameSequencerAndProposerHDPath
	}

	// If private keys are used, ensure the keys are different to avoid resuing
	// the same wallet for both.
	if usingSequencerPrivateKey && usingProposerPrivateKey &&
		cfg.SequencerPrivateKey == cfg.ProposerPrivateKey {

		return ErrSameSequencerAndProposerPrivKey
	}

	usingTypedBatches := cfg.SequencerBatchType != ""
	validBatchType := cfg.SequencerBatchType == "legacy" || cfg.SequencerBatchType == "zlib"
	if usingTypedBatches && !validBatchType {
		return ErrInvalidBatchType
	}

	// Ensure the Sentry Data Source Name is set when using Sentry.
	if cfg.SentryEnable && cfg.SentryDsn == "" {
		return ErrSentryDSNNotSet
	}

	return nil
}
