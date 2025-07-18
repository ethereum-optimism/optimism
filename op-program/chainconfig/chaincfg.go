package chainconfig

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/superutil"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"
)

var (
	ErrMissingChainConfig = errors.New("missing chain config")
	errChainNotFound      = errors.New("chain not found")
)

// OPSepoliaChainConfig loads the op-sepolia chain config. This is intended for tests that need an arbitrary, valid chain config.
func OPSepoliaChainConfig() *params.ChainConfig {
	return mustLoadChainConfig("op-sepolia")
}

//go:embed configs/*json
var customChainConfigFS embed.FS

func CustomChainIDs() ([]eth.ChainID, error) {
	return customChainIDs(customChainConfigFS)
}

func customChainIDs(customChainFS embed.FS) ([]eth.ChainID, error) {
	entries, err := customChainFS.ReadDir("configs")
	if err != nil {
		return nil, fmt.Errorf("failed to list custom configs: %w", err)
	}
	var chainIDs []eth.ChainID
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), "-genesis-l2.json") {
			chainID, err := eth.ParseDecimalChainID(strings.TrimSuffix(entry.Name(), "-genesis-l2.json"))
			if err != nil {
				return nil, fmt.Errorf("incorrectly named genesis-l2 config (%s): %w", entry.Name(), err)
			}
			chainIDs = append(chainIDs, chainID)
		}
	}

	return chainIDs, nil
}

// RollupConfigByChainID locates the rollup config from either the superchain-registry or the embed.
// Returns ErrMissingChainConfig if the rollup config is not found.
func RollupConfigByChainID(chainID eth.ChainID) (*rollup.Config, error) {
	config, err := rollup.LoadOPStackRollupConfig(eth.EvilChainIDToUInt64(chainID))
	if err == nil {
		return config, err
	}
	return rollupConfigByChainID(chainID, customChainConfigFS)
}

func rollupConfigByChainID(chainID eth.ChainID, customChainFS embed.FS) (*rollup.Config, error) {
	// Load custom rollup configs from embed FS
	file, err := customChainFS.Open(fmt.Sprintf("configs/%v-rollup.json", chainID))
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%w: no rollup config available for chain ID: %v", ErrMissingChainConfig, chainID)
	} else if err != nil {
		return nil, fmt.Errorf("failed to get rollup config for chain ID %v: %w", chainID, err)
	}
	defer file.Close()

	var customRollupConfig rollup.Config
	return &customRollupConfig, customRollupConfig.ParseRollupConfig(file)
}

// ChainConfigByChainID locates the genesis chain config from either the superchain-registry or the embed.
// Returns ErrMissingChainConfig if the chain config is not found.
func ChainConfigByChainID(chainID eth.ChainID) (*params.ChainConfig, error) {
	config, err := superutil.LoadOPStackChainConfigFromChainID(eth.EvilChainIDToUInt64(chainID))
	if err == nil {
		return config, err
	}
	return chainConfigByChainID(chainID, customChainConfigFS)
}

func chainConfigByChainID(chainID eth.ChainID, customChainFS embed.FS) (*params.ChainConfig, error) {
	// Load from custom chain configs from embed FS
	data, err := customChainFS.ReadFile(fmt.Sprintf("configs/%v-genesis-l2.json", chainID))
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%w: no chain config available for chain ID: %v", ErrMissingChainConfig, chainID)
	} else if err != nil {
		return nil, fmt.Errorf("failed to get chain config for chain ID %v: %w", chainID, err)
	}
	var genesis core.Genesis
	err = json.Unmarshal(data, &genesis)
	if err != nil {
		return nil, fmt.Errorf("failed to parse chain config for chain ID %v: %w", chainID, err)
	}
	return genesis.Config, nil
}

func mustLoadChainConfig(name string) *params.ChainConfig {
	chainCfg := chaincfg.ChainByName(name)
	if chainCfg == nil {
		panic(fmt.Errorf("%w: unknown chain config %q", errChainNotFound, name))
	}
	cfg, err := ChainConfigByChainID(eth.ChainIDFromUInt64(chainCfg.ChainID))
	if err != nil {
		panic(fmt.Errorf("failed to load rollup config: %q: %w", name, err))
	}
	return cfg
}

// DependencySetByChainID locates the dependency set from either the superchain-registry or the embed.
// Returns ErrMissingChainConfig if the dependency set is not found.
func DependencySetByChainID(chainID eth.ChainID) (depset.DependencySet, error) {
	depSet, err := depset.FromRegistry(chainID)
	if err == nil {
		return depSet, nil
	}
	return dependencySetByChainID(chainID, customChainConfigFS)
}

func dependencySetByChainID(chainID eth.ChainID, customChainFS embed.FS) (depset.DependencySet, error) {
	// Load custom dependency set configs from embed FS
	data, err := customChainFS.ReadFile("configs/depsets.json")
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%w: no dependency set available for chain ID: %v", ErrMissingChainConfig, chainID)
	} else if err != nil {
		return nil, fmt.Errorf("failed to get dependency set for chain ID %v: %w", chainID, err)
	}

	var depSets []*depset.StaticConfigDependencySet

	err = json.Unmarshal(data, &depSets)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dependency set for chain ID %v: %w", chainID, err)
	}
	for _, depSet := range depSets {
		if depSet.HasChain(chainID) {
			return depSet, nil
		}
	}
	return nil, fmt.Errorf("%w: no dependency set config includes chain ID: %v", errChainNotFound, chainID)
}

func CheckConfigFilenames() error {
	return checkConfigFilenames(customChainConfigFS, "configs")
}

func checkConfigFilenames(customChainFS embed.FS, configPath string) error {
	entries, err := customChainFS.ReadDir(configPath)
	if err != nil {
		return fmt.Errorf("failed to check custom configs directory: %w", err)
	}
	var rollupChainIDs []eth.ChainID
	var genesisChainIDs []eth.ChainID
	for _, entry := range entries {
		entryName := entry.Name()
		switch {
		case "placeholder.json" == entryName:
		case "depsets.json" == entryName:
		case strings.HasSuffix(entryName, "-genesis-l2.json"):
			id, err := eth.ParseDecimalChainID(strings.TrimSuffix(entry.Name(), "-genesis-l2.json"))
			if err != nil {
				return fmt.Errorf("incorrectly named genesis-l2 config (%s). expected <chain-id>-genesis-l2.json: %w", entryName, err)
			}
			genesisChainIDs = append(genesisChainIDs, id)
		case strings.HasSuffix(entryName, "-rollup.json"):
			id, err := eth.ParseDecimalChainID(strings.TrimSuffix(entry.Name(), "-rollup.json"))
			if err != nil {
				return fmt.Errorf("incorrectly named rollup config (%s). expected <chain-id>-rollup.json: %w", entryName, err)
			}
			rollupChainIDs = append(rollupChainIDs, id)
		default:
			return fmt.Errorf("invalid config file name: %s, Make sure that the only files in the custom config directory are placeholder.json, depsets.json, <chain-id>-genesis-l2.json or <chain-id>-rollup.json", entryName)
		}
	}
	if !slices.Equal(rollupChainIDs, genesisChainIDs) {
		return fmt.Errorf("mismatched chain IDs in custom configs: rollup chain IDs %v, genesis chain IDs %v. Make sure that the rollup and genesis configs have the same set of chain IDs prefixes", rollupChainIDs, genesisChainIDs)
	}
	return nil
}
