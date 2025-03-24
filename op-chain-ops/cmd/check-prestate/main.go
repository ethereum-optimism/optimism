package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/ethereum-optimism/optimism/op-program/prestates"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/superchain"
	"github.com/mattn/go-isatty"
	"golang.org/x/exp/maps"
	"golang.org/x/mod/modfile"
)

type PrestateInfo struct {
	Hash    common.Hash `json:"hash"`
	Version string      `json:"version"`
	Type    string      `json:"type"`

	OpProgram          CommitInfo `json:"op-program"`
	OpGeth             CommitInfo `json:"op-geth"`
	SuperchainRegistry CommitInfo `json:"superchain-registry"`

	UpToDateChains []string        `json:"up-to-date-chains"`
	OutdatedChains []OutdatedChain `json:"outdated-chains"`
	MissingChains  []string        `json:"missing-chains"`
}

type OutdatedChain struct {
	Name   string `json:"name"`
	Reason string `json:"reason,omitempty"`
}

type CommitInfo struct {
	Commit  string `json:"commit"`
	DiffUrl string `json:"diff-url"`
	DiffCmd string `json:"diff-cmd"`
}

func main() {
	color := isatty.IsTerminal(os.Stderr.Fd())
	handler := log.NewTerminalHandler(os.Stderr, color)
	oplog.SetGlobalLogHandler(handler)
	log := log.NewLogger(handler)

	// Define the flag variables
	var (
		prestateHashStr string
	)

	// Define and parse the command-line flags
	flag.StringVar(&prestateHashStr, "prestate-hash", "", "Specify the absolute prestate hash to verify")

	// Parse the command-line arguments
	flag.Parse()
	if prestateHashStr == "" {
		log.Crit("--prestate-hash is required")
	}
	prestateHash := common.HexToHash(prestateHashStr)
	if prestateHash == (common.Hash{}) {
		log.Crit("--prestate-hash is invalid")
	}

	prestateReleases, err := prestates.LoadReleases("")
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
	prestateTag := fmt.Sprintf("op-program/v%s", prestateVersion)
	log.Info("Found prestate", "version", prestateVersion, "type", prestateType, "tag", prestateTag)

	modFile, err := fetchMonorepoGoMod(prestateTag)
	if err != nil {
		log.Crit("Failed to fetch go mod", "err", err)
	}
	var gethVersion string
	for _, replace := range modFile.Replace {
		if replace.Old.Path == "github.com/ethereum/go-ethereum" {
			gethVersion = replace.New.Version
			break
		}
	}
	if gethVersion == "" {
		log.Crit("Failed to find op-geth replace in go.mod")
	}
	log.Info("Found op-geth version", "version", gethVersion)

	registryCommitBytes, err := fetch(fmt.Sprintf("https://github.com/ethereum-optimism/op-geth/raw/%s/superchain-registry-commit.txt", gethVersion))
	if err != nil {
		log.Crit("Failed to fetch superchain registry commit info", "err", err)
	}
	commit := strings.TrimSpace(string(registryCommitBytes))
	log.Info("Found superchain registry commit info", "commit", commit)

	prestateConfigData, err := fetch(fmt.Sprintf("https://github.com/ethereum-optimism/op-geth/raw/refs/tags/%s/superchain/superchain-configs.zip", gethVersion))
	if err != nil {
		log.Crit("Failed to fetch prestate's superchain registry config zip", "err", err)
	}
	prestateConfigs, err := superchain.NewChainConfigReader(prestateConfigData)
	if err != nil {
		log.Crit("Failed to parse prestate's superchain registry config zip", "err", err)
	}
	prestateNames := prestateConfigs.ChainNames()

	latestConfigs, err := latestSuperchainConfigs()
	if err != nil {
		log.Crit("Failed to get latest superchain configs", "err", err)
	}

	knownChains := make(map[string]bool)
	var supportedChains []string
	outdatedChains := make(map[string]OutdatedChain)
	for _, name := range prestateNames {
		knownChains[name] = true
		msg, err := checkConfig(name, prestateConfigs, latestConfigs)
		if err != nil {
			log.Crit("Failed to check config", "chain", name, "err", err)
		}
		if msg != "" {
			outdatedChains[name] = OutdatedChain{
				Name:   name,
				Reason: msg,
			}
		} else {
			supportedChains = append(supportedChains, name)
		}
	}

	var missingChains []string
	for _, chainName := range latestConfigs.ChainNames() {
		if !knownChains[chainName] {
			missingChains = append(missingChains, chainName)
		}
	}

	report := PrestateInfo{
		Hash:               prestateHash,
		Version:            prestateVersion,
		Type:               prestateType,
		OpProgram:          commitInfo("optimism", prestateTag, "develop", ""),
		OpGeth:             commitInfo("op-geth", gethVersion, "optimism", ""),
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

func checkConfig(network string, actual *superchain.ChainConfigLoader, expected *superchain.ChainConfigLoader) (string, error) {
	actualChainID, err := actual.ChainIDByName(network)
	if err != nil {
		return "", fmt.Errorf("failed to get actual chain ID for %v: %w", network, err)
	}
	expectedChainID, err := expected.ChainIDByName(network)
	if err != nil {
		return "", fmt.Errorf("failed to get expected chain ID for %v: %w", network, err)
	}
	if actualChainID != expectedChainID {
		return fmt.Sprintf("Chain ID mismatch, prestate=%v, latest=%v", actualChainID, expectedChainID), nil
	}
	actualChain, err := actual.GetChain(actualChainID)
	if err != nil {
		return "", fmt.Errorf("failed to get actual chain for %v: %w", network, err)
	}
	expectedChain, err := expected.GetChain(expectedChainID)
	if err != nil {
		return "", fmt.Errorf("failed to get expected chain for %v: %w", network, err)
	}
	actualConfig, err := actualChain.Config()
	if err != nil {
		return "", fmt.Errorf("failed to get config for actual chain %v: %w", network, err)
	}
	expectedConfig, err := expectedChain.Config()
	if err != nil {
		return "", fmt.Errorf("failed to get config for expected chain %v: %w", network, err)
	}
	configMsg, err := checkChainConfig(actualConfig, expectedConfig)
	if err != nil {
		return "", err
	}
	if configMsg != "" {
		return configMsg, nil
	}
	actualGenesis, err := actualChain.GenesisData()
	if err != nil {
		return "", fmt.Errorf("failed to get genesis for actual chain %v: %w", network, err)
	}
	expectedGenesis, err := expectedChain.GenesisData()
	if err != nil {
		return "", fmt.Errorf("failed to get genesis for expected chain %v: %w", network, err)
	}
	if !bytes.Equal(actualGenesis, expectedGenesis) {
		return "Genesis mismatch", nil
	}
	return "", nil
}

func checkChainConfig(actual *superchain.ChainConfig, expected *superchain.ChainConfig) (string, error) {
	actualStr, err := toml.Marshal(actual)
	if err != nil {
		return "", fmt.Errorf("failed to marshal actual chain config: %w", err)
	}
	expectedStr, err := toml.Marshal(expected)
	if err != nil {
		return "", fmt.Errorf("failed to marshal expected chain config: %w", err)
	}
	if !bytes.Equal(actualStr, expectedStr) {
		return "Chain config mismatch", nil
	}
	return "", nil
}

// latestSuperchainConfigs loads the latest config from the superchain-registry main branch using the
// sync-superchain.sh script from op-geth to create a zip of configs that can be read by op-geth's ChainConfigLoader.
func latestSuperchainConfigs() (*superchain.ChainConfigLoader, error) {
	// Download the op-geth script to build the superchain config
	script, err := fetch("https://github.com/ethereum-optimism/op-geth/raw/refs/heads/optimism/sync-superchain.sh")
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
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to build superchain config zip: %w", err)
	}
	configBytes, err := os.ReadFile(filepath.Join(dir, "superchain/superchain-configs.zip"))
	if err != nil {
		return nil, fmt.Errorf("failed to read generated superchain-configs.zip: %w", err)
	}
	return superchain.NewChainConfigReader(configBytes)
}

func commitInfo(repository string, commit string, mainBranch string, dir string) CommitInfo {
	return CommitInfo{
		Commit:  commit,
		DiffUrl: fmt.Sprintf("https://github.com/ethereum-optimism/%s/compare/%s...%s", repository, commit, mainBranch),
		DiffCmd: fmt.Sprintf("git fetch && git diff %s...origin/%s %s", commit, mainBranch, dir),
	}
}

func fetchMonorepoGoMod(opProgramTag string) (*modfile.File, error) {
	goModUrl := fmt.Sprintf("https://github.com/ethereum-optimism/optimism/raw/refs/tags/%s/go.mod", opProgramTag)
	goMod, err := fetch(goModUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch go.mod: %w", err)
	}

	return modfile.Parse("go.mod", goMod, nil)
}

func fetch(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %v: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch %v: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}
