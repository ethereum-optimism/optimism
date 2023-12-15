package batcher

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-batcher/compressor"
	"github.com/ethereum-optimism/optimism/op-batcher/flags"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

type CLIConfig struct {
	// L1EthRpc is the HTTP provider URL for L1.
	L1EthRpc string

	// L2EthRpc is the HTTP provider URL for the L2 execution engine. A comma-separated list enables the active L2 provider. Such a list needs to match the number of RollupRpcs provided.
	L2EthRpc string

	// RollupRpc is the HTTP provider URL for the L2 rollup node. A comma-separated list enables the active L2 provider. Such a list needs to match the number of L2EthRpcs provided.
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

	// MaxPendingTransactions is the maximum number of concurrent pending
	// transactions sent to the transaction manager (0 == no limit).
	MaxPendingTransactions uint64

	// MaxL1TxSize is the maximum size of a batch tx submitted to L1.
	MaxL1TxSize uint64

	Stopped bool

	BatchType uint

	TxMgrConfig      txmgr.CLIConfig
	LogConfig        oplog.CLIConfig
	MetricsConfig    opmetrics.CLIConfig
	PprofConfig      oppprof.CLIConfig
	CompressorConfig compressor.CLIConfig
	RPC              oprpc.CLIConfig
}

func (c *CLIConfig) Check() error {
	if c.L1EthRpc == "" {
		return errors.New("empty L1 RPC URL")
	}
	if c.L2EthRpc == "" {
		return errors.New("empty L2 RPC URL")
	}
	if c.RollupRpc == "" {
		return errors.New("empty rollup RPC URL")
	}
	if strings.Count(c.RollupRpc, ",") != strings.Count(c.L2EthRpc, ",") {
		return errors.New("number of rollup and eth URLs must match")
	}
	if c.PollInterval == 0 {
		return errors.New("must set PollInterval")
	}
	if c.MaxL1TxSize <= 1 {
		return errors.New("MaxL1TxSize must be greater than 0")
	}
	if c.BatchType > 1 {
		return fmt.Errorf("unknown batch type: %v", c.BatchType)
	}

	if err := c.MetricsConfig.Check(); err != nil {
		return err
	}
	if err := c.PprofConfig.Check(); err != nil {
		return err
	}
	if err := c.TxMgrConfig.Check(); err != nil {
		return err
	}
	if err := c.RPC.Check(); err != nil {
		return err
	}
	return nil
}

// NewConfig parses the Config from the provided flags or environment variables.
func NewConfig(ctx *cli.Context) *CLIConfig {
	return &CLIConfig{
		/* Required Flags */
		L1EthRpc:        ctx.String(flags.L1EthRpcFlag.Name),
		L2EthRpc:        ctx.String(flags.L2EthRpcFlag.Name),
		RollupRpc:       ctx.String(flags.RollupRpcFlag.Name),
		SubSafetyMargin: ctx.Uint64(flags.SubSafetyMarginFlag.Name),
		PollInterval:    ctx.Duration(flags.PollIntervalFlag.Name),

		/* Optional Flags */
		MaxPendingTransactions: ctx.Uint64(flags.MaxPendingTransactionsFlag.Name),
		MaxChannelDuration:     ctx.Uint64(flags.MaxChannelDurationFlag.Name),
		MaxL1TxSize:            ctx.Uint64(flags.MaxL1TxSizeBytesFlag.Name),
		Stopped:                ctx.Bool(flags.StoppedFlag.Name),
		BatchType:              ctx.Uint(flags.BatchTypeFlag.Name),
		TxMgrConfig:            txmgr.ReadCLIConfig(ctx),
		LogConfig:              oplog.ReadCLIConfig(ctx),
		MetricsConfig:          opmetrics.ReadCLIConfig(ctx),
		PprofConfig:            oppprof.ReadCLIConfig(ctx),
		CompressorConfig:       compressor.ReadCLIConfig(ctx),
		RPC:                    oprpc.ReadCLIConfig(ctx),
	}
}
