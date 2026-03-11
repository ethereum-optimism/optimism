package main

import (
	"bytes"
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
		fmt.Fprintf(os.Stderr, "usage: verify-nuts <fork>\n")
		os.Exit(1)
	}

	fork := forks.Name(os.Args[1])
	if err := run(fork); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
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

	locks, _, err := nuts.ReadLockFile(".")
	if err != nil {
		return err
	}

	entry, ok := locks[string(fork)]
	if !ok {
		return fmt.Errorf("no entry for fork %q in fork_lock.toml", fork)
	}

	// Step 1: Verify bundle file exists and hash matches.
	bundlePath := filepath.Join(root, entry.Bundle)
	bundleContent, err := os.ReadFile(bundlePath)
	if err != nil {
		return fmt.Errorf("reading bundle %s: %w", entry.Bundle, err)
	}

	hash := sha256.Sum256(bundleContent)
	actual := "sha256:" + hex.EncodeToString(hash[:])
	locked := strings.TrimSpace(entry.Hash)

	if actual != locked {
		return fmt.Errorf("hash mismatch: locked=%s actual=%s", locked, actual)
	}
	fmt.Printf("PASS: bundle hash matches lock (%s)\n", actual)

	// Step 2: If commit is recorded, verify the bundle was correctly built from that commit.
	if entry.Commit == "" {
		fmt.Println("SKIP: no commit recorded, cannot verify bundle provenance")
		return nil
	}

	fmt.Printf("Verifying bundle provenance from commit %s...\n", entry.Commit[:12])
	if err := verifyFromCommit(root, entry); err != nil {
		return fmt.Errorf("provenance verification: %w", err)
	}

	fmt.Println("PASS: regenerated bundle matches committed bundle")
	return nil
}

// verifyFromCommit creates a temporary worktree at the recorded commit,
// regenerates the NUT bundle, and compares it against the locked bundle.
func verifyFromCommit(root string, entry nuts.ForkLockEntry) error {
	worktreeDir, err := os.MkdirTemp("", "verify-nuts-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(worktreeDir)

	// Create worktree at the recorded commit.
	addCmd := exec.Command("git", "worktree", "add", "--detach", worktreeDir, entry.Commit)
	addCmd.Dir = root
	addCmd.Stderr = os.Stderr
	if err := addCmd.Run(); err != nil {
		return fmt.Errorf("creating worktree at %s: %w", entry.Commit[:12], err)
	}
	defer func() {
		removeCmd := exec.Command("git", "worktree", "remove", "--force", worktreeDir)
		removeCmd.Dir = root
		_ = removeCmd.Run()
	}()

	// Generate NUT bundle in the worktree.
	contractsDir := filepath.Join(worktreeDir, "packages", "contracts-bedrock")
	genCmd := exec.Command("just", "generate-nut-bundle")
	genCmd.Dir = contractsDir
	genCmd.Stdout = os.Stdout
	genCmd.Stderr = os.Stderr
	if err := genCmd.Run(); err != nil {
		return fmt.Errorf("generating NUT bundle at commit %s: %w", entry.Commit[:12], err)
	}

	// Read the regenerated bundle.
	regenPath := filepath.Join(contractsDir, "snapshots", "upgrades", "current-upgrade-bundle.json")
	regenContent, err := os.ReadFile(regenPath)
	if err != nil {
		return fmt.Errorf("reading regenerated bundle: %w", err)
	}

	// Read the committed bundle.
	bundlePath := filepath.Join(root, entry.Bundle)
	committedContent, err := os.ReadFile(bundlePath)
	if err != nil {
		return fmt.Errorf("reading committed bundle: %w", err)
	}

	if !bytes.Equal(regenContent, committedContent) {
		return fmt.Errorf("regenerated bundle at commit %s differs from committed bundle %s",
			entry.Commit[:12], entry.Bundle)
	}

	return nil
}
