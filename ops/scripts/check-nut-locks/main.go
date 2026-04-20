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
	"rust/kona/crates/protocol/hardforks/bundles/*_nut_bundle.json",
}

// konaBundleMirror returns the kona-hardforks crate path that mirrors a
// locked op-core bundle entry.
func konaBundleMirror(bundleRel string) string {
	return filepath.Join(
		"rust", "kona", "crates", "protocol", "hardforks", "bundles",
		filepath.Base(bundleRel),
	)
}

// checkKonaMirror verifies that the kona-hardforks crate's copy of the bundle
// is byte-identical to the canonical op-core copy. Both copies are written by
// `ops/scripts/nut-snapshot-for`; this guard catches hand-edits that would
// cause kona and op-node to execute different upgrade transactions.
func checkKonaMirror(root, fork string, canonical []byte, bundleRel string) error {
	mirrorRel := konaBundleMirror(bundleRel)
	mirrorPath := filepath.Join(root, mirrorRel)
	mirror, err := os.ReadFile(mirrorPath)
	if err != nil {
		return fmt.Errorf(
			"fork %s: reading kona bundle mirror %s: %w. "+
				"Run 'just nut-snapshot-for %s' to regenerate both copies",
			fork, mirrorRel, err, fork,
		)
	}
	canonicalHash := sha256.Sum256(canonical)
	mirrorHash := sha256.Sum256(mirror)
	if canonicalHash != mirrorHash {
		return fmt.Errorf(
			"fork %s: kona bundle mirror %s differs from canonical %s. "+
				"Re-run 'just nut-snapshot-for %s' to resync",
			fork, mirrorRel, bundleRel, fork,
		)
	}
	return nil
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
		// The kona-hardforks mirror is an implicit dependent of each op-core
		// entry; mark it locked so the reverse-check below doesn't flag it.
		lockedBundles[konaBundleMirror(entry.Bundle)] = true

		bundlePath := filepath.Join(root, entry.Bundle)
		content, err := os.ReadFile(bundlePath)
		if err != nil {
			return fmt.Errorf("fork %s: reading bundle %s: %w", fork, entry.Bundle, err)
		}

		if err := validateEntry(fork, entry, content); err != nil {
			return err
		}

		if err := checkKonaMirror(root, fork, content, entry.Bundle); err != nil {
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
