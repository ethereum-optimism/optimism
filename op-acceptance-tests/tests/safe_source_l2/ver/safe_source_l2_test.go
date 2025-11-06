package safe_source_l2

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// TestSafeSourceL2BasicSync verifies that a verifier node configured with --safe-source=l2
// can follow the safe head of another verifier without performing derivation.
func TestSafeSourceL2BasicSync(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithSafeSourceL2(t)

	// Advance the normal verifier's safe head
	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalSafe, 5, 30),
	)

	// Verify the safe-source=l2 verifier matches the normal verifier
	sys.L2CLB.Matched(sys.L2CL, types.LocalSafe, 30)

	// Get sync status from both nodes
	l2CLStatus := sys.L2CL.SyncStatus()
	l2CLBStatus := sys.L2CLB.SyncStatus()

	require := t.Require()
	require.Equal(l2CLStatus.SafeL2.Hash, l2CLBStatus.SafeL2.Hash, "Safe heads should match")
	require.Equal(l2CLStatus.SafeL2.Number, l2CLBStatus.SafeL2.Number, "Safe block numbers should match")
	require.Equal(l2CLStatus.FinalizedL2.Hash, l2CLBStatus.FinalizedL2.Hash, "Finalized heads should match")
	require.Equal(l2CLStatus.FinalizedL2.Number, l2CLBStatus.FinalizedL2.Number, "Finalized block numbers should match")

	// Advance further to ensure continued sync
	sys.L2CL.Advanced(types.LocalSafe, 5, 30)
	sys.L2CLB.Matched(sys.L2CL, types.LocalSafe, 30)
}
