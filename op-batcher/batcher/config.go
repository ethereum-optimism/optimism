package batcher

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli"

	"github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-batcher/metrics"
	"github.com/ethereum-optimism/optimism/op-batcher/rpc"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	opsigner "github.com/ethereum-optimism/optimism/op-signer/client"
)

type Config struct {
	log        log.Logger
	metr       metrics.Metricer
	L1Client   *ethclient.Client
	L2Client   *ethclient.Client
	RollupNode *sources.RollupClient

	PollInterval time.Duration
	From         common.Address

	TxManagerConfig txmgr.Config

	// RollupConfig is queried at startup
	Rollup *rollup.Config

	// Channel builder parameters
	Channel ChannelConfig
}

// Check ensures that the [Config] is valid.
func (c *Config) Check() error {
	if err := c.Rollup.Check(); err != nil {
		return err
	}
	if err := c.Channel.Check(); err != nil {
		return err
	}
	return nil
}

type CLIConfig struct {
	/* Required Params */

	// L1EthRpc is the HTTP provider URL for L1.
	L1EthRpc string

	// L2EthRpc is the HTTP provider URL for the L2 execution engine.
	L2EthRpc string

	// RollupRpc is the HTTP provider URL for the L2 rollup node.
	RollupRpc string

	// MaxChannelDuration is the maximum duration (in #L1-blocks) to keep a
	// channel open. This allows to more eagerly send batcher transactions
	// during times of low L2 transaction volume. Note that the effective
	// L1-block distance between batcher transactions is then MaxChannelDuration
	// + NumConfirmations because the batcher waits for NumConfirmations blocks
	// after sending a batcher tx and only then starts a new channel.
	//
	// If 0, duration checks are disabled.
	MaxChannelDuration uint64

	// The batcher tx submission safety margin (in #L1-blocks) to subtract from
	// a channel's timeout and sequencing window, to guarantee safe inclusion of
	// a channel on L1.
	SubSafetyMargin uint64

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

	// SequencerHDPath is the derivation path used to obtain the private key for
	// batched submission of sequencer transactions.
	SequencerHDPath string

	// PrivateKey is the private key used to submit sequencer transactions.
	PrivateKey string

	RPCConfig rpc.CLIConfig

	/* Optional Params */

	// TxManagerTimeout is the max amount of time to wait for the [txmgr].
	// This will default to: 10 * time.Minute.
	TxManagerTimeout time.Duration

	// OfflineGasEstimation specifies whether the batcher should calculate
	// gas estimations offline using the [core.IntrinsicGas] function.
	OfflineGasEstimation bool

	// MaxL1TxSize is the maximum size of a batch tx submitted to L1.
	MaxL1TxSize uint64

	// TargetL1TxSize is the target size of a batch tx submitted to L1.
	TargetL1TxSize uint64

	// TargetNumFrames is the target number of frames per channel.
	TargetNumFrames int

	// ApproxComprRatio is the approximate compression ratio (<= 1.0) of the used
	// compression algorithm.
	ApproxComprRatio float64

	Stopped bool

	LogConfig oplog.CLIConfig

	MetricsConfig opmetrics.CLIConfig

	PprofConfig oppprof.CLIConfig

	// SignerConfig contains the client config for op-signer service
	SignerConfig opsigner.CLIConfig
}

func (c CLIConfig) Check() error {
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
	if err := c.SignerConfig.Check(); err != nil {
		return err
	}
	return nil
}

// NewConfig parses the Config from the provided flags or environment variables.
func NewConfig(ctx *cli.Context) CLIConfig {
	return CLIConfig{
		/* Required Flags */
		L1EthRpc:                  ctx.GlobalString(flags.L1EthRpcFlag.Name),
		L2EthRpc:                  ctx.GlobalString(flags.L2EthRpcFlag.Name),
		RollupRpc:                 ctx.GlobalString(flags.RollupRpcFlag.Name),
		SubSafetyMargin:           ctx.GlobalUint64(flags.SubSafetyMarginFlag.Name),
		PollInterval:              ctx.GlobalDuration(flags.PollIntervalFlag.Name),
		NumConfirmations:          ctx.GlobalUint64(flags.NumConfirmationsFlag.Name),
		SafeAbortNonceTooLowCount: ctx.GlobalUint64(flags.SafeAbortNonceTooLowCountFlag.Name),
		ResubmissionTimeout:       ctx.GlobalDuration(flags.ResubmissionTimeoutFlag.Name),

		/* Optional Flags */
		OfflineGasEstimation: ctx.GlobalBool(flags.OfflineGasEstimationFlag.Name),
		TxManagerTimeout:     ctx.GlobalDuration(flags.TxManagerTimeoutFlag.Name),
		MaxChannelDuration:   ctx.GlobalUint64(flags.MaxChannelDurationFlag.Name),
		MaxL1TxSize:          ctx.GlobalUint64(flags.MaxL1TxSizeBytesFlag.Name),
		TargetL1TxSize:       ctx.GlobalUint64(flags.TargetL1TxSizeBytesFlag.Name),
		TargetNumFrames:      ctx.GlobalInt(flags.TargetNumFramesFlag.Name),
		ApproxComprRatio:     ctx.GlobalFloat64(flags.ApproxComprRatioFlag.Name),
		Stopped:              ctx.GlobalBool(flags.StoppedFlag.Name),
		Mnemonic:             ctx.GlobalString(flags.MnemonicFlag.Name),
		SequencerHDPath:      ctx.GlobalString(flags.SequencerHDPathFlag.Name),
		PrivateKey:           ctx.GlobalString(flags.PrivateKeyFlag.Name),
		RPCConfig:            rpc.ReadCLIConfig(ctx),
		LogConfig:            oplog.ReadCLIConfig(ctx),
		MetricsConfig:        opmetrics.ReadCLIConfig(ctx),
		PprofConfig:          oppprof.ReadCLIConfig(ctx),
		SignerConfig:         opsigner.ReadCLIConfig(ctx),
	}
}
