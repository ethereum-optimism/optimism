package fetch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/ethereum-optimism/optimism/op-fetcher/pkg/fetcher/fetch/script"
	"github.com/urfave/cli/v2"
)

type FetcherConfig struct {
	L1RPCURL     string
	ChainConfigs map[uint64]script.ChainConfig
}

func NewFetcherConfig(ctx *cli.Context) (*FetcherConfig, error) {
	configDir := ctx.String(ChainConfigDirFlag.Name)
	configs, err := readChainTomls(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read chain configs: %w", err)
	}

	chainConfigs := make(map[uint64]script.ChainConfig)
	for chainId, config := range configs {
		chainConfigs[chainId] = config
	}

	return &FetcherConfig{
		L1RPCURL:     ctx.String(L1RPCURLFlag.Name),
		ChainConfigs: chainConfigs,
	}, nil
}

// readChainTomls reads all toml files in a given directory and returns a
// map of chainId to ChainConfig
func readChainTomls(dirPath string) (map[uint64]script.ChainConfig, error) {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	configs := make(map[uint64]script.ChainConfig)
	for _, file := range files {
		if file.IsDir() ||
			!strings.HasSuffix(file.Name(), ".toml") ||
			file.Name() == "superchain.toml" {
			continue
		}

		filePath := filepath.Join(dirPath, file.Name())
		var config script.ChainConfig

		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
		}

		if err := toml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal TOML from %s: %w", filePath, err)
		}

		if config.ChainId == 0 {
			return nil, fmt.Errorf("chain id is not set for %s", filePath)
		}

		config.ChainName = strings.TrimSuffix(file.Name(), ".toml")
		configs[config.ChainId] = config
	}

	return configs, nil
}
