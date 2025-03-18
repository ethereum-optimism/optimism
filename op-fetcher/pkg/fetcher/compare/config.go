package compare

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-fetcher/pkg/fetcher/fetch/script"
	"github.com/urfave/cli/v2"
)

type CompareConfig struct {
	CombinedAddresses map[uint64]ChainInfo
	FetchOutput       map[uint64]script.ChainConfig
}

type AddressesFile struct {
	Chains map[uint64]ChainInfo
}

type ChainInfo struct {
	script.Addresses
	script.Roles
}

func NewCompareConfig(ctx *cli.Context) (*CompareConfig, error) {
	fetchOutputDir := ctx.String(FetchOutputDirFlag.Name)
	addressesFile := ctx.String(AddressesFileFlag.Name)

	combinedAddresses := make(map[uint64]ChainInfo)
	addressesFileData, err := os.ReadFile(addressesFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read addresses file: %w", err)
	}

	if err := json.Unmarshal(addressesFileData, &combinedAddresses); err != nil {
		return nil, fmt.Errorf("failed to unmarshal addresses file: %w", err)
	}

	chainConfigs, err := readChainConfigs(fetchOutputDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read chain configs: %w", err)
	}

	return &CompareConfig{
		CombinedAddresses: combinedAddresses,
		FetchOutput:       chainConfigs,
	}, nil
}

// readChainConfigs reads all json files in a given directory and returns a
// map of chainId to ChainConfig
func readChainConfigs(dirPath string) (map[uint64]script.ChainConfig, error) {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	configs := make(map[uint64]script.ChainConfig)
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(dirPath, file.Name())
		var config script.ChainConfig

		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
		}

		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal TOML from %s: %w", filePath, err)
		}

		if config.ChainId == 0 {
			return nil, fmt.Errorf("chainId is 0 for %s", filePath)
		}

		configs[config.ChainId] = config
	}

	return configs, nil
}
