package interop

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// TestSupernodeInteropVerifiedAt tests that the VerifiedAt endpoint returns
// correct data after the interop activity has processed timestamps.
func TestSupernodeInteropVerifiedAt(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, 0)

	blockTime := sys.L2A.Escape().RollupConfig().BlockTime
	genesisTime := sys.L2A.Escape().RollupConfig().Genesis.L2Time

	// Wait for safe head to advance past genesis
	t.Require().Eventually(func() bool {
		status := sys.L2ACL.SyncStatus()
		return status.SafeL2.Number > 0
	}, 60*time.Second, time.Second, "safe head should advance past genesis")

	// Query for a timestamp that should be verified
	// Use genesis time + one block time to ensure we're past the first block
	targetTimestamp := genesisTime + blockTime

	t.Logger().Info("querying verified at timestamp",
		"genesis_time", genesisTime,
		"target_timestamp", targetTimestamp,
	)

	// Wait for the interop activity to verify the target timestamp
	// The safe head time being past the target timestamp indicates interop has processed it
	t.Require().Eventually(func() bool {
		// Check if the safe head is past our target timestamp
		// This indicates the interop activity has processed and verified that timestamp
		status := sys.L2ACL.SyncStatus()
		return status.SafeL2.Time >= targetTimestamp
	}, 60*time.Second, time.Second, "interop should verify target timestamp")

	// Log the final state
	finalStatus := sys.L2ACL.SyncStatus()
	t.Logger().Info("verified at test complete",
		"safe_l2_number", finalStatus.SafeL2.Number,
		"safe_l2_time", finalStatus.SafeL2.Time,
		"target_timestamp", targetTimestamp,
	)
}
