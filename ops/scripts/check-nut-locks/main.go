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
	"op-node/rollup/derive/*_nut_bundle.json",
	"op-core/nuts/*_nut_bundle.json",
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

// checkLockedForksUnchanged verifies that locked forks have not had their
// hash or bundle path changed compared to the base branch.
func checkLockedForksUnchanged(root string, locks nuts.ForkLock) error {
	baseLocks, err := readBaseBranchLockFile(root)
	if err != nil {
		return fmt.Errorf("reading base branch fork_lock.toml: %w", err)
	}
	for fork, entry := range locks {
		baseEntry, existed := baseLocks[fork]
		if !existed {
			continue // new fork not in base branch, nothing to compare
		}
		if !baseEntry.Locked {
			continue // wasn't locked on base, changes are fine
		}
		// Fork was locked on base branch — hash and bundle path must not change
		if entry.Hash != baseEntry.Hash {
			return fmt.Errorf("fork %s is locked but its hash changed: base=%s current=%s; "+
				"locked forks cannot be updated without first unlocking", fork, baseEntry.Hash, entry.Hash)
		}
		if entry.Bundle != baseEntry.Bundle {
			return fmt.Errorf("fork %s is locked but its bundle path changed: base=%s current=%s",
				fork, baseEntry.Bundle, entry.Bundle)
		}
	}
	return nil
}

// readBaseBranchLockFile reads fork_lock.toml from the base branch via git show.
func readBaseBranchLockFile(root string) (nuts.ForkLock, error) {
	for _, base := range []string{"origin/develop", "origin/main"} {
		cmd := exec.Command("git", "show", base+":op-core/nuts/fork_lock.toml")
		cmd.Dir = root
		out, err := cmd.Output()
		if err != nil {
			continue
		}
		return nuts.ParseLockFile(out)
	}
	return nil, fmt.Errorf("fork_lock.toml not found on base branch")
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

		hash := sha256.Sum256(content)
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
				"run 'just update-nuts %s' to populate the commit field", fork, fork)
		}

		fmt.Printf("fork %s: bundle hash OK\n", fork)
	}

	// Check locked forks haven't changed vs base branch
	if err := checkLockedForksUnchanged(root, locks); err != nil {
		return err
	}

	// Reverse check: verify all NUT bundle JSONs have a lock entry
	if err := checkAllBundlesLocked(root, lockedBundles); err != nil {
		return err
	}

	return nil
}
