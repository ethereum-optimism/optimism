package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/ethereum-optimism/optimism/op-program/prestates"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/mattn/go-isatty"
	"golang.org/x/mod/modfile"
)

type PrestateInfo struct {
	Hash    common.Hash `json:"hash"`
	Version string      `json:"version"`
	Type    string      `json:"type"`

	OpProgram          CommitInfo `json:"op-program"`
	OpGeth             CommitInfo `json:"op-geth"`
	SuperchainRegistry CommitInfo `json:"superchain-registry"`
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
	commit := string(registryCommitBytes)
	log.Info("Found superchain registry commit info", "commit", commit)

	report := PrestateInfo{
		Hash:               prestateHash,
		Version:            prestateVersion,
		Type:               prestateType,
		OpProgram:          commitInfo("optimism", prestateTag, "develop", ""),
		OpGeth:             commitInfo("op-geth", gethVersion, "optimism", ""),
		SuperchainRegistry: commitInfo("superchain-registry", commit, "main", "superchain"),
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(report); err != nil {
		log.Crit("Failed to encode report", "err", err)
	}
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
