package archiver

import (
	"time"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-blob-archiver/flags"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
)

type CLIConfig struct {
	// L1EthRpc is the HTTP provider URL for L1.
	L1EthRpc string

	// RollupRpc is the HTTP provider URL for the L2 rollup node.
	// RollupRpc string

	// PollInterval is the delay between querying L1 for more transactions
	// and archiving the blobs.
	PollInterval   time.Duration
	NetworkTimeout time.Duration

	// Stopped bool

	S3BucketName      string
	S3Region          string
	BatchInboxAddress string
	LogConfig         oplog.CLIConfig
	MetricsConfig     opmetrics.CLIConfig
	PprofConfig       oppprof.CLIConfig
}

func (c *CLIConfig) Check() error {
	// TODO(7512): check the sanity of flags loaded directly https://github.com/ethereum-optimism/optimism/issues/7512

	if err := c.MetricsConfig.Check(); err != nil {
		return err
	}
	if err := c.PprofConfig.Check(); err != nil {
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
		L1EthRpc: ctx.String(flags.L1EthRpcFlag.Name),
		// should be RollupConfig
		RollupRpc:    ctx.String(flags.RollupRpcFlag.Name),
		PollInterval: ctx.Duration(flags.PollIntervalFlag.Name),

		/* Optional Flags */
		Stopped:       ctx.Bool(flags.StoppedFlag.Name),
		LogConfig:     oplog.ReadCLIConfig(ctx),
		MetricsConfig: opmetrics.ReadCLIConfig(ctx),
		PprofConfig:   oppprof.ReadCLIConfig(ctx),
		RPC:           oprpc.ReadCLIConfig(ctx),
	}
}
