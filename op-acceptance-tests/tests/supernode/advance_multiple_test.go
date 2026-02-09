package supernode

import (
	"net/url"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/stretchr/testify/require"
)

// TestTwoChainProgress confirms that two L2 chains advance when using a shared CL
// it confirms:
// - the two L2 chains are different
// - the two CLs are using the same supernode
// - the two CLs are advancing unsafe and local safe heads
func TestTwoChainProgress(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewTwoL2(t)

	blockTime := sys.L2A.Escape().RollupConfig().BlockTime
	waitTime := time.Duration(blockTime+1) * time.Second

	// Check that the two CLs are on different chains
	require.NotEqual(t, sys.L2ACL.ChainID(), sys.L2BCL.ChainID())

	// Check that the two CLs are using the same supernode
	uA, err := url.Parse(sys.L2ACL.Escape().UserRPC())
	require.NoError(t, err)
	uB, err := url.Parse(sys.L2BCL.Escape().UserRPC())
	require.NoError(t, err)
	require.Equal(t, uA.Scheme, uB.Scheme)
	require.Equal(t, uA.Host, uB.Host)
	require.Equal(t, uA.Port(), uB.Port())

	// Record initial sync status
	statusA := sys.L2ACL.SyncStatus()
	statusB := sys.L2BCL.SyncStatus()

	t.Logger().Info("initial sync status",
		"chainA_unsafe", statusA.UnsafeL2.Number,
		"chainA_safe", statusA.SafeL2.Number,
		"chainB_unsafe", statusB.UnsafeL2.Number,
		"chainB_safe", statusB.SafeL2.Number,
	)

	// unsafe heads should advance
	t.Require().Eventually(func() bool {
		newStatusA := sys.L2ACL.SyncStatus()
		newStatusB := sys.L2BCL.SyncStatus()
		return newStatusA.UnsafeL2.Number > statusA.UnsafeL2.Number &&
			newStatusB.UnsafeL2.Number > statusB.UnsafeL2.Number
	}, 30*time.Second, waitTime, "chains should advance unsafe heads")

	// local safe heads should advance
	t.Require().Eventually(func() bool {
		newStatusA := sys.L2ACL.SyncStatus()
		newStatusB := sys.L2BCL.SyncStatus()
		t.Logger().Info("waiting for local safe head progression",
			"chainA_local_safe", newStatusA.LocalSafeL2.Number,
			"chainB_local_safe", newStatusB.LocalSafeL2.Number,
		)
		return newStatusA.LocalSafeL2.Number > statusA.LocalSafeL2.Number &&
			newStatusB.LocalSafeL2.Number > statusB.LocalSafeL2.Number
	}, 60*time.Second, waitTime, "chains should advance local safe heads")

	// Log final status
	finalStatusA := sys.L2ACL.SyncStatus()
	finalStatusB := sys.L2BCL.SyncStatus()
	t.Logger().Info("final sync status",
		"chainA_unsafe", finalStatusA.UnsafeL2.Number,
		"chainA_safe", finalStatusA.SafeL2.Number,
		"chainB_unsafe", finalStatusB.UnsafeL2.Number,
		"chainB_safe", finalStatusB.SafeL2.Number,
	)

}
