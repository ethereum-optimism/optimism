package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-core/nuts"
	opservice "github.com/ethereum-optimism/optimism/op-service"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: nut-snapshot-for <fork>\n")
		os.Exit(1)
	}

	fork := forks.Name(os.Args[1])
	if err := run(fork); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(fork forks.Name) error {
	if !forks.IsValid(fork) {
		return fmt.Errorf("unknown fork %q; valid forks: %v", fork, forks.All)
	}

	root, err := opservice.FindMonorepoRoot(".")
	if err != nil {
		return fmt.Errorf("finding monorepo root: %w", err)
	}

	// Copy current-upgrade-bundle.json → <fork>_nut_bundle.json.
	// The caller is responsible for running `just generate-nut-bundle` first if needed.
	srcPath := filepath.Join(root, "packages", "contracts-bedrock", "snapshots", "upgrades", "current-upgrade-bundle.json")
	content, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("reading bundle (run 'just generate-nut-bundle' in packages/contracts-bedrock/ first): %w", err)
	}

	bundleRel := filepath.Join("op-core", "nuts", "bundles", string(fork)+"_nut_bundle.json")
	dstPath := filepath.Join(root, bundleRel)
	if err := os.WriteFile(dstPath, content, 0600); err != nil {
		return fmt.Errorf("writing bundle to %s: %w", bundleRel, err)
	}
	fmt.Printf("Copied bundle to %s\n", bundleRel)

	// Compute sha256 of the bundle.
	hash := sha256.Sum256(content)
	hashStr := "sha256:" + hex.EncodeToString(hash[:])

	// Record the merge-base with develop (not HEAD) so the commit survives
	// squash-merge. Contracts must be merged to develop before snapshotting.
	commitCmd := exec.Command("git", "merge-base", "HEAD", "origin/develop")
	commitCmd.Dir = root
	commitOut, err := commitCmd.Output()
	if err != nil {
		return fmt.Errorf("finding merge-base with origin/develop (fetch first?): %w", err)
	}
	commit := strings.TrimSpace(string(commitOut))

	// Read existing fork_lock.toml, update the entry, write back.
	locks, lockPath, err := nuts.ReadLockFile(".")
	if err != nil {
		return err
	}

	locks[string(fork)] = nuts.ForkLockEntry{
		Bundle: bundleRel,
		Hash:   hashStr,
		Commit: commit,
	}

	if err := nuts.WriteLockFile(lockPath, locks); err != nil {
		return err
	}

	fmt.Printf("Updated fork_lock.toml: fork=%s hash=%s commit=%s\n", fork, hashStr, commit[:12])
	return nil
}
