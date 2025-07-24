package super

import (
	"errors"
	"fmt"
	"os"

	"github.com/HashKeyChain/verse/op-challenger/game/fault/trace/vm"
	"github.com/HashKeyChain/verse/op-node/chaincfg"
	"github.com/HashKeyChain/verse/op-node/rollup"
	"github.com/HashKeyChain/verse/op-service/eth"
)

var ErrDuplicateChain = errors.New("duplicate chain")

type RollupConfigs struct {
	cfgs map[eth.ChainID]*rollup.Config
}

func NewRollupConfigs(vmCfg vm.Config) (*RollupConfigs, error) {
	cfgs := make(map[eth.ChainID]*rollup.Config)
	for _, network := range vmCfg.Networks {
		cfg, err := chaincfg.GetRollupConfig(network)
		if err != nil {
			return nil, err
		}
		if err := addConfig(cfgs, cfg); err != nil {
			return nil, err
		}
	}
	for _, path := range vmCfg.RollupConfigPaths {
		cfg, err := loadRollupConfig(path)
		if err != nil {
			return nil, err
		}
		if err := addConfig(cfgs, cfg); err != nil {
			return nil, err
		}
	}
	return &RollupConfigs{
		cfgs: cfgs,
	}, nil
}

func NewRollupConfigsFromParsed(rollupCfgs ...*rollup.Config) (*RollupConfigs, error) {
	cfgs := make(map[eth.ChainID]*rollup.Config)
	for _, cfg := range rollupCfgs {
		if err := addConfig(cfgs, cfg); err != nil {
			return nil, err
		}
	}
	return &RollupConfigs{cfgs: cfgs}, nil
}

func addConfig(cfgs map[eth.ChainID]*rollup.Config, cfg *rollup.Config) error {
	chainID := eth.ChainIDFromBig(cfg.L2ChainID)
	if _, ok := cfgs[chainID]; ok {
		return fmt.Errorf("%w: %v", ErrDuplicateChain, chainID)
	}
	cfgs[chainID] = cfg
	return nil
}

func (c *RollupConfigs) Get(chainID eth.ChainID) (*rollup.Config, bool) {
	cfg, ok := c.cfgs[chainID]
	return cfg, ok
}

func loadRollupConfig(rollupConfigPath string) (*rollup.Config, error) {
	file, err := os.Open(rollupConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read rollup config: %w", err)
	}
	defer file.Close()

	var rollupConfig rollup.Config
	return &rollupConfig, rollupConfig.ParseRollupConfig(file)
}
