package prestate

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-prestate/registry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-prestate/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/superchain"
)

type KonaPrestate struct {
}

func NewKonaPrestate() *KonaPrestate {
	return &KonaPrestate{}
}

func (p *KonaPrestate) FindVersions(log log.Logger, prestateVersion string) (
	elCommitInfo types.CommitInfo,
	fppCommitInfo types.CommitInfo,
	superChainRegistryCommit string,
	prestateConfigs *superchain.ChainConfigLoader) {

	prestateTag := fmt.Sprintf("kona-client/v%s", prestateVersion)
	log.Info("Found prestate tag", "tag", prestateTag)
	fppCommitInfo = types.NewCommitInfo("ethereum-optimism", "optimism", prestateTag, "develop", "rust/kona")

	superChainRegistryCommit, err := fetchSuperchainRegistryCommit(prestateTag)
	if err != nil {
		log.Crit("Failed to fetch superchain registry commit", "err", err)
	}

	// Kona doesn't directly depend on op-reth but uses various crates from it.
	// Skip attempting to report a specific op-reth version for now.
	elCommitInfo = types.CommitInfo{}

	// kona has its own build process to convert superchain-registry config into a custom JSON format it uses
	// Rather than re-implement that custom JSON format and work out how to convert it to the go format
	// (which could be brittle), we use the op-geth sync process to convert the superchain registry at the same commit
	// to the go format directly. This is unfortunately also potentially brittle since we have to use the latest
	// sync script from op-geth rather than a fixed version but seems like the lowest risk option.
	configs, err := registry.SuperchainConfigsForCommit(superChainRegistryCommit)
	if err != nil {
		log.Crit("Failed to fetch chain configs for prestate", "err", err)
	}
	prestateConfigs = configs
	return
}

// fetchSuperchainRegistryCommit returns the superchain-registry commit SHA that the
// kona-client release identified by ref was built against, by reading the pinned
// commit file from the optimism monorepo at that tag.
//
// Only kona-client tags that have op-core/superchain/superchain-registry-commit.txt
// are supported (v1.5.1 and later); older tags will return a 404 error.
func fetchSuperchainRegistryCommit(ref string) (string, error) {
	const path = "op-core/superchain/superchain-registry-commit.txt"
	endpoint := "https://api.github.com/repos/ethereum-optimism/optimism/contents/" + path +
		"?ref=" + url.QueryEscape(ref)

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch superchain-registry version at %s@%s, http status: %s", path, ref, resp.Status)
	}

	var content struct {
		Type     string `json:"type"`
		Encoding string `json:"encoding"`
		Content  string `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&content); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if content.Type != "file" {
		return "", fmt.Errorf("expected a file at %s@%s, got type %q", path, ref, content.Type)
	}
	if content.Encoding != "base64" {
		return "", fmt.Errorf("unexpected content encoding %q at %s@%s", content.Encoding, path, ref)
	}
	decoded, err := base64.StdEncoding.DecodeString(content.Content)
	if err != nil {
		return "", fmt.Errorf("decode base64 content at %s@%s: %w", path, ref, err)
	}
	sha := strings.TrimSpace(string(decoded))
	if sha == "" {
		return "", fmt.Errorf("empty commit SHA at %s@%s", path, ref)
	}
	return sha, nil
}
