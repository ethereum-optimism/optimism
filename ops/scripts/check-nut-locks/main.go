package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ethereum-optimism/optimism/op-core/nuts"
	opservice "github.com/ethereum-optimism/optimism/op-service"
)

// nutBundleGlobs are the locations where NUT bundle JSON files may live.
// Update this list when adding new bundle locations.
var nutBundleGlobs = []string{
	"op-core/nuts/bundles/*_nut_bundle.json",
}

// checkAllBundlesLocked searches known paths for *_nut_bundle.json files and
// verifies each has a corresponding entry in fork_lock.toml.
func checkAllBundlesLocked(root string, lockedBundles map[string]bool) error {
	for _, pattern := range nutBundleGlobs {
		matches, err := filepath.Glob(filepath.Join(root, pattern))
		if err != nil {
			return fmt.Errorf("globbing %s: %w", pattern, err)
		}
		for _, match := range matches {
			rel, err := filepath.Rel(root, match)
			if err != nil {
				return err
			}
			if !lockedBundles[rel] {
				return fmt.Errorf(
					"NUT bundle %s has no entry in op-core/nuts/fork_lock.toml",
					rel,
				)
			}
		}
	}
	return nil
}

// validateEntry checks a single fork lock entry against its bundle file content.
func validateEntry(fork string, entry nuts.ForkLockEntry, bundleContent []byte) error {
	hash := sha256.Sum256(bundleContent)
	actual := "sha256:" + hex.EncodeToString(hash[:])

	expectedHash := strings.TrimSpace(entry.Hash)
	if actual != expectedHash {
		return fmt.Errorf(
			"bundle hash mismatch for fork %s: expected=%s actual=%s. "+
				"If this change is intentional, update the hash in op-core/nuts/fork_lock.toml",
			fork, expectedHash, actual,
		)
	}

	if entry.Commit == "" {
		return fmt.Errorf("fork %s has no commit recorded; "+
			"run 'just nut-snapshot-for %s' to populate the commit field", fork, fork)
	}

	return nil
}

// checkCommitAncestry verifies that a commit is an ancestor of origin/develop.
func checkCommitAncestry(root, fork string, commit string) error {
	// Note: if you are here because you want to enable a bundle for a fork to be generated from a
	// commit on a branch other than `develop`, you will
	// 1. need to add a special case to this function
	// 2. need to cherry pick the PR to the `develop` branch.
	// See the "Regarding L2 contract releases" section in release-process.md for more information..
	cmd := exec.Command("git", "merge-base", "--is-ancestor", commit, "origin/develop")
	cmd.Dir = root
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("fork %s: commit %s is not an ancestor of origin/develop", fork, commit[:12])
	}
	return nil
}

func main() {
	if err := run("."); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(dir string) error {
	root, err := opservice.FindMonorepoRoot(dir)
	if err != nil {
		return fmt.Errorf("finding monorepo root: %w", err)
	}

	locks, _, err := nuts.ReadLockFile(dir)
	if err != nil {
		return err
	}

	lockedBundles := make(map[string]bool)
	for fork, entry := range locks {
		lockedBundles[entry.Bundle] = true

		bundlePath := filepath.Join(root, entry.Bundle)
		content, err := os.ReadFile(bundlePath)
		if err != nil {
			return fmt.Errorf("fork %s: reading bundle %s: %w", fork, entry.Bundle, err)
		}

		if err := validateEntry(fork, entry, content); err != nil {
			return err
		}

		if err := checkCommitAncestry(root, fork, entry.Commit); err != nil {
			return err
		}

		fmt.Printf("fork %s: bundle hash OK\n", fork)
	}

	// Reverse check: verify all NUT bundle JSONs have a lock entry
	if err := checkAllBundlesLocked(root, lockedBundles); err != nil {
		return err
	}

	return nil
}
