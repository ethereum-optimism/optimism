package sync_tester_hfs

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestSyncTesterHardforks(gt *testing.T) {
	t := devtest.SerialT(gt)

	sys := presets.NewSimpleWithSyncTester(t)
	require := t.Require()
	logger := t.Logger()
	ctx := t.Ctx()

	// Check the L2CL passed configured hardforks
	jovianTime := sys.L2Chain.Escape().ChainConfig().JovianTime
	require.NotNil(jovianTime, "jovian must be activated")

	// Hardforks will be activated from Bedrock to Jovian, 10 hardforks with 6 second time delta between.
	// 6 * 10 = 60s, so we need at least 30 (60 / 2 + 1) L2 blocks with block time 2 to make the CL experience scheduled hardforks.
	targetNum := 32

	// Unsafe advancement: NewPayload -> ForkchoiceUpdated(no attr)
	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalUnsafe, uint64(targetNum), targetNum+10),
		sys.L2CL2.AdvancedFn(types.LocalUnsafe, uint64(targetNum), targetNum+10),
	)

	current := sys.L2CL2.HeadBlockRef(types.LocalUnsafe)
	require.Greater(current.Time, *jovianTime, "must pass jovian block")
	// Check block hash state from L2CL2 which was synced using the sync tester
	require.Equal(sys.L2EL.BlockRefByNumber(current.Number).Hash, current.Hash, "hash mismatch")
	logger.Info("Advancement using CLP2P done", "head", sys.L2EL.UnsafeHead())

	// Disconnect CLP2P to solely rely on derivation
	sys.L2CL2.DisconnectPeer(sys.L2CL)
	sys.L2CL.DisconnectPeer(sys.L2CL2)
	sys.L2CL2.Stop()
	sessionIDs := sys.SyncTester.ListSessions()
	require.GreaterOrEqual(len(sessionIDs), 1, "at least one session")
	sessionID := sessionIDs[0]
	logger.Info("SyncTester EL", "sessionID", sessionID)
	syncTesterClient := sys.SyncTester.Escape().APIWithSession(sessionID)
	// Resync starting from genesis
	require.NoError(syncTesterClient.ResetSession(ctx))
	sys.SyncTesterL2EL.UnsafeHead().NumEqualTo(0)

	// Wait until safe head reached Jovian
	sys.L2CL.Reached(types.LocalSafe, current.Number, 20)

	// Check safe head advancement can solely rely on derivation reaching Jovian
	// Safe advancement: ForkchoiceUpdated(with attr) -> GetPayload -> NewPayload -> ForkchoiceUpdated(no attr)
	sys.L2CL2.Start()
	sys.L2CL2.Reached(types.LocalSafe, current.Number, 20)
	sys.SyncTesterL2EL.Reached(eth.Safe, current.Number, 10)

	current = sys.L2CL2.HeadBlockRef(types.LocalSafe)
	require.Greater(current.Time, *jovianTime, "must pass jovian block")
	// Check block hash state from L2CL2 which was synced using the sync tester
	require.Equal(sys.L2EL.BlockRefByNumber(current.Number).Hash, current.Hash, "hash mismatch")
	logger.Info("Advancement using derivation done", "head", sys.L2EL.UnsafeHead())
}
