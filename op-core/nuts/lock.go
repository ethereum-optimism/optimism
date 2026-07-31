package nuts

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	opservice "github.com/ethereum-optimism/optimism/op-service"
)

// ForkLockEntry represents a single fork's entry in fork_lock.toml.
type ForkLockEntry struct {
	Bundle string `toml:"bundle"`
	Hash   string `toml:"hash"`
	Commit string `toml:"commit"`
}

// ForkLock is the full contents of fork_lock.toml, keyed by fork name.
type ForkLock map[string]ForkLockEntry

// LockFilePath returns the absolute path to fork_lock.toml relative to the given directory.
func LockFilePath(dir string) (string, error) {
	root, err := opservice.FindMonorepoRoot(dir)
	if err != nil {
		return "", fmt.Errorf("finding monorepo root: %w", err)
	}
	return filepath.Join(root, "op-core", "nuts", "fork_lock.toml"), nil
}

// ReadLockFile reads and parses fork_lock.toml from the monorepo root.
func ReadLockFile(dir string) (ForkLock, string, error) {
	lockPath, err := LockFilePath(dir)
	if err != nil {
		return nil, "", err
	}
	var locks ForkLock
	if _, err := toml.DecodeFile(lockPath, &locks); err != nil {
		return nil, "", fmt.Errorf("reading fork lock file: %w", err)
	}
	return locks, lockPath, nil
}

// WriteLockFile writes fork_lock.toml back to disk with a header comment.
func WriteLockFile(lockPath string, locks ForkLock) error {
	f, err := os.Create(lockPath)
	if err != nil {
		return fmt.Errorf("opening fork lock file for writing: %w", err)
	}
	defer f.Close()

	_, err = fmt.Fprint(f, `# To update a fork's bundle, run: just nut-snapshot-for <fork>
# Contract changes must be merged to develop before snapshotting.

`)
	if err != nil {
		return err
	}

	// Encode entries in chronological fork order so that adding a new fork
	// doesn't reshuffle older entries.
	enc := toml.NewEncoder(f)
	written := make(map[string]bool, len(locks))
	for _, fork := range forks.All {
		entry, ok := locks[string(fork)]
		if !ok {
			continue
		}
		if err := enc.Encode(map[string]ForkLockEntry{string(fork): entry}); err != nil {
			return fmt.Errorf("writing fork lock file: %w", err)
		}
		written[string(fork)] = true
	}
	// Write any unknown fork names last (alphabetical fallback).
	for name, entry := range locks {
		if written[name] {
			continue
		}
		if err := enc.Encode(map[string]ForkLockEntry{name: entry}); err != nil {
			return fmt.Errorf("writing fork lock file: %w", err)
		}
	}

	_, err = fmt.Fprint(f, `
# REVIEWER NOTE: Changes to this file affect which NUT bundles are embedded
# into op-node and kona-node for hardfork activations. Review carefully.
`)
	if err != nil {
		return err
	}
	return nil
}
