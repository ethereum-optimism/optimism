package config

import (
	"errors"
	"fmt"
	"time"

	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"

	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrMissingL1EthRPC           = errors.New("missing l1 eth rpc url")
	ErrMissingGameFactoryAddress = errors.New("missing game factory address")
	ErrMissingRollupRpc          = errors.New("missing rollup rpc url")
)

const (
	// DefaultGameWindow is the default maximum time duration in the past
	// to look for games to monitor. The default value is 11 days, which
	// is a 4 day resolution buffer plus the 7 day game finalization window.
	DefaultGameWindow = time.Duration(11 * 24 * time.Hour)
	// DefaultMonitorInterval is the default interval at which the dispute
	// monitor will check for new games to monitor.
	DefaultMonitorInterval = time.Second * 30
)

// Config is a well typed config that is parsed from the CLI params.
// It also contains config options for auxiliary services.
type Config struct {
	L1EthRpc           string         // L1 RPC Url
	GameFactoryAddress common.Address // Address of the dispute game factory
	RollupRpc          string         // The rollup node RPC URL.

	MonitorInterval time.Duration // Frequency to check for new games to monitor.
	GameWindow      time.Duration // Maximum window to look for games to monitor.

	MetricsConfig opmetrics.CLIConfig
	PprofConfig   oppprof.CLIConfig
}

func NewConfig(gameFactoryAddress common.Address, l1EthRpc string) Config {
	return Config{
		L1EthRpc:           l1EthRpc,
		GameFactoryAddress: gameFactoryAddress,

		MonitorInterval: DefaultMonitorInterval,
		GameWindow:      DefaultGameWindow,

		MetricsConfig: opmetrics.DefaultCLIConfig(),
		PprofConfig:   oppprof.DefaultCLIConfig(),
	}
}

func (c Config) Check() error {
	if c.L1EthRpc == "" {
		return ErrMissingL1EthRPC
	}
	if c.RollupRpc == "" {
		return ErrMissingRollupRpc
	}
	if c.GameFactoryAddress == (common.Address{}) {
		return ErrMissingGameFactoryAddress
	}
	if err := c.MetricsConfig.Check(); err != nil {
		return fmt.Errorf("metrics config: %w", err)
	}
	if err := c.PprofConfig.Check(); err != nil {
		return fmt.Errorf("pprof config: %w", err)
	}
	return nil
}
