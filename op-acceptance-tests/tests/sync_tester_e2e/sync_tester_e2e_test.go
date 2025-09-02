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

	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalUnsafe, 5, 30),
		sys.L2CL2.AdvancedFn(types.LocalUnsafe, 5, 30),
	)

	// Test that we can get chain ID from SyncTester
	syncTester := sys.SyncTester.Escape()
	syncTesterChainID, err := syncTester.API().ChainID(t.Ctx())
	require.NoError(err, "should be able to get chain ID from SyncTester")
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
}
