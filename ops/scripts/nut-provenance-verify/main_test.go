package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/nuts"
	"github.com/stretchr/testify/require"
)

// initGitRepo creates a bare-minimum git repo with an initial commit
// and returns the repo root and the commit SHA.
func initGitRepo(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "config", "commit.gpgsign", "false"},
		{"git", "commit", "--allow-empty", "-m", "init"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "cmd %v failed: %s", args, out)
	}

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	require.NoError(t, err)

	return dir, string(out[:len(out)-1]) // trim newline
}

// writeFileInRepo creates a file at a relative path within the repo and commits it.
func writeFileInRepo(t *testing.T, root, relPath string, content []byte) string {
	t.Helper()
	absPath := filepath.Join(root, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(absPath), 0755))
	require.NoError(t, os.WriteFile(absPath, content, 0644))

	cmd := exec.Command("git", "add", relPath)
	cmd.Dir = root
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "add "+relPath)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))

	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = root
	sha, err := cmd.Output()
	require.NoError(t, err)
	return string(sha[:len(sha)-1])
}

func TestVerifyFromCommit_MatchingBundle(t *testing.T) {
	root, _ := initGitRepo(t)

	bundleContent := []byte(`{"metadata":{"version":"1.0.0"},"transactions":[]}`)
	bundlePath := "packages/contracts-bedrock/snapshots/upgrades/current-upgrade-bundle.json"
	commit := writeFileInRepo(t, root, bundlePath, bundleContent)

	// Write the "committed" bundle that verifyFromCommit compares against.
	committedBundleRel := "op-core/nuts/bundles/test_nut_bundle.json"
	committedBundlePath := filepath.Join(root, committedBundleRel)
	require.NoError(t, os.MkdirAll(filepath.Dir(committedBundlePath), 0755))
	require.NoError(t, os.WriteFile(committedBundlePath, bundleContent, 0644))

	entry := nuts.ForkLockEntry{
		Bundle: committedBundleRel,
		Commit: commit,
	}

	// Generator is a no-op: the bundle file already exists at the commit.
	noopGenerator := func(contractsDir string) error { return nil }

	err := verifyFromCommit(root, "test-fork", entry, noopGenerator)
	require.NoError(t, err)
}

func TestVerifyFromCommit_MismatchedBundle(t *testing.T) {
	root, _ := initGitRepo(t)

	bundleContent := []byte(`{"metadata":{"version":"1.0.0"},"transactions":[]}`)
	bundlePath := "packages/contracts-bedrock/snapshots/upgrades/current-upgrade-bundle.json"
	commit := writeFileInRepo(t, root, bundlePath, bundleContent)

	// Write a different bundle as the "committed" version.
	committedBundleRel := "op-core/nuts/bundles/test_nut_bundle.json"
	committedBundlePath := filepath.Join(root, committedBundleRel)
	require.NoError(t, os.MkdirAll(filepath.Dir(committedBundlePath), 0755))
	require.NoError(t, os.WriteFile(committedBundlePath, []byte(`{"modified":true}`), 0644))

	entry := nuts.ForkLockEntry{
		Bundle: committedBundleRel,
		Commit: commit,
	}

	noopGenerator := func(contractsDir string) error { return nil }

	err := verifyFromCommit(root, "test-fork", entry, noopGenerator)
	require.ErrorContains(t, err, "bundle regenerated from commit")
}

func TestVerifyFromCommit_GeneratorModifiesBundle(t *testing.T) {
	root, _ := initGitRepo(t)

	// Commit an initial bundle.
	originalContent := []byte(`{"metadata":{"version":"1.0.0"},"transactions":[]}`)
	bundlePath := "packages/contracts-bedrock/snapshots/upgrades/current-upgrade-bundle.json"
	commit := writeFileInRepo(t, root, bundlePath, originalContent)

	// The committed bundle matches what the generator will produce.
	regeneratedContent := []byte(`{"metadata":{"version":"2.0.0"},"transactions":[{"new":true}]}`)
	committedBundleRel := "op-core/nuts/bundles/test_nut_bundle.json"
	committedBundlePath := filepath.Join(root, committedBundleRel)
	require.NoError(t, os.MkdirAll(filepath.Dir(committedBundlePath), 0755))
	require.NoError(t, os.WriteFile(committedBundlePath, regeneratedContent, 0644))

	entry := nuts.ForkLockEntry{
		Bundle: committedBundleRel,
		Commit: commit,
	}

	// Generator overwrites the bundle with new content (simulating forge regeneration).
	modifyingGenerator := func(contractsDir string) error {
		outPath := filepath.Join(contractsDir, "snapshots", "upgrades", "current-upgrade-bundle.json")
		return os.WriteFile(outPath, regeneratedContent, 0644)
	}

	err := verifyFromCommit(root, "test-fork", entry, modifyingGenerator)
	require.NoError(t, err)
}
