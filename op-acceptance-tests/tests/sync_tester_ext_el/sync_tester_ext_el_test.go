package sync_tester_ext_el

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestSyncTesterExtEL(gt *testing.T) {
	t := devtest.SerialT(gt)
	// This test uses NewMinimalExternalEL which includes:
	// - Minimal setup with external L1EL, L1CL, L2EL, L2CL (sequencer)
	// - SyncTester that connects to the L2CL instead of L2EL
	sys := presets.NewMinimalExternalEL(t)
	require := t.Require()

	// Test that we can get chain IDs from L2CL node
	// l2CLChainID := sys.L2CL.ID().ChainID()
	// require.Equal(eth.ChainIDFromUInt64(901), l2CLChainID, "L2CL should be on chain 901")

	// Test that the network started successfully
	require.NotNil(sys.L1EL, "L1 EL node should be available")
	require.NotNil(sys.L2CL, "L2 CL node should be available")
	require.NotNil(sys.SyncTester, "SyncTester should be available")

	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalUnsafe, 22285448, 30),
	)

	// Test that we can get chain ID from SyncTester
	syncTester := sys.SyncTester.Escape()
	syncTesterChainID, err := syncTester.API().ChainID(t.Ctx())
	require.NoError(err, "should be able to get chain ID from SyncTester")
	// require.Equal(eth.ChainIDFromUInt64(901), syncTesterChainID, "SyncTester should be on chain 901")

	// Test that L2CL node and SyncTester are on the same chain
	// require.Equal(l2CLChainID, syncTesterChainID, "L2CL node and SyncTester should be on the same chain")

	// Test that we can get sync status from L2CL node
	l2CLSyncStatus := sys.L2CL.SyncStatus()
	require.NotNil(l2CLSyncStatus, "L2CL should have sync status")

	t.Logger().Info("SyncTester ExtEL test completed successfully",
		// "l2cl_chain_id", l2CLChainID,
		"sync_tester_chain_id", syncTesterChainID,
		"l2cl_sync_status", l2CLSyncStatus)
}
