package interop

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestSupernodeInteropActivationAtGenesis tests behavior when interop is activated
// at genesis time (timestamp 0 offset). This verifies the first few timestamps are
// processed correctly with interop verification from the very beginning.
// Also verifies that VerifiedAt (via superroot_atTimestamp) works correctly.
func TestSupernodeInteropActivationAtGenesis(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newSupernodeInteropWithTimeTravel(t, 0)

	genesisTime := sys.L2A.Escape().RollupConfig().Genesis.L2Time
	blockTime := sys.L2A.Escape().RollupConfig().BlockTime

	t.Logger().Info("testing interop activation at genesis",
		"genesis_time", genesisTime,
		"block_time", blockTime,
	)

	// Create a SuperNodeClient to call superroot_atTimestamp (which uses VerifiedAt internally)
	ctx := t.Ctx()
	snClient := sys.SuperNodeClient()

	// The first timestamp to be verified should be genesis + blockTime
	// (genesis block doesn't have L1 data recorded in safeDB yet)
	targetTimestamp := genesisTime + blockTime
	t.Logger().Info("checking VerifiedAt for first block after genesis", "timestamp", targetTimestamp)

	// Wait for interop to verify the first block after genesis
	var genesisResp eth.SuperRootAtTimestampResponse
	t.Require().Eventually(func() bool {
		var err error
		genesisResp, err = snClient.SuperRootAtTimestamp(ctx, targetTimestamp)
		if err != nil {
			t.Logger().Warn("superroot_atTimestamp error, retrying", "timestamp", targetTimestamp, "err", err)
			return false
		}
		if genesisResp.Data == nil {
			t.Logger().Debug("waiting for interop to verify first block", "timestamp", targetTimestamp)
			return false
		}
		return true
	}, 60*time.Second, time.Second, "VerifiedAt should return data for first block after genesis (interop-verified)")

	t.Logger().Info("genesis activation verified",
		"timestamp", targetTimestamp,
		"verified_required_l1", genesisResp.Data.VerifiedRequiredL1,
		"super_root", genesisResp.Data.SuperRoot,
	)
}
