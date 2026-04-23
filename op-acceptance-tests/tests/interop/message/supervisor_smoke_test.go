package msg

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/stretchr/testify/require"
)

// TestInteropSystemSupernode tests that the supernode's CL nodes track finalized L1 block information
func TestInteropSystemSupernode(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, 0)

	// First ensure L1 network is online and has blocks
	sys.L1Network.WaitForOnline()
	initialBlock := sys.L1Network.WaitForBlock()
	t.Logger().Info("Got initial L1 block", "block", initialBlock)

	// Wait for L1 finalization
	finalizedBlock := sys.L1Network.WaitForFinalization()
	t.Logger().Info("L1 block finalized", "block", finalizedBlock)

	// Wait for each CL node to observe the finalized L1 block.
	// The CL may lag behind the L1 EL, so poll until it catches up.
	require.Eventually(t, func() bool {
		ss := sys.L2ACL.SyncStatus()
		return ss.FinalizedL1.Number >= finalizedBlock.Number
	}, 30*time.Second, 2*time.Second,
		"Chain A CL should observe finalized L1 block %d", finalizedBlock.Number)
	t.Logger().Info("Chain A finalized L1", "block", sys.L2ACL.SyncStatus().FinalizedL1)

	require.Eventually(t, func() bool {
		ss := sys.L2BCL.SyncStatus()
		return ss.FinalizedL1.Number >= finalizedBlock.Number
	}, 30*time.Second, 2*time.Second,
		"Chain B CL should observe finalized L1 block %d", finalizedBlock.Number)
	t.Logger().Info("Chain B finalized L1", "block", sys.L2BCL.SyncStatus().FinalizedL1)
}
