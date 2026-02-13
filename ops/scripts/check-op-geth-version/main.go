package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"golang.org/x/mod/modfile"

	opservice "github.com/ethereum-optimism/optimism/op-service"
)

const (
	goEthereumPath = "github.com/ethereum/go-ethereum"
	opGethPath     = "github.com/ethereum-optimism/op-geth"
)

// The op-geth minor version encodes the upstream geth version as exactly 6 digits:
//   - 2 digits for geth major version (zero-padded)
//   - 2 digits for geth minor version (zero-padded)
//   - 2 digits for geth patch version (zero-padded)
//
// Examples:
//   - v1.101407.0      -> geth v1.14.7
//   - v1.101605.0-rc.2 -> geth v1.16.5
var opGethVersionPattern = regexp.MustCompile(`^v\d+\.\d{6}\.\d+(-rc\.\d+)?$`)

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

	goModPath := filepath.Join(root, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return fmt.Errorf("reading go.mod: %w", err)
	}

	modFile, err := modfile.Parse(goModPath, content, nil)
	if err != nil {
		return fmt.Errorf("parsing go.mod: %w", err)
	}

	// Find the replace directive for go-ethereum -> op-geth
	var opGethVersion string
	for _, rep := range modFile.Replace {
		if rep.Old.Path == goEthereumPath {
			if rep.New.Path != opGethPath {
				return fmt.Errorf("go-ethereum replacement must point to %s, got %s", opGethPath, rep.New.Path)
			}
			opGethVersion = rep.New.Version
			break
		}
	}

	if opGethVersion == "" {
		return fmt.Errorf("no replace directive found for %s", goEthereumPath)
	}

	if !opGethVersionPattern.MatchString(opGethVersion) {
		return fmt.Errorf("invalid op-geth version %q", opGethVersion)
	}

	return nil
}
