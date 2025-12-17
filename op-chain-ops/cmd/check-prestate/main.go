package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-prestate/prestate"
	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-prestate/registry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-prestate/types"
	"github.com/ethereum-optimism/optimism/op-program/prestates"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/superchain"
	"github.com/mattn/go-isatty"
	"golang.org/x/exp/maps"
)

type FPProgramType interface {
	FindVersions(log log.Logger, prestateVersion string) (
		elCommitInfo types.CommitInfo,
		fppCommitInfo types.CommitInfo,
		superChainRegistryCommit string,
		prestateConfigs *superchain.ChainConfigLoader)
}

func main() {
	color := isatty.IsTerminal(os.Stderr.Fd())
	handler := log.NewTerminalHandler(os.Stderr, color)
	oplog.SetGlobalLogHandler(handler)
	log := log.NewLogger(handler)

	// Define the flag variables
	var (
		prestateHashStr string
		chainsStr       string
	)

	// Define and parse the command-line flags
	flag.StringVar(&prestateHashStr, "prestate-hash", "", "Specify the absolute prestate hash to verify")
	flag.StringVar(&chainsStr, "chains", "", "List of chains to consider in the report. Comma separated. Default: all chains in the superchain-registry")

	var versionsOverrideFile string
	flag.StringVar(&versionsOverrideFile, "versions-file", "", "Override the prestate versions TOML file")

	// Parse the command-line arguments
	flag.Parse()
	if prestateHashStr == "" {
		log.Crit("--prestate-hash is required")
	}
	chainFilter := func(chainName string) bool {
		return true
	}
	var filteredChainNames []string
	if chainsStr != "" {
		chains := make(map[string]bool)
		for _, chain := range strings.Split(chainsStr, ",") {
			chains[strings.TrimSpace(chain)] = true
		}
		chainFilter = func(chainName string) bool {
			return chains[chainName]
		}
		filteredChainNames = maps.Keys(chains)
	}
	prestateHash := common.HexToHash(prestateHashStr)
	if prestateHash == (common.Hash{}) {
		log.Crit("--prestate-hash is invalid")
	}

	prestateReleases, err := prestates.LoadReleases(versionsOverrideFile)
	if err != nil {
		log.Crit("Failed to load prestate releases list", "err", err)
	}

	var prestateVersion string
	var prestateType string
	for version, prestates := range prestateReleases.Prestates {
		for _, prestate := range prestates {
			if common.HexToHash(prestate.Hash) == prestateHash {
				prestateVersion = version
				prestateType = prestate.Type
				break
			}
		}
	}
	if prestateVersion == "" {
		log.Crit("Failed to find a prestate release with hash", "hash", prestateHash)
	}
	log.Info("Found prestate", "version", prestateVersion, "type", prestateType)

	var prestateImpl FPProgramType
	switch prestateType {
	case "cannon32", "cannon64", "interop":
		prestateImpl = prestate.NewOPProgramPrestate()
	case "cannon-kona":
		prestateImpl = prestate.NewKonaPrestate()
	default:
		log.Crit("Invalid prestate type", "type", prestateType)
	}
	elCommitInfo, fppCommitInfo, commit, prestateConfigs := prestateImpl.FindVersions(log, prestateVersion)
	if err != nil {
		log.Crit("Failed to load configuration for prestate info", "err", err)
	}

	prestateNames := prestateConfigs.ChainNames()

	latestConfigs, err := registry.LatestSuperchainConfigs()
	if err != nil {
		log.Crit("Failed to get latest superchain configs", "err", err)
	}

	knownChains := make(map[string]bool)
	supportedChains := make([]string, 0) // Not null for json serialization
	outdatedChains := make(map[string]types.OutdatedChain)
	for _, name := range prestateNames {
		if !chainFilter(name) {
			continue
		}
		knownChains[name] = true
		diff, err := checkConfig(name, prestateConfigs, latestConfigs)
		if err != nil {
			log.Crit("Failed to check config", "chain", name, "err", err)
		}
		if diff != nil {
			outdatedChains[name] = types.OutdatedChain{
				Name: name,
				Diff: diff,
			}
		} else {
			supportedChains = append(supportedChains, name)
		}
	}

	missingChains := make([]string, 0) // Not null for json serialization
	expectedChainNames := filteredChainNames
	if len(expectedChainNames) == 0 {
		expectedChainNames = latestConfigs.ChainNames()
	}
	for _, chainName := range expectedChainNames {
		if !chainFilter(chainName) {
			continue
		}
		if !knownChains[chainName] {
			missingChains = append(missingChains, chainName)
		}
	}

	report := types.PrestateInfo{
		Hash:               prestateHash,
		Version:            prestateVersion,
		Type:               prestateType,
		FppProgram:         fppCommitInfo,
		ExecutionClient:    elCommitInfo,
		SuperchainRegistry: commitInfo("superchain-registry", commit, "main", "superchain"),
		UpToDateChains:     supportedChains,
		OutdatedChains:     maps.Values(outdatedChains),
		MissingChains:      missingChains,
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(report); err != nil {
		log.Crit("Failed to encode report", "err", err)
	}
}

func checkConfig(network string, actual *superchain.ChainConfigLoader, expected *superchain.ChainConfigLoader) (*types.Diff, error) {
	actualChainID, err := actual.ChainIDByName(network)
	if err != nil {
		return nil, fmt.Errorf("failed to get actual chain ID for %v: %w", network, err)
	}
	expectedChainID, err := expected.ChainIDByName(network)
	if err != nil {
		return nil, fmt.Errorf("failed to get expected chain ID for %v: %w", network, err)
	}
	if actualChainID != expectedChainID {
		return &types.Diff{
			Msg:      "Chain ID mismatch",
			Prestate: actualChainID,
			Latest:   expectedChainID,
		}, nil
	}
	actualChain, err := actual.GetChain(actualChainID)
	if err != nil {
		return nil, fmt.Errorf("failed to get actual chain for %v: %w", network, err)
	}
	expectedChain, err := expected.GetChain(expectedChainID)
	if err != nil {
		return nil, fmt.Errorf("failed to get expected chain for %v: %w", network, err)
	}
	actualConfig, err := actualChain.Config()
	if err != nil {
		return nil, fmt.Errorf("failed to get config for actual chain %v: %w", network, err)
	}
	expectedConfig, err := expectedChain.Config()
	if err != nil {
		return nil, fmt.Errorf("failed to get config for expected chain %v: %w", network, err)
	}
	configDiff, err := checkChainConfig(actualConfig, expectedConfig)
	if err != nil {
		return nil, err
	}
	if configDiff != nil {
		return configDiff, nil
	}
	actualGenesis, err := actualChain.GenesisData()
	if err != nil {
		return nil, fmt.Errorf("failed to get genesis for actual chain %v: %w", network, err)
	}
	expectedGenesis, err := expectedChain.GenesisData()
	if err != nil {
		return nil, fmt.Errorf("failed to get genesis for expected chain %v: %w", network, err)
	}
	if !bytes.Equal(actualGenesis, expectedGenesis) {
		return &types.Diff{
			Msg:      "Genesis mismatch",
			Prestate: string(actualGenesis),
			Latest:   string(expectedGenesis),
		}, nil
	}
	return nil, nil
}

func checkChainConfig(actual *superchain.ChainConfig, expected *superchain.ChainConfig) (*types.Diff, error) {
	actualStr, err := toml.Marshal(actual)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal actual chain config: %w", err)
	}
	expectedStr, err := toml.Marshal(expected)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal expected chain config: %w", err)
	}
	if !bytes.Equal(actualStr, expectedStr) {
		return &types.Diff{
			Msg:      "Chain config mismatch",
			Prestate: actual,
			Latest:   expected,
		}, nil
	}
	return nil, nil
}
func commitInfo(repository string, commit string, mainBranch string, dir string) types.CommitInfo {
	return types.CommitInfo{
		Commit:  commit,
		DiffUrl: fmt.Sprintf("https://github.com/ethereum-optimism/%s/compare/%s...%s", repository, commit, mainBranch),
		DiffCmd: fmt.Sprintf("git fetch && git diff %s...origin/%s %s", commit, mainBranch, dir),
	}
}
