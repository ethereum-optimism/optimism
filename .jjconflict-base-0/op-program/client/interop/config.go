package interop

import (
	"github.com/ethereum-optimism/optimism/op-program/client/boot"
	"github.com/ethereum-optimism/optimism/op-program/client/l1"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
)

// getFullConfig creates a new depset.FullConfigSet using the boot-info config sources,
// and a L1 preimage oracle to load the L1 block info of the rollup anchor blocks.
func getFullConfig(c boot.ConfigSource, l1PreimageOracle l1.Oracle, depSet depset.DependencySet) (depset.FullConfigSet, error) {
	configs := make(depset.StaticRollupConfigSet)
	for _, chID := range depSet.Chains() {
		rollupCfg, err := c.RollupConfig(chID)
		if err != nil {
			return nil, err
		}
		l1Header := l1PreimageOracle.HeaderByBlockHash(rollupCfg.Genesis.L1.Hash)
		configs[chID] = depset.StaticRollupConfigFromRollupConfig(rollupCfg, l1Header.Time())
	}
	return depset.NewFullConfigSetMerged(configs, depSet)
}
