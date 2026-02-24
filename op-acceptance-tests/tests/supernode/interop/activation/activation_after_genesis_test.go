package activation

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestSupernodeInteropActivationAfterGenesis tests behavior when interop is activated
// AFTER genesis. This verifies that VerifiedAt (via superroot_atTimestamp) returns
// verified data for timestamps both before and after the activation boundary.
func TestSupernodeInteropActivationAfterGenesis(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, InteropActivationDelay)

	genesisTime := sys.GenesisTime
	activationTime := sys.InteropActivationTime
	blockTime := sys.L2A.Escape().RollupConfig().BlockTime

	// Select timestamps before and after activation
	// Pre-activation: one block after genesis (before interop is active)
	// Post-activation: one block after activation (after interop is active)
	preActivationTs := genesisTime + blockTime
	postActivationTs := activationTime + blockTime

	t.Logger().Info("testing interop activation boundary",
		"genesis_time", genesisTime,
		"activation_time", activationTime,
		"pre_activation_ts", preActivationTs,
		"post_activation_ts", postActivationTs,
		"block_time", blockTime,
	)

	ctx := t.Ctx()
	snClient := sys.SuperNodeClient()

	// Wait for both timestamps to be verified via SuperRootAtTimestamp
	// Pre-activation timestamps are auto-verified (interop wasn't active yet)
	// Post-activation timestamps require interop verification
	var preActivationResp, postActivationResp eth.SuperRootAtTimestampResponse
	t.Require().Eventually(func() bool {
		var err error

		// Check pre-activation timestamp
		preActivationResp, err = snClient.SuperRootAtTimestamp(ctx, preActivationTs)
		if err != nil {
			t.Logger().Warn("superroot_atTimestamp error for pre-activation", "timestamp", preActivationTs, "err", err)
			return false
		}
		preVerified := preActivationResp.Data != nil

		// Check post-activation timestamp
		postActivationResp, err = snClient.SuperRootAtTimestamp(ctx, postActivationTs)
		if err != nil {
			t.Logger().Warn("superroot_atTimestamp error for post-activation", "timestamp", postActivationTs, "err", err)
			return false
		}
		postVerified := postActivationResp.Data != nil

		t.Logger().Info("waiting for both timestamps to be verified",
			"pre_activation_ts", preActivationTs,
			"pre_verified", preVerified,
			"post_activation_ts", postActivationTs,
			"post_verified", postVerified,
		)

		return preVerified && postVerified
	}, 300*time.Second, time.Second, "both pre and post activation timestamps should be verified")

	t.Logger().Info("activation boundary test complete",
		"pre_activation_ts", preActivationTs,
		"pre_activation_super_root", preActivationResp.Data.SuperRoot,
		"post_activation_ts", postActivationTs,
		"post_activation_super_root", postActivationResp.Data.SuperRoot,
	)
}
