package compare

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum-optimism/optimism/op-fetcher/pkg/fetcher/fetch/script"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

type Comparator struct {
	CombinedAddresses map[uint64]ChainInfo
	ChainList         map[uint64]ChainListEntry
	FetchOutput       map[uint64]script.ChainConfig
	lgr               log.Logger
}

type AddressesFile struct {
	Chains map[uint64]ChainInfo
}

type ChainInfo struct {
	script.Addresses
	script.Roles
}

type ChainListEntry struct {
	ChainID          uint64                  `json:"chainId"`
	FaultProofStatus script.FaultProofStatus `json:"fault_proofs"`
}

func NewComparatorFromCli(ctx *cli.Context) (*Comparator, error) {
	logCfg := oplog.ReadCLIConfig(ctx)
	lgr := oplog.NewLogger(oplog.AppOut(ctx), logCfg)
	fetchOutputDir := ctx.String(FetchedDir.Name)
	addressesFile := ctx.String(AddressesFileFlag.Name)
	chainListFile := ctx.String(ChainListFileFlag.Name)
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

	chainList, err := readChainList(chainListFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read chain list: %w", err)
	}

	return &Comparator{
		CombinedAddresses: combinedAddresses,
		ChainList:         chainList,
		FetchOutput:       chainConfigs,
		lgr:               lgr,
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
		if file.IsDir() ||
			!strings.HasSuffix(file.Name(), ".json") {
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
			return nil, fmt.Errorf("chain id is not set for %s", filePath)
		}

		configs[config.ChainId] = config
	}

	if len(configs) == 0 {
		return nil, fmt.Errorf("no chain configs found in %s", dirPath)
	}

	return configs, nil
}

func readChainList(filePath string) (map[uint64]ChainListEntry, error) {
	chainListData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read chain list file: %w", err)
	}

	var chainListSlice []ChainListEntry
	if err := json.Unmarshal(chainListData, &chainListSlice); err != nil {
		return nil, fmt.Errorf("failed to unmarshal chain list file: %w", err)
	}

	chainListMap := make(map[uint64]ChainListEntry)
	for _, entry := range chainListSlice {
		chainListMap[entry.ChainID] = entry
	}

	return chainListMap, nil
}
