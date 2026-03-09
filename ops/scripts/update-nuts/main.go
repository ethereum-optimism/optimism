package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	opservice "github.com/ethereum-optimism/optimism/op-service"
)

type forkLockEntry struct {
	Bundle string `toml:"bundle"`
	Hash   string `toml:"hash"`
	Commit string `toml:"commit,omitempty"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: update-nuts <fork>\n")
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

	// Generate fresh bundle from contracts.
	fmt.Println("Generating NUT bundle...")
	contractsDir := filepath.Join(root, "packages", "contracts-bedrock")
	cmd := exec.Command("just", "generate-nut-bundle")
	cmd.Dir = contractsDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("generating NUT bundle: %w", err)
	}

	// Copy current-upgrade-bundle.json → <fork>_nut_bundle.json.
	srcPath := filepath.Join(contractsDir, "snapshots", "upgrades", "current-upgrade-bundle.json")
	content, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("reading generated bundle: %w", err)
	}

	bundleRel := filepath.Join("op-node", "rollup", "derive", string(fork)+"_nut_bundle.json")
	dstPath := filepath.Join(root, bundleRel)
	if err := os.WriteFile(dstPath, content, 0644); err != nil {
		return fmt.Errorf("writing bundle to %s: %w", bundleRel, err)
	}
	fmt.Printf("Copied bundle to %s\n", bundleRel)

	// Compute sha256 of the bundle.
	hash := sha256.Sum256(content)
	hashStr := "sha256:" + hex.EncodeToString(hash[:])

	// Get current git commit.
	commitCmd := exec.Command("git", "rev-parse", "HEAD")
	commitCmd.Dir = root
	commitOut, err := commitCmd.Output()
	if err != nil {
		return fmt.Errorf("getting git commit: %w", err)
	}
	commit := strings.TrimSpace(string(commitOut))

	// Read existing fork_lock.toml, update the entry, write back.
	lockPath := filepath.Join(root, "op-core", "nuts", "fork_lock.toml")
	var locks map[string]forkLockEntry
	if _, err := toml.DecodeFile(lockPath, &locks); err != nil {
		return fmt.Errorf("reading fork lock file: %w", err)
	}
	if locks == nil {
		locks = make(map[string]forkLockEntry)
	}

	locks[string(fork)] = forkLockEntry{
		Bundle: bundleRel,
		Hash:   hashStr,
		Commit: commit,
	}

	f, err := os.Create(lockPath)
	if err != nil {
		return fmt.Errorf("opening fork lock file for writing: %w", err)
	}
	defer f.Close()

	// Write header comment.
	if _, err := fmt.Fprintln(f, "# NUT Bundle Fork Lock"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(f, "# To update a locked bundle, run: just update-nuts <fork>"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(f); err != nil {
		return err
	}

	enc := toml.NewEncoder(f)
	if err := enc.Encode(locks); err != nil {
		return fmt.Errorf("writing fork lock file: %w", err)
	}

	fmt.Printf("Updated fork_lock.toml: fork=%s hash=%s commit=%s\n", fork, hashStr, commit[:12])
	return nil
}
