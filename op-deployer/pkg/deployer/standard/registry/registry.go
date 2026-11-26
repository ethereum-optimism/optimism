// Package registry holds the standard values op-deployer reads from the superchain registry
// configs embedded in op-core/superchain/bundle.
//
// They live apart from package standard because the embedded bundle is generated at build time
// and is not part of the published Go module, so any package that imports it does not compile
// outside this repository. Keeping these readers here keeps standard, and the state types built
// on it, importable by downstream Go modules.
package registry

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	opparams "github.com/ethereum-optimism/optimism/op-core/params"
	"github.com/ethereum-optimism/optimism/op-core/superchain"
	"github.com/ethereum-optimism/optimism/op-core/superchain/bundle"
)

// SuperchainFor returns the embedded registry's superchain config for the given L1 chain ID.
func SuperchainFor(chainID uint64) (superchain.Superchain, error) {
	switch chainID {
	case 1:
		return bundle.GetSuperchain("mainnet")
	case 11155111:
		return bundle.GetSuperchain("sepolia")
	default:
		return superchain.Superchain{}, fmt.Errorf("unsupported chain ID: %d", chainID)
	}
}

// maxDefaultHardfork is the latest fork DefaultHardforkSchedule activates at genesis, whatever
// OP Mainnet schedules. A chain with Lagoon scheduled needs an interop dependency set, and
// op-node and kona-node refuse to start a chain the superchain-registry does not know without
// one. The chains op-deployer creates, dev and production alike, are not in the registry, and
// single-chain setups do not supply a dependency set. Delete this cap and defaultHardforkFor
// once they do, so that the default follows OP Mainnet again.
const maxDefaultHardfork = forks.Karst

// DefaultHardforkSchedule is used to determine which hardforks should be activated by default.
// It activates, at genesis, the most recent fork that has an activation timestamp scheduled on
// OP Mainnet, but no later than maxDefaultHardfork.
func DefaultHardforkSchedule() *genesis.UpgradeScheduleDeployConfig {
	sched := &genesis.UpgradeScheduleDeployConfig{}
	sched.ActivateForkAtGenesis(defaultHardfork())

	return sched
}

var defaultHardfork = sync.OnceValue(func() forks.Name {
	chain, err := bundle.GetChain(opparams.OPMainnetChainID)
	if err != nil {
		panic(fmt.Errorf("get op mainnet chain: %w", err))
	}
	chainConfig, err := chain.Config()
	if err != nil {
		panic(fmt.Errorf("load op mainnet chain config: %w", err))
	}
	return defaultHardforkFor(chainConfig.Hardforks)
})

// defaultHardforkFor returns the latest mainline fork scheduled in config, capped at
// maxDefaultHardfork.
func defaultHardforkFor(config superchain.HardforkConfig) forks.Name {
	fork := latestScheduledMainlineFork(config)
	if slices.Index(forks.All, fork) > slices.Index(forks.All, maxDefaultHardfork) {
		return maxDefaultHardfork
	}
	return fork
}

func latestScheduledMainlineFork(config superchain.HardforkConfig) forks.Name {
	configValue := reflect.ValueOf(config)
	mainlineForks := forks.From(forks.Canyon)
	for i := len(mainlineForks) - 1; i >= 0; i-- {
		fork := mainlineForks[i]
		// HardforkConfig fields follow the <ForkName>Time convention. Looking them up
		// from forks.All keeps this selection current when a new mainline fork is added.
		field := configValue.FieldByNameFunc(func(name string) bool {
			return strings.EqualFold(strings.TrimSuffix(name, "Time"), string(fork))
		})
		if !field.IsValid() {
			panic(fmt.Sprintf("mainline fork %q is missing from superchain.HardforkConfig", fork))
		}
		activationTime, ok := field.Interface().(*uint64)
		if !ok {
			panic(fmt.Sprintf("superchain.HardforkConfig field for %q must be *uint64", fork))
		}
		if activationTime != nil {
			return fork
		}
	}

	// Regolith is active at genesis for every registry-backed rollup config, but its
	// activation is implicit and is therefore not represented in HardforkConfig.
	return forks.Regolith
}
