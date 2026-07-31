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

// bundleGenerator generates a NUT bundle in the given contracts directory.
type bundleGenerator func(contractsDir string) error

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: nut-provenance-verify <fork>\n")
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
	expectedHash := strings.TrimSpace(entry.Hash)

	if actual != expectedHash {
		return fmt.Errorf("hash mismatch: expected=%s actual=%s", expectedHash, actual)
	}
	fmt.Printf("PASS: bundle hash matches lock (%s)\n", actual)

	// Step 2: Verify the bundle was correctly built from the recorded commit.
	if entry.Commit == "" {
		return fmt.Errorf("fork %q has no commit recorded; cannot verify provenance", fork)
	}

	fmt.Printf("Verifying bundle provenance from commit %s...\n", entry.Commit[:12])
	if err := verifyFromCommit(root, fork, entry, func(contractsDir string) error {
		cmd := exec.Command("just", "generate-nut-bundle")
		cmd.Dir = contractsDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}); err != nil {
		return fmt.Errorf("provenance verification: %w", err)
	}

	fmt.Println("PASS: regenerated bundle matches committed bundle")
	return nil
}

// verifyFromCommit creates a temporary worktree at the recorded commit,
// regenerates the NUT bundle, and compares it against the locked bundle.
func verifyFromCommit(root string, fork forks.Name, entry nuts.ForkLockEntry, generate bundleGenerator) error {
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
	if err := generate(contractsDir); err != nil {
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
		return fmt.Errorf(
			"the bundle regenerated from commit %s does not match the committed bundle at %s for the %s fork . See op-core/nuts/README.md for details on how to lock a new bundle.",
			entry.Commit[:12], entry.Bundle, fork,
		)
	}

	return nil
}
