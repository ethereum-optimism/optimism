package manager

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadConfigDefaultsWithoutFile(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join(t.TempDir(), "missing.yaml"), ConfigOverrides{})
	require.Error(t, err)
	require.Nil(t, cfg)

	wd := t.TempDir()
	oldWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(wd))
	defer func() {
		require.NoError(t, os.Chdir(oldWD))
	}()

	cfg, err = LoadConfig("", ConfigOverrides{})
	require.NoError(t, err)
	require.Equal(t, ".oprm/releases", cfg.RunsDir)
	require.Equal(t, "develop", cfg.BaseBranch)
	require.Equal(t, "ethereum-optimism", cfg.GitHub.Owner)
	require.Equal(t, "optimism", cfg.GitHub.Repo)
	require.Equal(t, "../op-geth", cfg.OpGeth.CheckoutPath)
}

func TestLoadConfigMergesFileAndOverrides(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(`runs_dir: custom/runs
base_branch: release-branch
github:
  owner: example
  repo: repo
`), 0o644))

	cfg, err := LoadConfig(configPath, ConfigOverrides{RunsDir: "override/runs", BaseBranch: "develop"})
	require.NoError(t, err)
	require.Equal(t, filepath.Clean("override/runs"), cfg.RunsDir)
	require.Equal(t, "develop", cfg.BaseBranch)
	require.Equal(t, "example", cfg.GitHub.Owner)
	require.Equal(t, "repo", cfg.GitHub.Repo)
	require.Equal(t, "ethereum-optimism", cfg.OpGeth.Owner)
	require.Equal(t, "op-geth", cfg.OpGeth.Repo)
	require.Equal(t, "../op-geth", cfg.OpGeth.CheckoutPath)
}

func TestLoadConfigAppliesRepoOverrides(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(`github:
  owner: upstream
  repo: optimism
op_geth:
  owner: upstream
  repo: op-geth
  checkout_path: /tmp/op-geth
`), 0o644))

	cfg, err := LoadConfig(configPath, ConfigOverrides{
		GitHubOwner:        "nonsense",
		GitHubRepo:         "optimism",
		OpGethOwner:        "nonsense",
		OpGethRepo:         "op-geth",
		OpGethCheckoutPath: "../forked-op-geth",
	})
	require.NoError(t, err)
	require.Equal(t, "nonsense", cfg.GitHub.Owner)
	require.Equal(t, "optimism", cfg.GitHub.Repo)
	require.Equal(t, "nonsense", cfg.OpGeth.Owner)
	require.Equal(t, "op-geth", cfg.OpGeth.Repo)
	require.Equal(t, "../forked-op-geth", cfg.OpGeth.CheckoutPath)
}
