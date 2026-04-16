package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/ops/checks/policy"
)

// loadPolicy resolves the policy from the embedded baseline, then
// layered files (first the auto-generated learned.yaml if present,
// then the operator override if specified). Missing override files
// are silently ignored so running in a fresh checkout works without
// setup.
//
// The repo root is discovered by walking up from cwd looking for a
// go.mod or ops/checks/ directory. Falls back to cwd.
func loadPolicy(explicitPath string) (*policy.Policy, error) {
	root := findRepoRoot()
	overrides := policy.DefaultOverridePaths(root)
	if explicitPath != "" {
		overrides = append(overrides, explicitPath)
	}
	p, err := policy.Load(overrides...)
	if err != nil {
		return nil, fmt.Errorf("loading policy: %w", err)
	}
	return p, nil
}

func findRepoRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "ops", "checks")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return cwd
		}
		dir = parent
	}
}
