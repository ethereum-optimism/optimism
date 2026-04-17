package main

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/ops/checks/policy"
)

// loadPolicy resolves the policy from the embedded baseline, then
// layered files (first the auto-generated learned.yaml if present,
// then the operator override if specified). Missing override files
// are silently ignored so running in a fresh checkout works without
// setup.
//
// Repo root discovery is shared with other commands via findRepoRoot
// (see paths.go).
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
