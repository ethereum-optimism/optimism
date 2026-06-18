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
// the kona-client release identified by ref was built against.
//
// For tags that still carry the legacy `op-core/superchain/superchain-registry-commit.txt`
// pin (v1.5.1 up to the submodule-coupling change) it reads that file — preferred
// when present so historical prestate checks are unchanged. For newer tags the
// canonical pin is the superchain-registry git submodule, so it reads the submodule
// gitlink commit instead (trying the current root path and the historical path under
// packages/contracts-bedrock/lib). If the tag isn't present locally, the function
// fetches it from origin first.
func fetchSuperchainRegistryCommit(ref string) (string, error) {
	if err := ensureRefAvailable(ref); err != nil {
		return "", err
	}

	// Legacy pinned-commit file, present on older tags.
	const legacyPath = "op-core/superchain/superchain-registry-commit.txt"
	if stdout, _, err := runGit("show", fmt.Sprintf("%s:%s", ref, legacyPath)); err == nil {
		if sha := strings.TrimSpace(stdout); sha != "" {
			return sha, nil
		}
	}

	// Newer tags: the superchain-registry submodule gitlink (at the repo root) is
	// the canonical pin.
	if sha, ok := gitlinkCommit(ref, "superchain-registry"); ok {
		return sha, nil
	}

	return "", fmt.Errorf(
		"could not determine superchain-registry commit for %s: neither %s nor a "+
			"superchain-registry submodule gitlink was found at that ref", ref, legacyPath)
}

// gitlinkCommit returns the commit a submodule gitlink points to at ref, if path
// is a gitlink (mode 160000 / type commit) in that tree.
func gitlinkCommit(ref, path string) (string, bool) {
	stdout, _, err := runGit("ls-tree", ref, "--", path)
	if err != nil {
		return "", false
	}
	// Format: "<mode> <type> <sha>\t<path>".
	fields := strings.Fields(strings.TrimSpace(stdout))
	if len(fields) >= 3 && fields[0] == "160000" && fields[1] == "commit" {
		return fields[2], true
	}
	return "", false
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
