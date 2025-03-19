package fetch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/ethereum-optimism/optimism/op-fetcher/pkg/fetcher/fetch/script"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

type Fetcher struct {
	L1RPCURL     string
	ChainConfigs map[uint64]script.ChainConfig
	OutputDir    string
	lgr          log.Logger
}

func NewFetcherFromCli(cliCtx *cli.Context) (*Fetcher, error) {
	logCfg := oplog.ReadCLIConfig(cliCtx)
	lgr := oplog.NewLogger(oplog.AppOut(cliCtx), logCfg)
	configDir := cliCtx.String(ChainConfigDirFlag.Name)
	outputDir := cliCtx.String(OutputDirFlag.Name)
	configs, err := readChainTomls(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read chain configs: %w", err)
	}

	chainConfigs := make(map[uint64]script.ChainConfig)
	for chainId, config := range configs {
		chainConfigs[chainId] = config
	}

	return &Fetcher{
		L1RPCURL:     cliCtx.String(L1RPCURLFlag.Name),
		ChainConfigs: chainConfigs,
		OutputDir:    outputDir,
		lgr:          lgr,
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

		zeroAddress := common.Address{}
		if config.Addresses.SystemConfigProxy == zeroAddress ||
			config.Addresses.L1StandardBridgeProxy == zeroAddress {
			return nil, fmt.Errorf("SystemConfigProxy and L1StandardBridgeProxy must be set for %s", filePath)
		}

		config.ChainName = strings.TrimSuffix(file.Name(), ".toml")
		configs[config.ChainId] = config
	}

	return configs, nil
}
