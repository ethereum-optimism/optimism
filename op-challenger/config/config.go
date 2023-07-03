package config

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

var (
	ErrMissingL1EthRPC      = errors.New("missing l1 eth rpc url")
	ErrMissingGameAddress   = errors.New("missing game address")
	ErrMissingAlphabetTrace = errors.New("missing alphabet trace")
)

// Config is a well typed config that is parsed from the CLI params.
// This also contains config options for auxiliary services.
// It is used to initialize the challenger.
type Config struct {
	L1EthRpc      string         // L1 RPC Url
	GameAddress   common.Address // Address of the fault game
	AlphabetTrace string         // String for the AlphabetTraceProvider

	TxMgrConfig   txmgr.CLIConfig
	RPCConfig     oprpc.CLIConfig
	LogConfig     oplog.CLIConfig
	MetricsConfig opmetrics.CLIConfig
	PprofConfig   oppprof.CLIConfig
}

func NewConfig(L1EthRpc string,
	GameAddress common.Address,
	AlphabetTrace string,
	TxMgrConfig txmgr.CLIConfig,
	RPCConfig oprpc.CLIConfig,
	LogConfig oplog.CLIConfig,
	MetricsConfig opmetrics.CLIConfig,
	PprofConfig oppprof.CLIConfig,
) Config {
	return Config{
		L1EthRpc,
		GameAddress,
		AlphabetTrace,
		TxMgrConfig,
		RPCConfig,
		LogConfig,
		MetricsConfig,
		PprofConfig,
	}
}

func (c Config) Check() error {
	if c.L1EthRpc == "" {
		return ErrMissingL1EthRPC
	}
	if c.GameAddress == (common.Address{}) {
		return ErrMissingGameAddress
	}
	if c.AlphabetTrace == "" {
		return ErrMissingAlphabetTrace
	}
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

// NewConfigFromCLI parses the Config from the provided flags or environment variables.
func NewConfigFromCLI(ctx *cli.Context) (*Config, error) {
	if err := flags.CheckRequired(ctx); err != nil {
		return nil, err
	}
	dgfAddress, err := opservice.ParseAddress(ctx.String(flags.DGFAddressFlag.Name))
	if err != nil {
		return nil, err
	}

	txMgrConfig := txmgr.ReadCLIConfig(ctx)
	rpcConfig := oprpc.ReadCLIConfig(ctx)
	logConfig := oplog.ReadCLIConfig(ctx)
	metricsConfig := opmetrics.ReadCLIConfig(ctx)
	pprofConfig := oppprof.ReadCLIConfig(ctx)

	return &Config{
		// Required Flags
		L1EthRpc:      ctx.String(flags.L1EthRpcFlag.Name),
		GameAddress:   dgfAddress,
		AlphabetTrace: ctx.String(flags.AlphabetFlag.Name),
		TxMgrConfig:   txMgrConfig,
		// Optional Flags
		RPCConfig:     rpcConfig,
		LogConfig:     logConfig,
		MetricsConfig: metricsConfig,
		PprofConfig:   pprofConfig,
	}, nil
}
