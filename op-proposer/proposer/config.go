package proposer

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/urfave/cli"

	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum-optimism/optimism/op-proposer/flags"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

// Config contains the well typed fields that are used to initialize the output submitter.
// It is intended for programmatic use.
type Config struct {
	L2OutputOracleAddr common.Address
	PollInterval       time.Duration
	TxManager          txmgr.TxManager
	L1Client           *ethclient.Client
	RollupClient       *sources.RollupClient
	AllowNonFinalized  bool
}

// CLIConfig is a well typed config that is parsed from the CLI params.
// This also contains config options for auxiliary services.
// It is transformed into a `Config` before the L2 output submitter is started.
type CLIConfig struct {
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

	// AllowNonFinalized can be set to true to propose outputs
	// for L2 blocks derived from non-finalized L1 data.
	AllowNonFinalized bool

	TxMgrConfig txmgr.CLIConfig

	RPCConfig oprpc.CLIConfig

	LogConfig oplog.CLIConfig

	MetricsConfig opmetrics.CLIConfig

	PprofConfig oppprof.CLIConfig
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
	if err := c.TxMgrConfig.Check(); err != nil {
		return err
	}
	return nil
}

// NewConfig parses the Config from the provided flags or environment variables.
func NewConfig(ctx *cli.Context) CLIConfig {
	return CLIConfig{
		// Required Flags
		L1EthRpc:     ctx.GlobalString(flags.L1EthRpcFlag.Name),
		RollupRpc:    ctx.GlobalString(flags.RollupRpcFlag.Name),
		L2OOAddress:  ctx.GlobalString(flags.L2OOAddressFlag.Name),
		PollInterval: ctx.GlobalDuration(flags.PollIntervalFlag.Name),
		TxMgrConfig:  txmgr.ReadCLIConfig(ctx),
		// Optional Flags
		AllowNonFinalized: ctx.GlobalBool(flags.AllowNonFinalizedFlag.Name),
		RPCConfig:         oprpc.ReadCLIConfig(ctx),
		LogConfig:         oplog.ReadCLIConfig(ctx),
		MetricsConfig:     opmetrics.ReadCLIConfig(ctx),
		PprofConfig:       oppprof.ReadCLIConfig(ctx),
	}
}
