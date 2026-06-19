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

// opCoreSyncScriptPath resolves op-core/superchain/sync-superchain.sh in the repo
// containing the current working directory.
func opCoreSyncScriptPath() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("failed to locate repo root: %w", err)
	}
	root := strings.TrimSpace(string(out))
	return filepath.Join(root, "op-core", "superchain", "sync-superchain.sh"), nil
}
