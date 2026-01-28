package boundary

import (
	"net/url"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

// TestSupernodeInteropActivationBoundary tests behavior when interop is activated
// AFTER genesis. This verifies:
// - Safety proceeds normally from genesis until the activation timestamp
// - After activation, interop verification kicks in
// - The transition from pre-interop to post-interop is smooth
// - VerifiedAt (via superroot_atTimestamp) returns verified data for both phases
//
// Key expectation: Before the interop activation timestamp, safe heads should advance
// WITHOUT requiring cross-chain verification (normal derivation safety).
// VerifiedAt should still work because timestamps before activation are auto-verified.
func TestSupernodeInteropActivationBoundary(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewTwoL2SupernodeInteropDelayed(t, InteropActivationDelay)

	genesisTime := sys.GenesisTime
	activationTime := sys.InteropActivationTime
	blockTime := sys.L2A.Escape().RollupConfig().BlockTime

	t.Logger().Info("testing interop activation boundary",
		"genesis_time", genesisTime,
		"activation_time", activationTime,
		"delay_seconds", sys.DelaySeconds,
		"block_time", blockTime,
	)

	// Create a SuperNodeClient to call superroot_atTimestamp (which uses VerifiedAt internally)
	// The UserRPC returns a chain-specific path like http://host:port/901
	// We need the root path (http://host:port/) which has the superroot API
	chainRpcAddr := sys.L2ACL.Escape().UserRPC()
	parsedURL, err := url.Parse(chainRpcAddr)
	t.Require().NoError(err, "failed to parse RPC URL")
	baseRpcAddr := parsedURL.Scheme + "://" + parsedURL.Host + "/"

	ctx := t.Ctx()
	rpc, err := client.NewRPC(ctx, t.Logger(), baseRpcAddr, client.WithLazyDial())
	t.Require().NoError(err, "failed to create RPC client")
	snClient := sources.NewSuperNodeClient(rpc)
	defer snClient.Close()

	// Calculate how many blocks should be produced before activation
	blocksBeforeActivation := sys.DelaySeconds / blockTime
	t.Logger().Info("expected blocks before activation", "blocks", blocksBeforeActivation)

	// PHASE 1: Wait for safe heads to advance BEFORE the activation timestamp
	// This proves that normal safety (without interop) works from genesis to activation
	t.Require().Eventually(func() bool {
		statusA := sys.L2ACL.SyncStatus()
		statusB := sys.L2BCL.SyncStatus()

		// Safe heads should advance even though we haven't reached activation yet
		// The key is that safe time is still BEFORE activation time
		preActivation := statusA.SafeL2.Time < activationTime && statusB.SafeL2.Time < activationTime
		hasProgress := statusA.SafeL2.Number > 0 && statusB.SafeL2.Number > 0

		t.Logger().Debug("waiting for pre-activation safety progress",
			"chainA_safe_num", statusA.SafeL2.Number,
			"chainA_safe_time", statusA.SafeL2.Time,
			"chainB_safe_num", statusB.SafeL2.Number,
			"chainB_safe_time", statusB.SafeL2.Time,
			"activation_time", activationTime,
			"pre_activation", preActivation,
		)

		return hasProgress && preActivation
	}, 60*time.Second, time.Second, "safe heads should progress before interop activation")

	// Record the state at this point (before activation)
	preActivationStatusA := sys.L2ACL.SyncStatus()
	preActivationStatusB := sys.L2BCL.SyncStatus()

	t.Logger().Info("pre-activation state captured",
		"chainA_safe_num", preActivationStatusA.SafeL2.Number,
		"chainA_safe_time", preActivationStatusA.SafeL2.Time,
		"chainB_safe_num", preActivationStatusB.SafeL2.Number,
		"chainB_safe_time", preActivationStatusB.SafeL2.Time,
	)

	// Verify we actually have blocks before activation
	t.Require().Less(preActivationStatusA.SafeL2.Time, activationTime,
		"chain A safe time should be before activation")
	t.Require().Less(preActivationStatusB.SafeL2.Time, activationTime,
		"chain B safe time should be before activation")
	t.Require().Greater(preActivationStatusA.SafeL2.Number, uint64(0),
		"chain A should have safe blocks before activation")
	t.Require().Greater(preActivationStatusB.SafeL2.Number, uint64(0),
		"chain B should have safe blocks before activation")

	// PHASE 1b: Verify VerifiedAt works for pre-activation timestamps
	// Timestamps before activation should be auto-verified (interop wasn't active yet)
	preActivationTs := preActivationStatusA.SafeL2.Time
	t.Logger().Info("checking VerifiedAt for pre-activation timestamp", "timestamp", preActivationTs)

	preActivationResp, err := snClient.SuperRootAtTimestamp(ctx, preActivationTs)
	t.Require().NoError(err, "superroot_atTimestamp should not error for pre-activation timestamp")
	t.Require().NotNil(preActivationResp.Data, "VerifiedAt should return data for pre-activation timestamp (auto-verified)")
	t.Logger().Info("pre-activation VerifiedAt check passed",
		"timestamp", preActivationTs,
		"verified_required_l1", preActivationResp.Data.VerifiedRequiredL1,
		"super_root", preActivationResp.Data.SuperRoot,
	)

	// PHASE 2: Wait for safe heads to advance PAST the activation timestamp
	// This proves that interop verification works after activation
	t.Require().Eventually(func() bool {
		statusA := sys.L2ACL.SyncStatus()
		statusB := sys.L2BCL.SyncStatus()

		postActivation := statusA.SafeL2.Time >= activationTime && statusB.SafeL2.Time >= activationTime

		t.Logger().Debug("waiting for post-activation safety progress",
			"chainA_safe_time", statusA.SafeL2.Time,
			"chainB_safe_time", statusB.SafeL2.Time,
			"activation_time", activationTime,
			"post_activation", postActivation,
		)

		return postActivation
	}, 90*time.Second, time.Second, "safe heads should progress past interop activation")

	// Record final state
	postActivationStatusA := sys.L2ACL.SyncStatus()
	postActivationStatusB := sys.L2BCL.SyncStatus()

	t.Logger().Info("post-activation state captured",
		"chainA_safe_num", postActivationStatusA.SafeL2.Number,
		"chainA_safe_time", postActivationStatusA.SafeL2.Time,
		"chainB_safe_num", postActivationStatusB.SafeL2.Number,
		"chainB_safe_time", postActivationStatusB.SafeL2.Time,
	)

	// Verify both chains crossed the activation boundary
	t.Require().GreaterOrEqual(postActivationStatusA.SafeL2.Time, activationTime,
		"chain A should be at or past activation time")
	t.Require().GreaterOrEqual(postActivationStatusB.SafeL2.Time, activationTime,
		"chain B should be at or past activation time")

	// Verify progress was made across the boundary
	t.Require().Greater(postActivationStatusA.SafeL2.Number, preActivationStatusA.SafeL2.Number,
		"chain A should have more safe blocks after activation")
	t.Require().Greater(postActivationStatusB.SafeL2.Number, preActivationStatusB.SafeL2.Number,
		"chain B should have more safe blocks after activation")

	// PHASE 2b: Verify VerifiedAt works for post-activation timestamps
	// Timestamps at or after activation should be verified via the interop activity
	// The interop activity may still be catching up, so we poll until verified
	postActivationTs := postActivationStatusA.SafeL2.Time
	t.Logger().Info("checking VerifiedAt for post-activation timestamp", "timestamp", postActivationTs)

	var postActivationResp eth.SuperRootAtTimestampResponse
	t.Require().Eventually(func() bool {
		var err error
		postActivationResp, err = snClient.SuperRootAtTimestamp(ctx, postActivationTs)
		if err != nil {
			t.Logger().Warn("superroot_atTimestamp error, retrying", "timestamp", postActivationTs, "err", err)
			return false
		}
		if postActivationResp.Data == nil {
			t.Logger().Debug("waiting for interop to verify post-activation timestamp", "timestamp", postActivationTs)
			return false
		}
		return true
	}, 60*time.Second, time.Second, "VerifiedAt should return data for post-activation timestamp (interop-verified)")
	t.Logger().Info("post-activation VerifiedAt check passed",
		"timestamp", postActivationTs,
		"verified_required_l1", postActivationResp.Data.VerifiedRequiredL1,
		"super_root", postActivationResp.Data.SuperRoot,
	)

	// Additional check: verify that the activation timestamp itself is verified
	t.Logger().Info("checking VerifiedAt for exact activation timestamp", "timestamp", activationTime)
	var activationResp eth.SuperRootAtTimestampResponse
	t.Require().Eventually(func() bool {
		var err error
		activationResp, err = snClient.SuperRootAtTimestamp(ctx, activationTime)
		if err != nil {
			t.Logger().Warn("superroot_atTimestamp error for activation, retrying", "timestamp", activationTime, "err", err)
			return false
		}
		if activationResp.Data == nil {
			t.Logger().Debug("waiting for interop to verify activation timestamp", "timestamp", activationTime)
			return false
		}
		return true
	}, 60*time.Second, time.Second, "VerifiedAt should return data for activation timestamp")
	t.Logger().Info("activation timestamp VerifiedAt check passed",
		"timestamp", activationTime,
		"verified_required_l1", activationResp.Data.VerifiedRequiredL1,
		"super_root", activationResp.Data.SuperRoot,
	)

	t.Logger().Info("activation boundary test complete",
		"pre_activation_blocks_A", preActivationStatusA.SafeL2.Number,
		"post_activation_blocks_A", postActivationStatusA.SafeL2.Number,
		"pre_activation_blocks_B", preActivationStatusB.SafeL2.Number,
		"post_activation_blocks_B", postActivationStatusB.SafeL2.Number,
	)
}

