package sync_tester_safesourcel2

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestSyncTesterSafeSourceL2(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleWithSyncTesterSafeSourceL2(t)
	require := t.Require()
	logger := t.Logger()
	ctx := t.Ctx()

	target := uint64(5)
	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalUnsafe, target, 30),
		sys.L2CL2.AdvancedFn(types.LocalUnsafe, target, 30),
	)

	// Stop L2CL2 which is using safe-source=l2
	sys.L2CL2.Stop()

	// Reset Sync Tester EL
	sessionIDs := sys.SyncTester.ListSessions()
	require.GreaterOrEqual(len(sessionIDs), 1, "at least one session")
	sessionID := sessionIDs[0]
	logger.Info("SyncTester EL", "sessionID", sessionID)
	syncTesterClient := sys.SyncTester.Escape().APIWithSession(sessionID)
	require.NoError(syncTesterClient.ResetSession(ctx))

	// Wait for L2CL to advance more unsafe and safe blocks
	sys.L2CL.Advanced(types.LocalUnsafe, target+5, 30)
	sys.L2CL.Advanced(types.LocalSafe, target+3, 30)

	// Restarting will allow L2CL2 to query safe head from L2CL via safe-source=l2
	sys.L2CL2.Start()

	// Wait until P2P is connected for unsafe head gossip
	sys.L2CL2.IsP2PConnected(sys.L2CL)

	// L2CL2 should catch up via safe-source=l2
	target = uint64(20)
	sys.L2CL.Reached(types.LocalSafe, target, 30)
	sys.L2CL2.Reached(types.LocalSafe, target, 30)

	// Verify safe heads match
	l2CLStatus := sys.L2CL.SyncStatus()
	l2CL2Status := sys.L2CL2.SyncStatus()

	require.Equal(l2CLStatus.SafeL2.Hash, l2CL2Status.SafeL2.Hash, "Safe heads should match")
	require.Equal(l2CLStatus.SafeL2.Number, l2CL2Status.SafeL2.Number, "Safe block numbers should match")

	logger.Info("SyncTester SafeSourceL2 test completed successfully",
		"l2cl_safe", l2CLStatus.SafeL2,
		"l2cl2_safe", l2CL2Status.SafeL2)
}
