package opcon

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// safeDBOpt gives every op-node in the topology a SafeDB path so it can answer
// optimism_safeHeadAtL1Block under the default op-node CL kind. op-con-node
// serves its own safe-head history from SQL and ignores the path, so the option
// is harmless when slots route to op-con-node. Shared by the verifier and
// sequencer safe-head-DB parity tests so the flip pair stays on identical
// topology config.
func safeDBOpt() presets.Option {
	return presets.WithGlobalL2CLOption(sysgo.L2CLOptionFn(func(p devtest.T, _ sysgo.ComponentTarget, cfg *sysgo.L2CLConfig) {
		cfg.SafeDBPath = p.TempDir()
	}))
}

// awaitSafeConvergence waits until both CL nodes advance their safe heads and
// the second node is in sync with the first at the safe level — the shared
// "batcher landed blocks on L1 and both nodes derived the same safe chain"
// precondition used across the opcon suite.
func awaitSafeConvergence(t devtest.T, sys *presets.SingleChainMultiNode, attempts int) {
	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalSafe, 1, attempts),
		sys.L2CLB.AdvancedFn(types.LocalSafe, 1, attempts),
	)
	sys.L2CLB.InSync(sys.L2CL, types.LocalSafe, attempts)
}
