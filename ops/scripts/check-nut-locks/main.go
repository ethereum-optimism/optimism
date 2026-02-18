package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	opservice "github.com/ethereum-optimism/optimism/op-service"
)

type forkLockEntry struct {
	Bundle string `toml:"bundle"`
	Hash   string `toml:"hash"`
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

	lockPath := filepath.Join(root, "op-core", "nuts", "fork_lock.toml")
	var locks map[string]forkLockEntry
	if _, err := toml.DecodeFile(lockPath, &locks); err != nil {
		return fmt.Errorf("reading fork lock file: %w", err)
	}

	for fork, entry := range locks {
		bundlePath := filepath.Join(root, entry.Bundle)
		content, err := os.ReadFile(bundlePath)
		if err != nil {
			return fmt.Errorf("fork %s: reading bundle %s: %w", fork, entry.Bundle, err)
		}

		hash := sha256.Sum256(content)
		actual := "sha256:" + hex.EncodeToString(hash[:])

		locked := strings.TrimSpace(entry.Hash)
		if actual != locked {
			return fmt.Errorf(
				"bundle hash mismatch for fork %s: locked=%s actual=%s. "+
					"If this change is intentional, update the hash in op-core/nuts/fork_lock.toml",
				fork, locked, actual,
			)
		}

		fmt.Printf("fork %s: bundle hash OK\n", fork)
	}

	return nil
}
