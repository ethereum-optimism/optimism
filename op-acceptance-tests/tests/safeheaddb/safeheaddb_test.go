package safeheaddb

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestTruncateDatabaseOnResync(gt *testing.T) {
	t := devtest.SerialT(gt)
	t.Skip("Known issue: safe head db is not currently truncated when EL sync is used")
	sys := presets.NewSingleChainMultiNode(t)

	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalSafe, 1, 30),
		sys.L2CLB.AdvancedFn(types.LocalSafe, 1, 30))

	sys.L2CLB.Matched(sys.L2CL, types.LocalSafe, 30)
	sys.L2CLB.VerifySafeHeadDatabaseMatches(sys.L2CL)

	sys.L2ELB.Stop()
	sys.L2CLB.Stop()

	sys.L2CL.Advanced(types.LocalSafe, 3, 30)

	sys.L2ELB.Start()
	sys.L2CLB.Start()
	sys.L2ELB.PeerWith(sys.L2EL)

	sys.L2CLB.Matched(sys.L2CL, types.LocalSafe, 30)

	sys.L2CLB.VerifySafeHeadDatabaseMatches(sys.L2CL)
}
