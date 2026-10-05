package main

import (
	"errors"
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

func TestVerifyFromCommitReported_RetainsActualGenerationAndFailures(t *testing.T) {
	for _, scenario := range []struct {
		name       string
		mismatch   bool
		generation error
	}{
		{name: "fresh matching bundle"},
		{name: "mismatched generated bundle", mismatch: true},
		{name: "failed generator", generation: errors.New("intentional generator failure")},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			root, _ := initGitRepo(t)
			writeFileInRepo(t, root, "mise.toml", []byte("[tools]\nforge = '1.2.3'\n"))
			writeFileInRepo(t, root, "packages/contracts-bedrock/justfile", []byte("generate-nut-bundle:\n  forge script actual-generator\n"))
			writeFileInRepo(t, root, "packages/contracts-bedrock/foundry.toml", []byte("[profile.default]\nsrc = 'src'\n"))
			commit := writeFileInRepo(t, root, "packages/contracts-bedrock/snapshots/upgrades/current-upgrade-bundle.json", []byte(`{"historical":true}`))
			bundle := "op-core/nuts/bundles/test_nut_bundle.json"
			regenerated := []byte(`{"fresh_generation":true}`)
			locked := regenerated
			if scenario.mismatch {
				locked = []byte(`{"different_locked_bundle":true}`)
			}
			require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(root, bundle)), 0755))
			require.NoError(t, os.WriteFile(filepath.Join(root, bundle), locked, 0644))
			// The report must retain the historical tool file from the detached
			// worktree, rather than copying this checkout's changed configuration.
			require.NoError(t, os.WriteFile(filepath.Join(root, "mise.toml"), []byte("[tools]\nforge = '1.8.3'\n"), 0644))
			report := filepath.Join(t.TempDir(), "report")
			entry := nuts.ForkLockEntry{Bundle: bundle, Commit: commit}
			err := verifyFromCommitReported(root, "test-fork", entry, func(contractsDir string) error {
				require.NoError(t, os.WriteFile(filepath.Join(contractsDir, "snapshots/upgrades/current-upgrade-bundle.json"), regenerated, 0644))
				return scenario.generation
			}, report)
			if scenario.generation != nil {
				require.ErrorContains(t, err, "intentional generator failure")
				require.NoFileExists(t, filepath.Join(report, "regenerated-bundle.json"))
			} else {
				if scenario.mismatch {
					require.ErrorContains(t, err, "does not match")
				} else {
					require.NoError(t, err)
				}
				actual, err := os.ReadFile(filepath.Join(report, "regenerated-bundle.json"))
				require.NoError(t, err)
				require.Equal(t, regenerated, actual)
			}
			actual, err := os.ReadFile(filepath.Join(report, "locked-bundle.json"))
			require.NoError(t, err)
			require.Equal(t, locked, actual)
			actual, err = os.ReadFile(filepath.Join(report, "mise.toml"))
			require.NoError(t, err)
			require.Equal(t, "[tools]\nforge = '1.2.3'\n", string(actual))
			for _, name := range []string{"source.json", "tracked-stage.bin", "worktree-status.txt", "submodules.txt", "submodule-inventories.json"} {
				require.FileExists(t, filepath.Join(report, name))
			}
			cmd := exec.Command("git", "worktree", "list", "--porcelain")
			cmd.Dir = root
			worktrees, err := cmd.Output()
			require.NoError(t, err)
			require.NotContains(t, string(worktrees), "verify-nuts-")
		})
	}
}
