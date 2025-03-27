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

const (
	monorepoGoModAtTag            = "https://github.com/ethereum-optimism/optimism/raw/refs/tags/%s/go.mod"
	superchainRegistryCommitAtRef = "https://github.com/ethereum-optimism/op-geth/raw/%s/superchain-registry-commit.txt"
	superchainConfigsZipAtTag     = "https://github.com/ethereum-optimism/op-geth/raw/refs/tags/%s/superchain/superchain-configs.zip"
	syncSuperchainScript          = "https://github.com/ethereum-optimism/op-geth/raw/refs/heads/optimism/sync-superchain.sh"
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
	Name string `json:"name"`
	Diff *Diff  `json:"diff,omitempty"`
}

type CommitInfo struct {
	Commit  string `json:"commit"`
	DiffUrl string `json:"diff-url"`
	DiffCmd string `json:"diff-cmd"`
}

type Diff struct {
	Msg      string `json:"message"`
	Prestate any    `json:"prestate"`
	Latest   any    `json:"latest"`
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

	registryCommitBytes, err := fetch(fmt.Sprintf(superchainRegistryCommitAtRef, gethVersion))
	if err != nil {
		log.Crit("Failed to fetch superchain registry commit info", "err", err)
	}
	commit := strings.TrimSpace(string(registryCommitBytes))
	log.Info("Found superchain registry commit info", "commit", commit)

	prestateConfigData, err := fetch(fmt.Sprintf(superchainConfigsZipAtTag, gethVersion))
	if err != nil {
		log.Crit("Failed to fetch prestate's superchain registry config zip", "err", err)
	}
	prestateConfigs, err := superchain.NewChainConfigLoader(prestateConfigData)
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
		if !chainFilter(name) {
			continue
		}
		knownChains[name] = true
		diff, err := checkConfig(name, prestateConfigs, latestConfigs)
		if err != nil {
			log.Crit("Failed to check config", "chain", name, "err", err)
		}
		if diff != nil {
			outdatedChains[name] = OutdatedChain{
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

func checkConfig(network string, actual *superchain.ChainConfigLoader, expected *superchain.ChainConfigLoader) (*Diff, error) {
	actualChainID, err := actual.ChainIDByName(network)
	if err != nil {
		return nil, fmt.Errorf("failed to get actual chain ID for %v: %w", network, err)
	}
	expectedChainID, err := expected.ChainIDByName(network)
	if err != nil {
		return nil, fmt.Errorf("failed to get expected chain ID for %v: %w", network, err)
	}
	if actualChainID != expectedChainID {
		return &Diff{
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
		return &Diff{
			Msg:      "Genesis mismatch",
			Prestate: string(actualGenesis),
			Latest:   string(expectedGenesis),
		}, nil
	}
	return nil, nil
}

func checkChainConfig(actual *superchain.ChainConfig, expected *superchain.ChainConfig) (*Diff, error) {
	actualStr, err := toml.Marshal(actual)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal actual chain config: %w", err)
	}
	expectedStr, err := toml.Marshal(expected)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal expected chain config: %w", err)
	}
	if !bytes.Equal(actualStr, expectedStr) {
		return &Diff{
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
	script, err := fetch(syncSuperchainScript)
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

func commitInfo(repository string, commit string, mainBranch string, dir string) CommitInfo {
	return CommitInfo{
		Commit:  commit,
		DiffUrl: fmt.Sprintf("https://github.com/ethereum-optimism/%s/compare/%s...%s", repository, commit, mainBranch),
		DiffCmd: fmt.Sprintf("git fetch && git diff %s...origin/%s %s", commit, mainBranch, dir),
	}
}

func fetchMonorepoGoMod(opProgramTag string) (*modfile.File, error) {
	goModUrl := fmt.Sprintf(monorepoGoModAtTag, opProgramTag)
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
