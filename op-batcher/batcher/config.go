package batcher

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli"

	"github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	opsigner "github.com/ethereum-optimism/optimism/op-signer/client"
)

type Config struct {
	log             log.Logger
	L1Client        *ethclient.Client
	L2Client        *ethclient.Client
	RollupNode      *sources.RollupClient
	PollInterval    time.Duration
	TxManagerConfig txmgr.Config
	From            common.Address
	SignerFnFactory opcrypto.SignerFactory
	DaRpc           string
	NamespaceId     string

	// RollupConfig is queried at startup
	Rollup *rollup.Config

	// Channel creation parameters
	Channel ChannelConfig
}

type CLIConfig struct {
	/* Required Params */

	// L1EthRpc is the HTTP provider URL for L1.
	L1EthRpc string

	// L2EthRpc is the HTTP provider URL for the L2 execution engine.
	L2EthRpc string

	// RollupRpc is the HTTP provider URL for the L2 rollup node.
	RollupRpc string

	// The batcher tx submission safety margin (in #L1-blocks) to subtract from
	// a channel's timeout and sequencing window, to guarantee safe inclusion of
	// a channel on L1.
	SubSafetyMargin uint64

	// DaRpc is the HTTP provider URL for the Data Availability node.
	DaRpc string

	// NamespaceId is the id of the namespace of the Data Availability node.
	NamespaceId string

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

	RPCConfig oprpc.CLIConfig

	/* Optional Params */

	// MaxL1TxSize is the maximum size of a batch tx submitted to L1.
	MaxL1TxSize uint64

	// TargetL1TxSize is the target size of a batch tx submitted to L1.
	TargetL1TxSize uint64

	// TargetNumFrames is the target number of frames per channel.
	TargetNumFrames int

	// ApproxComprRatio is the approximate compression ratio (<= 1.0) of the used
	// compression algorithm.
	ApproxComprRatio float64

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
		DaRpc:                     ctx.GlobalString(flags.DaRpcFlag.Name),
		NamespaceId:               ctx.GlobalString(flags.NamespaceIdFlag.Name),
		SubSafetyMargin:           ctx.GlobalUint64(flags.SubSafetyMarginFlag.Name),
		PollInterval:              ctx.GlobalDuration(flags.PollIntervalFlag.Name),
		NumConfirmations:          ctx.GlobalUint64(flags.NumConfirmationsFlag.Name),
		SafeAbortNonceTooLowCount: ctx.GlobalUint64(flags.SafeAbortNonceTooLowCountFlag.Name),
		ResubmissionTimeout:       ctx.GlobalDuration(flags.ResubmissionTimeoutFlag.Name),

		/* Optional Flags */
		MaxL1TxSize:      ctx.GlobalUint64(flags.MaxL1TxSizeBytesFlag.Name),
		TargetL1TxSize:   ctx.GlobalUint64(flags.TargetL1TxSizeBytesFlag.Name),
		TargetNumFrames:  ctx.GlobalInt(flags.TargetNumFramesFlag.Name),
		ApproxComprRatio: ctx.GlobalFloat64(flags.ApproxComprRatioFlag.Name),
		Mnemonic:         ctx.GlobalString(flags.MnemonicFlag.Name),
		SequencerHDPath:  ctx.GlobalString(flags.SequencerHDPathFlag.Name),
		PrivateKey:       ctx.GlobalString(flags.PrivateKeyFlag.Name),
		RPCConfig:        oprpc.ReadCLIConfig(ctx),
		LogConfig:        oplog.ReadCLIConfig(ctx),
		MetricsConfig:    opmetrics.ReadCLIConfig(ctx),
		PprofConfig:      oppprof.ReadCLIConfig(ctx),
		SignerConfig:     opsigner.ReadCLIConfig(ctx),
	}
}
