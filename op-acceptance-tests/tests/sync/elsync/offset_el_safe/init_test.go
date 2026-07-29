package offset_el_safe

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
)

const testOffset = 10 * time.Second

func newOffsetELSafeSystem(t devtest.T) *presets.SingleChainMultiNode {
	return presets.NewSingleChainMultiNode(t,
		presets.WithGlobalL2CLOption(sysgo.L2CLOptionFn(func(_ devtest.T, _ sysgo.ComponentTarget, cfg *sysgo.L2CLConfig) {
			cfg.VerifierSyncMode = nodeSync.ELSync
			cfg.OffsetELSafe = testOffset
		})),
	)
}
