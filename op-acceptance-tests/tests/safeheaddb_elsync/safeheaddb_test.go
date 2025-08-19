package safeheaddb_elsync

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestTruncateDatabaseOnELResync(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNode(t)

	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalSafe, 1, 30),
		sys.L2CLB.AdvancedFn(types.LocalSafe, 1, 30))

	sys.L2CLB.Matched(sys.L2CL, types.LocalSafe, 30)
	sys.L2CLB.VerifySafeHeadDatabaseMatches(sys.L2CL)

	// Stop the verifier node. Since the sysgo EL uses in-memory storage this also wipes its database.
	// With the EL reset to genesis, when the CL restarts it will use EL sync to resync the chain rather than
	// deriving it from L1.
	sys.L2ELB.Stop()
	sys.L2CLB.Stop()

	sys.L2CL.Advanced(types.LocalSafe, 3, 30)

	sys.L2ELB.Start()
	sys.L2CLB.Start()
	sys.L2ELB.PeerWith(sys.L2EL)

	sys.L2CLB.Matched(sys.L2CL, types.LocalSafe, 30)
	sys.L2CLB.Advanced(types.LocalSafe, 1, 30) // At least one safe head db update after resync

	sys.L2CLB.VerifySafeHeadDatabaseMatches(sys.L2CL)
}

func TestNotTruncateDatabaseOnRestartWithExistingDatabase(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNode(t)

	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalSafe, 1, 30),
		sys.L2CLB.AdvancedFn(types.LocalSafe, 1, 30))
	sys.L2CLB.Matched(sys.L2CL, types.LocalSafe, 30)

	preRestartSafeBlock := sys.L2CLB.SafeL2BlockRef().Number
	sys.L2CLB.VerifySafeHeadDatabaseMatches(sys.L2CL, dsl.WithMinRequiredL2Block(preRestartSafeBlock))

	// Restart the verifier op-node, but not the EL so the existing chain data is not deleted.
	sys.L2CLB.Stop()

	sys.L2CL.Advanced(types.LocalSafe, 3, 30)

	sys.L2CLB.Start()

	sys.L2CLB.Matched(sys.L2CL, types.LocalSafe, 30)
	sys.L2CLB.Advanced(types.LocalSafe, 1, 30) // At least one safe head db update after resync

	sys.L2CLB.VerifySafeHeadDatabaseMatches(sys.L2CL, dsl.WithMinRequiredL2Block(preRestartSafeBlock))
}
