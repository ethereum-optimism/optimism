package prestate

import (
	"bytes"
	"fmt"
	"os/exec"
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

// fetchSuperchainRegistryCommit returns the superchain-registry commit SHA that
// the kona-client release identified by ref was built against, by reading the
// pinned commit file from the local optimism monorepo checkout at that tag.
//
// Only kona-client tags that have op-core/superchain/superchain-registry-commit.txt
// are supported (v1.5.1 and later). If the tag isn't present locally, the
// function fetches it from origin before giving up.
func fetchSuperchainRegistryCommit(ref string) (string, error) {
	const path = "op-core/superchain/superchain-registry-commit.txt"

	if err := ensureRefAvailable(ref); err != nil {
		return "", err
	}

	stdout, stderr, err := runGit("show", fmt.Sprintf("%s:%s", ref, path))
	if err != nil {
		return "", fmt.Errorf("git show %s:%s failed: %w (%s)", ref, path, err, strings.TrimSpace(stderr))
	}
	sha := strings.TrimSpace(stdout)
	if sha == "" {
		return "", fmt.Errorf("empty commit SHA at %s@%s", path, ref)
	}
	return sha, nil
}

// ensureRefAvailable verifies that ref resolves in the local repo; if not, it
// attempts to fetch the tag from origin.
func ensureRefAvailable(ref string) error {
	if refExists(ref) {
		return nil
	}
	refspec := fmt.Sprintf("refs/tags/%s:refs/tags/%s", ref, ref)
	if _, stderr, err := runGit("fetch", "--quiet", "origin", refspec); err != nil {
		return fmt.Errorf("ref %q not found locally and git fetch origin %s failed: %w (%s)", ref, refspec, err, strings.TrimSpace(stderr))
	}
	if !refExists(ref) {
		return fmt.Errorf("ref %q still not found after git fetch origin %s", ref, refspec)
	}
	return nil
}

func refExists(ref string) bool {
	_, _, err := runGit("rev-parse", "--verify", "--quiet", ref+"^{commit}")
	return err == nil
}

func runGit(args ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("git", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}
