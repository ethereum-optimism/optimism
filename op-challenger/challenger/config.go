package challenger

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/urfave/cli"

	flags "github.com/ethereum-optimism/optimism/op-challenger/flags"
	sources "github.com/ethereum-optimism/optimism/op-node/sources"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	txmgr "github.com/ethereum-optimism/optimism/op-service/txmgr"
)

// Config contains the well typed fields that are used to initialize the challenger.
// It is intended for programmatic use.
type Config struct {
	L2OutputOracleAddr common.Address
	DisputeGameFactory common.Address
	NetworkTimeout     time.Duration
	TxManager          txmgr.TxManager
	L1Client           *ethclient.Client
	RollupClient       *sources.RollupClient
}

// CLIConfig is a well typed config that is parsed from the CLI params.
// This also contains config options for auxiliary services.
// It is transformed into a `Config` before the Challenger is started.
type CLIConfig struct {
	// L1EthRpc is the HTTP provider URL for L1.
	L1EthRpc string

	// RollupRpc is the HTTP provider URL for the rollup node.
	RollupRpc string

	// L2OOAddress is the L2OutputOracle contract address.
	L2OOAddress string

	// DGFAddress is the DisputeGameFactory contract address.
	DGFAddress string

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
		L1EthRpc:    ctx.GlobalString(flags.L1EthRpcFlag.Name),
		RollupRpc:   ctx.GlobalString(flags.RollupRpcFlag.Name),
		L2OOAddress: ctx.GlobalString(flags.L2OOAddressFlag.Name),
		DGFAddress:  ctx.GlobalString(flags.DGFAddressFlag.Name),
		TxMgrConfig: txmgr.ReadCLIConfig(ctx),
		// Optional Flags
		RPCConfig:     oprpc.ReadCLIConfig(ctx),
		LogConfig:     oplog.ReadCLIConfig(ctx),
		MetricsConfig: opmetrics.ReadCLIConfig(ctx),
		PprofConfig:   oppprof.ReadCLIConfig(ctx),
	}
}
