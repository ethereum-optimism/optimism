package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-prestate/prestate"
	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-prestate/types"
	"github.com/ethereum-optimism/optimism/op-program/prestates"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/superchain"
	"github.com/mattn/go-isatty"
	"golang.org/x/exp/maps"
)

const (
	syncSuperchainScript = "https://github.com/ethereum-optimism/op-geth/raw/refs/heads/optimism/sync-superchain.sh"
)

type FPProgramType interface {
	FindVersions(log log.Logger, prestateVersion string) (
		elCommitInfo types.CommitInfo,
		fppCommitInfo types.CommitInfo,
		superChainRegistryCommit string,
		prestateConfigs types.ChainConfigs)
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
	default:
		log.Crit("Invalid prestate type", "type", prestateType)
	}
	elCommitInfo, fppCommitInfo, commit, prestateConfigs := prestateImpl.FindVersions(log, prestateVersion)
	if err != nil {
		log.Crit("Failed to load configuration for prestate info", "err", err)
	}

	prestateNames := prestateConfigs.ChainNames()

	latestConfigs, err := latestSuperchainConfigs()
	if err != nil {
		log.Crit("Failed to get latest superchain configs", "err", err)
	}

	knownChains := make(map[string]bool)
	var supportedChains []string
	outdatedChains := make(map[string]types.OutdatedChain)
	for _, name := range prestateNames {
		if !chainFilter(name) {
			continue
		}
		knownChains[name] = true
		diff, err := checkConfig(name, prestateConfigs, types.NewChainConfigLoaderAdapter(latestConfigs))
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

func checkConfig(network string, actual types.ChainConfigs, expected types.ChainConfigs) (*types.Diff, error) {
	actualChainID, actualConfig, actualGenesis, err := actual.ChainConfig(network)
	if err != nil {
		return nil, fmt.Errorf("failed to get actual chain config for %s: %w", network, err)
	}
	expectedChainID, expectedConfig, expectedGenesis, err := expected.ChainConfig(network)
	if err != nil {
		return nil, fmt.Errorf("failed to get expected chain config for %s: %w", network, err)
	}

	if actualChainID != expectedChainID {
		return &types.Diff{
			Msg:      "Chain ID mismatch",
			Prestate: actualChainID,
			Latest:   expectedChainID,
		}, nil
	}
	configDiff, err := checkChainConfig(actualConfig, expectedConfig)
	if err != nil {
		return nil, err
	}
	if configDiff != nil {
		return configDiff, nil
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

// latestSuperchainConfigs loads the latest config from the superchain-registry main branch using the
// sync-superchain.sh script from op-geth to create a zip of configs that can be read by op-geth's ChainConfigLoader.
func latestSuperchainConfigs() (*superchain.ChainConfigLoader, error) {
	// Download the op-geth script to build the superchain config
	script, err := prestate.Fetch(syncSuperchainScript)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sync-superchain.sh script: %w", err)
	}
	dir, err := os.MkdirTemp("", "checkprestate")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)
	if err := os.Mkdir(filepath.Join(dir, "superchain"), 0o700); err != nil {
		return nil, fmt.Errorf("failed to create superchain dir: %w", err)
	}
	scriptPath := filepath.Join(dir, "sync-superchain.sh")
	if err := os.WriteFile(scriptPath, script, 0o700); err != nil {
		return nil, fmt.Errorf("failed to write sync-superchain.sh: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "superchain-registry-commit.txt"), []byte("main"), 0o600); err != nil {
		return nil, fmt.Errorf("failed to write superchain-registry-commit.txt: %w", err)
	}
	cmd := exec.Command(scriptPath)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to build superchain config zip: %w", err)
	}
	configBytes, err := os.ReadFile(filepath.Join(dir, "superchain/superchain-configs.zip"))
	if err != nil {
		return nil, fmt.Errorf("failed to read generated superchain-configs.zip: %w", err)
	}
	return superchain.NewChainConfigLoader(configBytes)
}

func commitInfo(repository string, commit string, mainBranch string, dir string) types.CommitInfo {
	return types.CommitInfo{
		Commit:  commit,
		DiffUrl: fmt.Sprintf("https://github.com/ethereum-optimism/%s/compare/%s...%s", repository, commit, mainBranch),
		DiffCmd: fmt.Sprintf("git fetch && git diff %s...origin/%s %s", commit, mainBranch, dir),
	}
}
