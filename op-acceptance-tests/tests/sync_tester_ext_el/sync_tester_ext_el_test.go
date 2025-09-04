package sync_tester_ext_el

import (
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestSyncTesterExtEL(gt *testing.T) {
	t := devtest.SerialT(gt)

	if os.Getenv("CIRCLECI_PIPELINE_SCHEDULE_NAME") != "build_daily" && os.Getenv("CIRCLECI_PARAMETERS_SYNC_TEST_OP_NODE_DISPATCH") != "true" {
		t.Skip("TestSyncTesterExtEL only runs on daily scheduled pipeline jobs: %s %s", os.Getenv("CIRCLECI_PIPELINE_SCHEDULE_NAME"), os.Getenv("CIRCLECI_PARAMETERS_SYNC_TEST_OP_NODE_DISPATCH"))
	}

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

	blocksToSync := uint64(20)
	targetBlock := InitialL2Block + blocksToSync
	sys.L2CL.Reached(types.LocalUnsafe, targetBlock, 500)

	l2CLSyncStatus = sys.L2CL.SyncStatus()
	require.NotNil(l2CLSyncStatus, "L2CL should have sync status")

	unsafeL2Ref := l2CLSyncStatus.UnsafeL2
	blk := sys.L2EL.BlockRefByNumber(unsafeL2Ref.Number)
	require.Equal(unsafeL2Ref.Hash, blk.Hash, "L2EL should be on the same block as L2CL")

	stSessions := sys.SyncTester.ListSessions()
	require.Equal(len(stSessions), 1, "expect exactly one session")

	stSession := sys.SyncTester.GetSession(stSessions[0])
	require.GreaterOrEqual(stSession.CurrentState.Latest, stSession.InitialState.Latest+blocksToSync, "SyncTester session Latest should be on the same block as L2CL")
	require.GreaterOrEqual(stSession.CurrentState.Safe, stSession.InitialState.Safe+blocksToSync, "SyncTester session Safe should be on the same block as L2CL")

	t.Logger().Info("SyncTester ExtEL test completed successfully",
		"l2cl_chain_id", l2CLChainID,
		"l2cl_sync_status", l2CLSyncStatus)
}
