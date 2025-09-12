package sync_tester_elsync

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestSyncTesterELSync(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleWithSyncTester(t)
	require := t.Require()
	logger := t.Logger()
	ctx := t.Ctx()

	target := uint64(5)
	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalUnsafe, target, 30),
		sys.L2CL2.AdvancedFn(types.LocalUnsafe, target, 30),
	)

	// Stop L2CL2 attached to Sync Tester EL Endpoint
	sys.L2CL2.Stop()

	// Reset Sync Tester EL
	sessionIDs := sys.SyncTester.ListSessions()
	require.GreaterOrEqual(len(sessionIDs), 1, "at least one session")
	sessionID := sessionIDs[0]
	logger.Info("SyncTester EL", "sessionID", sessionID)
	syncTesterClient := sys.SyncTester.Escape().APIWithSession(sessionID)
	require.NoError(syncTesterClient.ResetSession(ctx))

	// Wait for L2CL to advance more unsafe blocks
	sys.L2CL.Advanced(types.LocalUnsafe, target+5, 30)

	// Restarting will trigger EL sync since unsafe head payload will arrive to L2CL2 via P2P
	sys.L2CL2.Start()

	// Wait until P2P is connected

	// TODO TODO: make sure P2P is back but not connected
	sys.L2CL2.NotAdvanced(types.LocalUnsafe, 10)

}
