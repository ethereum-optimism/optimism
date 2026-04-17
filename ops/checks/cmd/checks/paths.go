package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// findRepoRoot returns the absolute path of the enclosing repo root.
// Tries `git rev-parse --show-toplevel` first; falls back to walking
// up from cwd looking for ops/checks/ (the optimism-native layout);
// falls back to cwd as a last resort.
//
// Cached on first call — commands invoke this multiple times, no
// reason to re-shell per call.
func findRepoRoot() string {
	cachedRepoRootOnce.Do(func() {
		cachedRepoRoot = discoverRepoRoot()
	})
	return cachedRepoRoot
}

var (
	cachedRepoRoot     string
	cachedRepoRootOnce sync.Once
)

func discoverRepoRoot() string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err == nil {
		if root := strings.TrimSpace(string(out)); root != "" {
			return root
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "ops", "checks")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return cwd
		}
		dir = parent
	}
}

// resolveFromRoot turns a potentially-relative path into a repo-
// root-relative absolute path. Absolute paths are returned as-is;
// empty paths return empty. Used by command handlers to normalize
// --graph / --catalog / --coverage-dir etc. so they Just Work
// regardless of the user's cwd.
func resolveFromRoot(path string) string {
	return resolveFromRootDir(path, findRepoRoot())
}

// resolveFromRootDir is like resolveFromRoot but takes an explicit
// root. Used by commands that accept --root and want to resolve
// other flags against the provided (not auto-discovered) root.
func resolveFromRootDir(path, root string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}

// loadGraphFriendly wraps graph.Load with an actionable error for
// the common "you forgot to run `checks build`" case.
func missingGraphError(graphPath string, err error) error {
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("graph file not found at %s — run `checks build` first", graphPath)
	}
	return fmt.Errorf("loading graph: %w", err)
}

// warnIfGraphStale prints a stderr note when graph is older than any
// of the representative sentinel files. Cheap; runs O(number of
// sentinels) stats.
//
// The sentinels are deliberately coarse: they catch the common "you
// edited a contract/go file and forgot to rebuild" failure mode. We
// don't walk every source file — that's what `checks build` is for.
func warnIfGraphStale(graphPath, root string) {
	info, err := os.Stat(graphPath)
	if err != nil {
		return
	}
	graphMtime := info.ModTime()

	sentinels := []string{
		"go.mod",
		"Cargo.toml",
		"rust/Cargo.toml",
		"packages/contracts-bedrock/foundry.toml",
		"packages/contracts-bedrock/src",
		"packages/contracts-bedrock/test",
	}
	for _, s := range sentinels {
		abs := filepath.Join(root, s)
		st, err := os.Stat(abs)
		if err != nil {
			continue
		}
		gap := st.ModTime().Sub(graphMtime)
		if gap < 30*time.Second {
			// Noise threshold — ignore sub-30s clock drift / same-
			// second file operations that aren't real staleness.
			continue
		}
		fmt.Fprintf(os.Stderr,
			"warning: graph at %s is %s older than %s — consider `checks build`\n",
			graphPath, gap.Round(time.Minute), s)
		return
	}
}
