package sync_tester_ext_el

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestSyncTesterExtEL(gt *testing.T) {
	// if os.Getenv("NIGHTLY_CI_TAILSCALE_JOB") != "true" {
	// 	gt.Skip("Skipping test because NIGHTLY_CI_TAILSCALE_JOB is not set")
	// }

	t := devtest.SerialT(gt)

	sys := presets.NewMinimalExternalELWithExternalL1(t)
	require := t.Require()

	// Test that we can get chain IDs from L2CL node
	l2CLChainID := sys.L2CL.ID().ChainID()
	require.Equal(eth.ChainIDFromUInt64(11155420), l2CLChainID, "L2CL should be on chain 11155420")

	// Test that the network started successfully
	require.NotNil(sys.L1EL, "L1 EL node should be available")
	require.NotNil(sys.L2CL, "L2 CL node should be available")
	require.NotNil(sys.SyncTester, "SyncTester should be available")

	// Test that we can get sync status from L2CL node
	l2CLSyncStatus := sys.L2CL.SyncStatus()
	require.NotNil(l2CLSyncStatus, "L2CL should have sync status")

	sys.L2CL.Advanced(types.LocalUnsafe, 32012768, 1000)

	t.Logger().Info("SyncTester ExtEL test completed successfully",
		"l2cl_chain_id", l2CLChainID,
		"l2cl_sync_status", l2CLSyncStatus)
}
