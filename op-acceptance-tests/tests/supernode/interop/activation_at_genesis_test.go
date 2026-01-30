package interop

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestSupernodeInteropActivationAtGenesis tests behavior when interop is activated
// at genesis time (timestamp 0 offset). This verifies the first few timestamps are
// processed correctly with interop verification from the very beginning.
// Also verifies that VerifiedAt (via superroot_atTimestamp) works correctly.
func TestSupernodeInteropActivationAtGenesis(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, 0)

	genesisTime := sys.L2A.Escape().RollupConfig().Genesis.L2Time
	blockTime := sys.L2A.Escape().RollupConfig().BlockTime

	t.Logger().Info("testing interop activation at genesis",
		"genesis_time", genesisTime,
		"block_time", blockTime,
	)

	// Create a SuperNodeClient to call superroot_atTimestamp (which uses VerifiedAt internally)
	ctx := t.Ctx()
	snClient := sys.SuperNodeClient()

	// The first timestamp to be verified should be the genesis time
	// Wait for safe head to advance past genesis
	t.Require().Eventually(func() bool {
		status := sys.L2ACL.SyncStatus()
		return status.SafeL2.Number > 0 && status.SafeL2.Time > genesisTime
	}, 60*time.Second, time.Second, "should advance past genesis")

	// Verify both chains have processed the activation boundary
	statusA := sys.L2ACL.SyncStatus()
	statusB := sys.L2BCL.SyncStatus()

	t.Logger().Info("genesis activation processed",
		"chainA_safe_time", statusA.SafeL2.Time,
		"chainB_safe_time", statusB.SafeL2.Time,
		"genesis_time", genesisTime,
	)

	// Both chains should have timestamps >= genesis
	t.Require().GreaterOrEqual(statusA.SafeL2.Time, genesisTime, "chain A should be at or past genesis")
	t.Require().GreaterOrEqual(statusB.SafeL2.Time, genesisTime, "chain B should be at or past genesis")

	// Verify VerifiedAt works for the safe head timestamp (interop was active from genesis)
	// Note: We use the safe head timestamp rather than the exact genesis timestamp because
	// the genesis block doesn't have L1 data recorded in the safeDB yet.
	safeTs := statusA.SafeL2.Time
	t.Logger().Info("checking VerifiedAt for safe head timestamp", "timestamp", safeTs)

	// The interop activity may still be catching up, so we poll until verified
	var safeResp eth.SuperRootAtTimestampResponse
	t.Require().Eventually(func() bool {
		var err error
		safeResp, err = snClient.SuperRootAtTimestamp(ctx, safeTs)
		if err != nil {
			t.Logger().Warn("superroot_atTimestamp error, retrying", "timestamp", safeTs, "err", err)
			return false
		}
		if safeResp.Data == nil {
			t.Logger().Debug("waiting for interop to verify safe head timestamp", "timestamp", safeTs)
			return false
		}
		return true
	}, 60*time.Second, time.Second, "VerifiedAt should return data for safe head timestamp (interop-verified)")

	t.Logger().Info("safe head timestamp VerifiedAt check passed",
		"timestamp", safeTs,
		"verified_required_l1", safeResp.Data.VerifiedRequiredL1,
		"super_root", safeResp.Data.SuperRoot,
	)
}
