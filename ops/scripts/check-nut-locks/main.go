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

	metadata, err := nuts.ReadBundleMetadata(bundleContent)
	if err != nil {
		return fmt.Errorf("fork %s: %w", fork, err)
	}
	if err := metadata.Validate(); err != nil {
		return fmt.Errorf("fork %s: %w", fork, err)
	}
	// The lock's extra_gas is the reviewer-visible mirror of the bundle's own metadata.extraGas.
	// A disagreement means one of the two was hand-edited; the bundle is authoritative.
	if entry.ExtraGas != metadata.ExtraGas {
		return fmt.Errorf(
			"fork %s: extra_gas in fork_lock.toml (%d) does not match the bundle's metadata.extraGas (%d); "+
				"run 'just nut-snapshot-for %s' to re-derive it from the bundle",
			fork, entry.ExtraGas, metadata.ExtraGas, fork,
		)
	}

	return nil
}

// checkPreForkState verifies the locked fork's own pre-fork state artifact exists
// — op-core/nuts/state/<fork>_state.json, the state as of this fork. Locking a
// fork's bundle should also produce this state (post-activation) so the NEXT
// fork's activation test can boot from it. Catching a forgotten one here keeps
// the chain of states complete, and cheaply (the proofs suite needs kona-host).
func checkPreForkState(root, fork string) error {
	rel := filepath.Join("op-core", "nuts", "state", fork+"_state.json")
	if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
		return fmt.Errorf(
			"fork %s is locked but its pre-fork state %s is missing — generate and commit it so "+
				"the next fork can boot from it (see op-core/nuts/state/README.md)", fork, rel)
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

		if err := checkPreForkState(root, fork); err != nil {
			return err
		}

		fmt.Printf("fork %s: bundle hash + pre-fork state OK\n", fork)
	}

	// Reverse check: verify all NUT bundle JSONs have a lock entry
	if err := checkAllBundlesLocked(root, lockedBundles); err != nil {
		return err
	}

	return nil
}
