package sync_tester_elsync

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestSyncTesterELSync(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleWithSyncTester(t)
	require := t.Require()
	logger := t.Logger()
	ctx := t.Ctx()

	startDelta := uint64(5)
	attempts := 30
	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalUnsafe, startDelta, attempts),
		sys.L2CL2.AdvancedFn(types.LocalUnsafe, startDelta, attempts),
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

	// Reseted and L2CL2 not connected to sync tester session so unsafe head will not advance
	require.Equal(uint64(0), sys.SyncTesterL2EL.BlockRefByLabel(eth.Unsafe).Number)

	// Wait for L2CL to advance more unsafe blocks
	delta := uint64(5)
	sys.L2CL.Advanced(types.LocalUnsafe, startDelta+delta, attempts)

	// EL Sync active
	session, err := syncTesterClient.GetSession(ctx)
	require.NoError(err)
	require.True(session.ELSyncActive)

	// Restarting will trigger EL sync since unsafe head payload will arrive to L2CL2 via P2P
	sys.L2CL2.Start()

	// Wait until P2P is connected
	sys.L2CL2.IsP2PConnected(sys.L2CL)

	// Sequencer EL and SyncTester EL advances together
	target := sys.L2EL.BlockRefByLabel(eth.Unsafe).Number + 5
	dsl.CheckAll(t,
		sys.L2CL2.ReachedFn(types.LocalUnsafe, target, attempts),
		// EL Sync complete
		sys.SyncTesterL2EL.ReachedFn(eth.Unsafe, target, attempts),
	)

	// Check CL2 view is consistent with read only EL
	unsafeHead := sys.L2CL2.SyncStatus().UnsafeL2
	require.GreaterOrEqual(unsafeHead.Number, target)
	require.Equal(sys.L2EL.BlockRefByNumber(unsafeHead.Number).Hash, unsafeHead.Hash)
}
