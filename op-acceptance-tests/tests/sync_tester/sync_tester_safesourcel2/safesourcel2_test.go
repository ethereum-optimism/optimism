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

	sys.L2CL.Matched(sys.L2CL2, types.LocalSafe, 30)

	logger.Info("SyncTester SafeSourceL2 test completed successfully")

	logger.Info("### Safe  ", "ver", sys.L2CL2.SafeL2BlockRef(), "seq", sys.L2CL.SafeL2BlockRef())
	logger.Info("### Unsafe", "ver", sys.L2CL2.UnsafeHead(), "seq", sys.L2CL.UnsafeHead())
	// Safe matches but unsafe gap still happen

	sys.L2CL.Matched(sys.L2CL2, types.LocalUnsafe, 100)

	logger.Info("### Safe  ", "ver", sys.L2CL2.SafeL2BlockRef(), "seq", sys.L2CL.SafeL2BlockRef())
	logger.Info("### Unsafe", "ver", sys.L2CL2.UnsafeHead(), "seq", sys.L2CL.UnsafeHead())
	logger.Info("### Finalzed", "ver", sys.L2CL2.SyncStatus().FinalizedL2, "seq", sys.L2CL.SyncStatus().FinalizedL2)

}
