package interop

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/interopgen/config"
	coredepset "github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-program/client/boot"
	"github.com/ethereum-optimism/optimism/op-program/client/l1"
)

// getFullConfig creates a new config.FullConfigSet using the boot-info config sources,
// and a L1 preimage oracle to load the L1 block info of the rollup anchor blocks.
func getFullConfig(c boot.ConfigSource, l1PreimageOracle l1.Oracle, depSet coredepset.DependencySet) (config.FullConfigSet, error) {
	configs := make(config.StaticRollupConfigSet)
	for _, chID := range depSet.Chains() {
		rollupCfg, err := c.RollupConfig(chID)
		if err != nil {
			return nil, err
		}
		l1Header := l1PreimageOracle.HeaderByBlockHash(rollupCfg.Genesis.L1.Hash)
		configs[chID] = config.StaticRollupConfigFromRollupConfig(rollupCfg, l1Header.Time())
	}
	return config.NewFullConfigSetMerged(configs, depSet)
}
