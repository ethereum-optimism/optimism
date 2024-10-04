package chainconfig

import (
	"embed"
	"encoding/json"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"
)

// OPSepoliaChainConfig loads the op-sepolia chain config. This is intended for tests that need an arbitrary, valid chain config.
func OPSepoliaChainConfig() *params.ChainConfig {
	return mustLoadChainConfig("op-sepolia")
}

//go:embed configs/*json
var customChainConfigFS embed.FS

var CustomChainConfig *params.ChainConfig
var CustomRollupConfig rollup.Config

func init() {
	// Load custom l2 genesis and rollup config from embed FS
	data, err := customChainConfigFS.ReadFile("configs/genesis-l2.json")
	if err != nil {
		panic(err)
	}
	var genesis core.Genesis
	err = json.Unmarshal(data, &genesis)
	if err != nil {
		panic(err)
	}
	CustomChainConfig = genesis.Config

	file, err := customChainConfigFS.Open("configs/rollup.json")
	if err != nil {
		panic(err)
	}
	dec := json.NewDecoder(file)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&CustomRollupConfig); err != nil {
		panic(err)
	}
	// Both configs must have the same L2 chainID
	if CustomChainConfig.ChainID.Uint64() != CustomRollupConfig.L2ChainID.Uint64() {
		panic(fmt.Errorf("mismatched genesis-l2.json chainid %d vs rollup.json chainid %d", CustomChainConfig.ChainID.Uint64(), CustomRollupConfig.L2ChainID.Uint64()))
	}
	// Do not override existing superchain registered configs
	if _, err := params.LoadOPStackChainConfig(CustomChainConfig.ChainID.Uint64()); err == nil {
		panic(fmt.Errorf("cannot override existing superchain registered config"))
	}
}

func RollupConfigByChainID(chainID uint64) (*rollup.Config, error) {
	if chainID == CustomRollupConfig.L2ChainID.Uint64() {
		return &CustomRollupConfig, nil
	}
	config, err := rollup.LoadOPStackRollupConfig(chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rollup config for chain ID %d: %w", chainID, err)
	}
	return config, nil
}

func ChainConfigByChainID(chainID uint64) (*params.ChainConfig, error) {
	if chainID == CustomChainConfig.ChainID.Uint64() {
		return CustomChainConfig, nil
	}
	return params.LoadOPStackChainConfig(chainID)
}

func mustLoadChainConfig(name string) *params.ChainConfig {
	chainCfg := chaincfg.ChainByName(name)
	if chainCfg == nil {
		panic(fmt.Errorf("unknown chain config %q", name))
	}
	cfg, err := ChainConfigByChainID(chainCfg.ChainID)
	if err != nil {
		panic(fmt.Errorf("failed to load rollup config: %q: %w", name, err))
	}
	return cfg
}
