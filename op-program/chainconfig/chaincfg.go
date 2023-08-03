package chainconfig

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/params"
)

var OPGoerliChainConfig, OPSepoliaChainConfig, OPMainnetChainConfig *params.ChainConfig

func init() {
	mustLoadConfig := func(chainID uint64) *params.ChainConfig {
		cfg, err := params.LoadOPStackChainConfig(chainID)
		if err != nil {
			panic(err)
		}
		return cfg
	}
	OPGoerliChainConfig = mustLoadConfig(420)
	OPSepoliaChainConfig = mustLoadConfig(11155420)
	OPMainnetChainConfig = mustLoadConfig(10)
}

var L2ChainConfigsByChainID = map[uint64]*params.ChainConfig{
	420:      OPGoerliChainConfig,
	11155420: OPSepoliaChainConfig,
	10:       OPMainnetChainConfig,
}

func RollupConfigByChainID(chainID uint64) (*rollup.Config, error) {
	config, err := rollup.LoadOPStackRollupConfig(chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rollup config for chain ID %d: %w", chainID, err)
	}
	return config, nil
}

func ChainConfigByChainID(chainID uint64) (*params.ChainConfig, error) {
	return params.LoadOPStackChainConfig(chainID)
}
