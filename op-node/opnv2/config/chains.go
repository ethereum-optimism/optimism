package config

import (
	"errors"
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/superchain"

	"github.com/ethereum-optimism/optimism/op-node/opnv2/flags"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
)

func ChainConfigsFromCLI(ctx *cli.Context) (depset.DependencySetSource, depset.RollupConfigSetSourceV2, error) {
	if ctx.IsSet(flags.NetworkFlag.Name) {
		return superchainSource(ctx)
	} else if ctx.IsSet(flags.RollupConfigFlag.Name) {
		return legacySource(ctx)
	} else if ctx.IsSet(flags.RollupConfigPathsFlag.Name) && ctx.IsSet(flags.DependencySetFlag.Name) {
		return dependencySetPathSource(ctx), rollupConfigPathsSource(ctx), nil
	} else {
		return nil, nil, fmt.Errorf("either %s, or %s, or a combination of %s and %s must be set",
			flags.NetworkFlag.Name, flags.RollupConfigFlag.Name,
			flags.DependencySetFlag.Name, flags.RollupConfigPathsFlag.Name)
	}
}

func legacySource(ctx *cli.Context) (depset.DependencySetSource, depset.RollupConfigSetSourceV2, error) {
	// Legacy flag from op-node v1.
	// We used an implied dependency set of 1, and a single rollup config.
	rollupCfg, err := depset.LoadRollupCfg(ctx.String(flags.RollupConfigFlag.Name))
	if err != nil {
		return nil, nil, err
	}
	chainID := eth.ChainIDFromBig(rollupCfg.L2ChainID)
	cfgs := depset.StaticRollupConfigSetV2{
		chainID: rollupCfg,
	}
	if rollupCfg.InteropTime == nil {
		// dependency set is not required if interop is not scheduled, fall back to a default single-chain depset.
		depSet, err := depset.NewStaticConfigDependencySet(map[eth.ChainID]*depset.StaticConfigDependency{
			chainID: {},
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create implied dependency set: %w", err)
		}
		return depSet, cfgs, nil
	}
	if !ctx.IsSet(flags.DependencySetFlag.Name) {
		return nil, nil, errors.New("interop fork is scheduled in legacy rollup config flag, must supply dependency-set path")
	}
	return dependencySetPathSource(ctx), cfgs, nil
}

func superchainSource(ctx *cli.Context) (depset.DependencySetSource, depset.RollupConfigSetSourceV2, error) {
	network := ctx.String(flags.NetworkFlag.Name)
	networkChainID, err := superchain.ChainIDByName(network)
	if err != nil {
		return nil, nil, err
	}
	// Use the dependency set from the first chain.
	// superchain-registry has checks to ensure consistency for all chains in the same set
	depSet, err := depset.FromRegistry(eth.ChainIDFromUInt64(networkChainID))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load dependency set for network %s: %w", network, err)
	}
	cfgs := make(depset.StaticRollupConfigSetV2)
	for _, chainID := range depSet.Chains() {
		chainIDU64, ok := chainID.Uint64()
		if !ok {
			return nil, nil, fmt.Errorf("unexpected non-uint64 chainID in registry: %s", chainID)
		}
		rollupCfg, err := rollup.LoadOPStackRollupConfig(chainIDU64)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load rollup config for network %s: %w", network, err)
		}
		cfgs[chainID] = rollupCfg
	}
	return depSet, cfgs, nil
}

func dependencySetPathSource(ctx *cli.Context) depset.DependencySetSource {
	return &depset.JSONDependencySetLoader{
		Path: ctx.Path(flags.DependencySetFlag.Name),
	}
}

func rollupConfigPathsSource(ctx *cli.Context) depset.RollupConfigSetSourceV2 {
	return &depset.JSONRollupConfigsLoaderV2{
		PathPattern: ctx.String(flags.RollupConfigPathsFlag.Name),
	}
}
