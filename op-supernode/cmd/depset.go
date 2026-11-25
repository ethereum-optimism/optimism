package main

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/config"
)

// applySupernodeDependencySet loads the supernode-level dependency set (if a
// path is configured) and assigns it to every virtual node config, replacing
// any value previously derived from per-VN flags or the registry fallback.
// The dependency set is a global property of the supernode and must be the
// same for all chains it manages.
func applySupernodeDependencySet(cfg *config.CLIConfig, vnCfgs map[eth.ChainID]*opnodecfg.Config) error {
	if cfg.DependencySetPath == "" {
		return nil
	}
	loader := &depset.JSONDependencySetLoader{Path: cfg.DependencySetPath}
	ds, err := loader.LoadDependencySet()
	if err != nil {
		return fmt.Errorf("failed to load supernode dependency set %q: %w", cfg.DependencySetPath, err)
	}
	for _, vnCfg := range vnCfgs {
		vnCfg.DependencySet = ds
	}
	return nil
}
