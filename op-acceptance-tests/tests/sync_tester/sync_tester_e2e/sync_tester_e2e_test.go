package sync_tester_e2e

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestSyncTesterE2E(gt *testing.T) {
	t := devtest.SerialT(gt)
	// This test uses DefaultSimpleSystemWithSyncTester which includes:
	// - Minimal setup with L1EL, L1CL, L2EL, L2CL (sequencer)
	// - Additional L2CL2 (verifier) that connects to SyncTester instead of L2EL
	sys := presets.NewSimpleWithSyncTester(t)
	require := t.Require()
	logger := t.Logger()
	ctx := t.Ctx()

	// Test that we can get chain IDs from both L2CL nodes
	l2CLChainID := sys.L2CL.ID().ChainID()
	require.Equal(eth.ChainIDFromUInt64(901), l2CLChainID, "first L2CL should be on chain 901")

	l2CL2ChainID := sys.L2CL2.ID().ChainID()
	require.Equal(eth.ChainIDFromUInt64(901), l2CL2ChainID, "second L2CL should be on chain 901")

	// Test that the network started successfully
	require.NotNil(sys.L1EL, "L1 EL node should be available")
	require.NotNil(sys.L2EL, "L2 EL node should be available")
	require.NotNil(sys.L2CL, "L2 CL node should be available")
	require.NotNil(sys.SyncTester, "SyncTester should be available")
	require.NotNil(sys.L2CL2, "Second L2 CL node should be available")
	require.NotNil(sys.SyncTesterL2EL, "SyncTester L2 EL node should be available")

	sessionIDs := sys.SyncTester.ListSessions()
	require.GreaterOrEqual(len(sessionIDs), 1, "at least one session")

	sessionID := sessionIDs[0]
	logger.Info("SyncTester EL", "sessionID", sessionID)

	session := sys.SyncTester.GetSession(sessionID)

	require.Equal(eth.FCUState{Latest: 0, Safe: 0, Finalized: 0}, session.InitialState)

	target := uint64(5)
	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalUnsafe, target, 30),
		sys.L2CL2.AdvancedFn(types.LocalUnsafe, target, 30),
	)

	// Test that we can get chain ID from SyncTester
	syncTesterChainID := sys.SyncTester.ChainID(sessionID)
	require.Equal(eth.ChainIDFromUInt64(901), syncTesterChainID, "SyncTester should be on chain 901")

	// Test that both L2CL nodes and SyncTester are on the same chain
	require.Equal(l2CLChainID, l2CL2ChainID, "both L2CL nodes should be on the same chain")
	require.Equal(l2CLChainID, syncTesterChainID, "L2CL nodes and SyncTester should be on the same chain")

	// Test that we can get sync status from L2CL nodes
	l2CLSyncStatus := sys.L2CL.SyncStatus()
	require.NotNil(l2CLSyncStatus, "first L2CL should have sync status")

	l2CL2SyncStatus := sys.L2CL2.SyncStatus()
	require.NotNil(l2CL2SyncStatus, "second L2CL should have sync status")

	t.Logger().Info("SyncTester E2E test completed successfully",
		"l2cl_chain_id", l2CLChainID,
		"l2cl2_chain_id", l2CL2ChainID,
		"sync_tester_chain_id", syncTesterChainID,
		"l2cl_sync_status", l2CLSyncStatus,
		"l2cl2_sync_status", l2CL2SyncStatus)

	unsafeNum := sys.SyncTesterL2EL.BlockRefByLabel(eth.Unsafe).Number
	require.True(unsafeNum >= target, unsafeNum)

	session = sys.SyncTester.GetSession(sessionID)
	require.GreaterOrEqual(session.CurrentState.Latest, target)

	sys.SyncTester.DeleteSession(sessionID)

	syncTesterClient := sys.SyncTester.Escape().APIWithSession(sessionID)

	require.ErrorContains(syncTesterClient.DeleteSession(ctx), "already deleted")

	_, err := syncTesterClient.GetSession(ctx)
	require.ErrorContains(err, "already deleted")
}
