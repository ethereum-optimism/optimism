package registry

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-prestate/util"
	"github.com/ethereum/go-ethereum/superchain"
)

const (
	syncSuperchainScript = "https://raw.githubusercontent.com/ethereum-optimism/op-geth/optimism/sync-superchain.sh"
)

// LatestSuperchainConfigs loads the latest config from the superchain-registry main branch using the
// sync-superchain.sh script from op-geth to create a zip of configs that can be read by op-geth's ChainConfigLoader.
func LatestSuperchainConfigs() (*superchain.ChainConfigLoader, error) {
	return SuperchainConfigsForCommit("main")
}

func SuperchainConfigsForCommit(registryCommit string) (*superchain.ChainConfigLoader, error) {
	// Download the op-geth script to build the superchain config
	script, err := util.Fetch(syncSuperchainScript)
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
	if err := os.WriteFile(filepath.Join(dir, "superchain-registry-commit.txt"), []byte(registryCommit), 0o600); err != nil {
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
