package prestate

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

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
	fppCommitInfo = types.NewCommitInfo("op-rs", "kona", prestateTag, "main", "")

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

func fetchSuperchainRegistryCommit(ref string) (string, error) {
	endpoint := "https://api.github.com/repos/op-rs/kona/contents/crates/protocol/registry/superchain-registry?ref=" +
		url.QueryEscape(ref)

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

	// Parse error payloads from GitHub if status != 200.
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch superchain-registry version, http status: %s", resp.Status)
	}

	// Success path: expect a single "submodule" content object with "sha".
	var content struct {
		Type string `json:"type"` // should be "submodule"
		SHA  string `json:"sha"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&content); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if content.Type != "submodule" {
		return "", fmt.Errorf("expected a submodule got type %q", content.Type)
	}
	return content.SHA, nil
}
