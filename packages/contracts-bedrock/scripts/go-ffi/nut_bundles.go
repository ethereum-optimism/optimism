package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-core/nuts"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

// NUTBundleEncoded is the ABI-encoded representation consumed by Solidity tests.
type NUTBundleEncoded struct {
	Fork string
	Path string
}

// GetNUTBundles returns ABI-encoded committed NUT bundles in chronological fork order.
// Called via FFI from Solidity tests.
// Usage: go-ffi nut-bundles
// Returns: ABI-encoded array of (string fork, string path)
func GetNUTBundles() {
	bundles, err := orderedNUTBundles(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get NUT bundles: %v\n", err)
		os.Exit(1)
	}

	nutBundleType, err := abi.NewType("tuple[]", "", []abi.ArgumentMarshaling{
		{Name: "fork", Type: "string"},
		{Name: "path", Type: "string"},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to define ABI type: %v\n", err)
		os.Exit(1)
	}

	args := abi.Arguments{{Type: nutBundleType}}
	result, err := args.Pack(bundles)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to ABI encode: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("0x%x", result)
}

func orderedNUTBundles(startDir string) ([]NUTBundleEncoded, error) {
	root, err := opservice.FindMonorepoRoot(startDir)
	if err != nil {
		return nil, fmt.Errorf("finding monorepo root: %w", err)
	}

	locks, _, err := nuts.ReadLockFile(startDir)
	if err != nil {
		return nil, err
	}

	return orderedNUTBundlesFromLocks(locks, root)
}

func orderedNUTBundlesFromLocks(locks nuts.ForkLock, root string) ([]NUTBundleEncoded, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolving monorepo root: %w", err)
	}

	for forkName := range locks {
		if !forks.IsValid(forks.Name(forkName)) {
			return nil, fmt.Errorf("locked fork %q is not in forks.All", forkName)
		}
	}

	// Output order is driven by [forks.From](Karst), not TOML or Go map
	// iteration order. Pre-Karst lock entries are validated above but not
	// emitted because this NUT-bundle staging path starts at Karst.
	bundles := make([]NUTBundleEncoded, 0, len(locks))
	for _, fork := range forks.From(forks.Karst) {
		entry, ok := locks[string(fork)]
		if !ok {
			continue
		}
		path, err := contractsBedrockReadablePath(root, entry.Bundle)
		if err != nil {
			return nil, fmt.Errorf("resolving %s NUT bundle: %w", fork, err)
		}
		bundles = append(bundles, NUTBundleEncoded{
			Fork: string(fork),
			Path: path,
		})
	}

	return bundles, nil
}

func contractsBedrockReadablePath(root string, bundlePath string) (string, error) {
	if bundlePath == "" {
		return "", fmt.Errorf("bundle path is empty")
	}

	var absPath string
	if filepath.IsAbs(bundlePath) {
		absPath = filepath.Clean(bundlePath)
	} else {
		absPath = filepath.Join(root, bundlePath)
	}

	relToRoot, err := filepath.Rel(root, absPath)
	if err != nil {
		return "", fmt.Errorf("checking monorepo-relative path: %w", err)
	}
	if relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) || filepath.IsAbs(relToRoot) {
		return "", fmt.Errorf("bundle path %q escapes monorepo root", bundlePath)
	}

	if info, err := os.Stat(absPath); err != nil {
		return "", fmt.Errorf("checking bundle file: %w", err)
	} else if info.IsDir() {
		return "", fmt.Errorf("bundle path %q is a directory", bundlePath)
	}

	contractsBedrockDir := filepath.Join(root, "packages", "contracts-bedrock")
	relToContractsBedrock, err := filepath.Rel(contractsBedrockDir, absPath)
	if err != nil {
		return "", fmt.Errorf("making contracts-bedrock-relative path: %w", err)
	}
	return filepath.ToSlash(relToContractsBedrock), nil
}
