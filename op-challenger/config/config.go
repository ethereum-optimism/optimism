package config

import (
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"

	flags "github.com/ethereum-optimism/optimism/op-challenger/flags"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	txmgr "github.com/ethereum-optimism/optimism/op-service/txmgr"
)

var (
	ErrMissingL1EthRPC       = errors.New("missing l1 eth rpc url")
	ErrMissingRollupRpc      = errors.New("missing rollup rpc url")
	ErrMissingL2OOAddress    = errors.New("missing l2 output oracle contract address")
	ErrMissingDGFAddress     = errors.New("missing dispute game factory contract address")
	ErrInvalidNetworkTimeout = errors.New("invalid network timeout")
	ErrMissingTxMgrConfig    = errors.New("missing tx manager config")
	ErrMissingRPCConfig      = errors.New("missing rpc config")
	ErrMissingLogConfig      = errors.New("missing log config")
	ErrMissingMetricsConfig  = errors.New("missing metrics config")
	ErrMissingPprofConfig    = errors.New("missing pprof config")
)

// Config is a well typed config that is parsed from the CLI params.
// This also contains config options for auxiliary services.
// It is used to initialize the challenger.
type Config struct {
	// L1EthRpc is the HTTP provider URL for L1.
	L1EthRpc string

	// RollupRpc is the HTTP provider URL for the rollup node.
	RollupRpc string

	// L2OOAddress is the L2OutputOracle contract address.
	L2OOAddress common.Address

	// DGFAddress is the DisputeGameFactory contract address.
	DGFAddress common.Address

	// NetworkTimeout is the timeout for network requests.
	NetworkTimeout time.Duration

	TxMgrConfig *txmgr.CLIConfig

	RPCConfig *oprpc.CLIConfig

	LogConfig *oplog.CLIConfig

	MetricsConfig *opmetrics.CLIConfig

	PprofConfig *oppprof.CLIConfig
}

func (c Config) Check() error {
	if c.L1EthRpc == "" {
		return ErrMissingL1EthRPC
	}
	if c.RollupRpc == "" {
		return ErrMissingRollupRpc
	}
	if c.L2OOAddress == (common.Address{}) {
		return ErrMissingL2OOAddress
	}
	if c.DGFAddress == (common.Address{}) {
		return ErrMissingDGFAddress
	}
	if c.NetworkTimeout == 0 {
		return ErrInvalidNetworkTimeout
	}
	if c.TxMgrConfig == nil {
		return ErrMissingTxMgrConfig
	}
	if c.RPCConfig == nil {
		return ErrMissingRPCConfig
	}
	if c.LogConfig == nil {
		return ErrMissingLogConfig
	}
	if c.MetricsConfig == nil {
		return ErrMissingMetricsConfig
	}
	if c.PprofConfig == nil {
		return ErrMissingPprofConfig
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

// NewConfig creates a Config with all optional values set to the CLI default value
func NewConfig(
	L1EthRpc string,
	RollupRpc string,
	L2OOAddress common.Address,
	DGFAddress common.Address,
	NetworkTimeout time.Duration,
	TxMgrConfig *txmgr.CLIConfig,
	RPCConfig *oprpc.CLIConfig,
	LogConfig *oplog.CLIConfig,
	MetricsConfig *opmetrics.CLIConfig,
	PprofConfig *oppprof.CLIConfig,
) *Config {
	return &Config{
		L1EthRpc:       L1EthRpc,
		RollupRpc:      RollupRpc,
		L2OOAddress:    L2OOAddress,
		DGFAddress:     DGFAddress,
		NetworkTimeout: NetworkTimeout,
		TxMgrConfig:    TxMgrConfig,
		RPCConfig:      RPCConfig,
		LogConfig:      LogConfig,
		MetricsConfig:  MetricsConfig,
		PprofConfig:    PprofConfig,
	}
}

// NewConfigFromCLI parses the Config from the provided flags or environment variables.
func NewConfigFromCLI(ctx *cli.Context) (*Config, error) {
	if err := flags.CheckRequired(ctx); err != nil {
		return nil, err
	}
	l1EthRpc := ctx.String(flags.L1EthRpcFlag.Name)
	if l1EthRpc == "" {
		return nil, ErrMissingL1EthRPC
	}
	rollupRpc := ctx.String(flags.RollupRpcFlag.Name)
	if rollupRpc == "" {
		return nil, ErrMissingRollupRpc
	}
	l2ooAddress, err := opservice.ParseAddress(ctx.String(flags.L2OOAddressFlag.Name))
	if err != nil {
		return nil, ErrMissingL2OOAddress
	}
	dgfAddress, err := opservice.ParseAddress(ctx.String(flags.DGFAddressFlag.Name))
	if err != nil {
		return nil, ErrMissingDGFAddress
	}

	txMgrConfig := txmgr.ReadCLIConfig(ctx)
	rpcConfig := oprpc.ReadCLIConfig(ctx)
	logConfig := oplog.ReadCLIConfig(ctx)
	metricsConfig := opmetrics.ReadCLIConfig(ctx)
	pprofConfig := oppprof.ReadCLIConfig(ctx)

	return &Config{
		// Required Flags
		L1EthRpc:    l1EthRpc,
		RollupRpc:   rollupRpc,
		L2OOAddress: l2ooAddress,
		DGFAddress:  dgfAddress,
		TxMgrConfig: &txMgrConfig,
		// Optional Flags
		RPCConfig:     &rpcConfig,
		LogConfig:     &logConfig,
		MetricsConfig: &metricsConfig,
		PprofConfig:   &pprofConfig,
	}, nil
}
