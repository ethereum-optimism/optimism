package registry

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ethereum-optimism/optimism/op-core/superchain"
)

const superchainRegistryURL = "https://github.com/ethereum-optimism/superchain-registry.git"

// LatestSuperchainConfigs loads the configs from the superchain-registry main branch.
func LatestSuperchainConfigs() (*superchain.ChainConfigLoader, error) {
	return SuperchainConfigsForCommit("main")
}

// SuperchainConfigsForCommit builds a ChainConfigLoader for the superchain-registry
// at registryCommit (a commit SHA, or a ref like "main"). It clones the registry at
// that commit into a temp dir and runs op-core's sync-superchain.sh in external mode
// to produce a bundle in the Go format the ChainConfigLoader reads — the same
// conversion the build uses for the pinned submodule, but for an arbitrary commit,
// leaving the repo's pinned submodule untouched.
func SuperchainConfigsForCommit(registryCommit string) (*superchain.ChainConfigLoader, error) {
	dir, err := os.MkdirTemp("", "checkprestate")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	srDir := filepath.Join(dir, "superchain-registry")
	if err := cloneRegistryAtCommit(srDir, registryCommit); err != nil {
		return nil, err
	}

	script, err := opCoreSyncScriptPath()
	if err != nil {
		return nil, err
	}
	outZip := filepath.Join(dir, "superchain-configs.zip")
	cmd := exec.Command("bash", script)
	cmd.Env = append(os.Environ(),
		"SUPERCHAIN_REGISTRY_DIR="+srDir,
		"SUPERCHAIN_CONFIGS_OUT="+outZip,
	)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to build superchain config bundle for %q: %w", registryCommit, err)
	}

	configBytes, err := os.ReadFile(outZip)
	if err != nil {
		return nil, fmt.Errorf("failed to read generated superchain-configs.zip: %w", err)
	}
	return superchain.NewChainConfigLoader(configBytes)
}

// cloneRegistryAtCommit shallow-fetches the superchain-registry at ref (a commit
// SHA or branch name) into dir. GitHub serves arbitrary commit SHAs to `git fetch`.
func cloneRegistryAtCommit(dir, ref string) error {
	// Defense-in-depth: ref is git-derived (a release-tag gitlink SHA or "main"),
	// never free-form input — but reject a leading-dash value so it can't be parsed
	// as a git option in the fetch/checkout below.
	if ref == "" || strings.HasPrefix(ref, "-") {
		return fmt.Errorf("refusing unsafe superchain-registry ref %q", ref)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create registry dir: %w", err)
	}
	steps := [][]string{
		{"init", "--quiet"},
		{"remote", "add", "origin", superchainRegistryURL},
		{"fetch", "--quiet", "--depth", "1", "origin", ref},
		{"checkout", "--quiet", "--detach", "FETCH_HEAD"},
	}
	for _, args := range steps {
		var stderr bytes.Buffer
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git %s failed for superchain-registry %q: %w (%s)",
				strings.Join(args, " "), ref, err, strings.TrimSpace(stderr.String()))
		}
	}
	return nil
}

// opCoreSyncScriptPath locates op-core/superchain/sync-superchain.sh by walking up
// from the working directory until it's found. check-prestate is run from a monorepo
// checkout (it resolves kona-client release tags via git), so the script is always an
// ancestor of the working directory. Walking up — rather than `git rev-parse
// --show-toplevel` — also works without a .git dir (e.g. a downloaded source archive)
// and isn't confused by nested or worktree git setups.
func opCoreSyncScriptPath() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	rel := filepath.Join("op-core", "superchain", "sync-superchain.sh")
	for {
		candidate := filepath.Join(dir, rel)
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find %s in any ancestor of the working directory; "+
				"run check-prestate from within the optimism monorepo checkout", rel)
		}
		dir = parent
	}
}
