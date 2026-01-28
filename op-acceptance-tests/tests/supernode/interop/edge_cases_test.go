package interop

import (
	"net/url"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

// TestSupernodeInteropChainLag tests the behavior when one chain temporarily
// falls behind the other. The supernode should wait for all chains to reach
// the target timestamp before verifying.
func TestSupernodeInteropChainLag(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, 0)

	blockTime := sys.L2A.Escape().RollupConfig().BlockTime

	// Let both chains advance initially
	t.Require().Eventually(func() bool {
		statusA := sys.L2ACL.SyncStatus()
		statusB := sys.L2BCL.SyncStatus()
		return statusA.SafeL2.Number > 2 && statusB.SafeL2.Number > 2
	}, 60*time.Second, time.Second, "both chains should advance initially")

	// Record the current state
	statusA := sys.L2ACL.SyncStatus()
	statusB := sys.L2BCL.SyncStatus()

	t.Logger().Info("initial state before lag test",
		"chainA_safe", statusA.SafeL2.Number,
		"chainB_safe", statusB.SafeL2.Number,
	)

	// Stop chain B's CL to simulate lag
	sys.ControlPlane.L2CLNodeState(
		stack.NewL2CLNodeID("sequencer", sys.L2B.ChainID()),
		stack.Stop,
	)

	// Wait a bit to let chain A advance while B is stopped
	time.Sleep(time.Duration(blockTime*3) * time.Second)

	// Chain A should have advanced
	newStatusA := sys.L2ACL.SyncStatus()
	t.Logger().Info("chain A advanced while B was stopped",
		"chainA_safe", newStatusA.SafeL2.Number,
		"chainA_delta", newStatusA.SafeL2.Number-statusA.SafeL2.Number,
	)

	// Resume chain B
	sys.ControlPlane.L2CLNodeState(
		stack.NewL2CLNodeID("sequencer", sys.L2B.ChainID()),
		stack.Start,
	)

	// Wait for chain B to catch up
	timeout := time.Duration(blockTime*20+60) * time.Second
	t.Require().Eventually(func() bool {
		currentB := sys.L2BCL.SyncStatus()
		// Chain B should catch up to where chain A was
		return currentB.SafeL2.Number >= statusA.SafeL2.Number
	}, timeout, time.Second, "chain B should catch up after resume")

	// Both chains should continue advancing together
	finalStatusA := sys.L2ACL.SyncStatus()
	finalStatusB := sys.L2BCL.SyncStatus()

	t.Logger().Info("final state after lag recovery",
		"chainA_safe", finalStatusA.SafeL2.Number,
		"chainB_safe", finalStatusB.SafeL2.Number,
	)

	t.Require().Greater(finalStatusA.SafeL2.Number, statusA.SafeL2.Number, "chain A should have advanced")
	t.Require().Greater(finalStatusB.SafeL2.Number, statusB.SafeL2.Number, "chain B should have advanced")
}

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

// TestSupernodeInteropContinuousProgression tests that the interop activity
// continues to make progress over an extended period without stalling.
func TestSupernodeInteropContinuousProgression(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, 0)

	blockTime := sys.L2A.Escape().RollupConfig().BlockTime

	// Take multiple snapshots over time to verify continuous progress
	numSnapshots := 3
	snapshotInterval := time.Duration(blockTime*5) * time.Second

	type snapshot struct {
		safeA uint64
		safeB uint64
		time  time.Time
	}
	snapshots := make([]snapshot, 0, numSnapshots)

	// Wait for initial progress
	t.Require().Eventually(func() bool {
		statusA := sys.L2ACL.SyncStatus()
		statusB := sys.L2BCL.SyncStatus()
		return statusA.SafeL2.Number > 2 && statusB.SafeL2.Number > 2
	}, 60*time.Second, time.Second, "initial progress required")

	// Take snapshots
	for i := 0; i < numSnapshots; i++ {
		statusA := sys.L2ACL.SyncStatus()
		statusB := sys.L2BCL.SyncStatus()
		snapshots = append(snapshots, snapshot{
			safeA: statusA.SafeL2.Number,
			safeB: statusB.SafeL2.Number,
			time:  time.Now(),
		})

		t.Logger().Info("snapshot taken",
			"snapshot", i+1,
			"chainA_safe", statusA.SafeL2.Number,
			"chainB_safe", statusB.SafeL2.Number,
		)

		if i < numSnapshots-1 {
			time.Sleep(snapshotInterval)
		}
	}

	// Verify progress between each snapshot
	for i := 1; i < len(snapshots); i++ {
		prev := snapshots[i-1]
		curr := snapshots[i]

		t.Require().Greater(curr.safeA, prev.safeA,
			"chain A should progress between snapshot %d and %d", i, i+1)
		t.Require().Greater(curr.safeB, prev.safeB,
			"chain B should progress between snapshot %d and %d", i, i+1)
	}

	t.Logger().Info("continuous progression verified",
		"initial_A", snapshots[0].safeA,
		"final_A", snapshots[len(snapshots)-1].safeA,
		"initial_B", snapshots[0].safeB,
		"final_B", snapshots[len(snapshots)-1].safeB,
	)
}

// TestSupernodeInteropBothChainsRequired tests that interop verification
// requires both chains to have safe blocks at a timestamp before it can
// be verified. This is tested by checking that safe heads advance together.
func TestSupernodeInteropBothChainsRequired(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, 0)

	blockTime := sys.L2A.Escape().RollupConfig().BlockTime

	// Wait for both chains to advance
	t.Require().Eventually(func() bool {
		statusA := sys.L2ACL.SyncStatus()
		statusB := sys.L2BCL.SyncStatus()
		return statusA.SafeL2.Number > 5 && statusB.SafeL2.Number > 5
	}, 90*time.Second, time.Second, "both chains should advance")

	// Get current timestamps
	statusA := sys.L2ACL.SyncStatus()
	statusB := sys.L2BCL.SyncStatus()

	t.Logger().Info("both chains have advanced",
		"chainA_safe", statusA.SafeL2.Number,
		"chainA_safe_time", statusA.SafeL2.Time,
		"chainB_safe", statusB.SafeL2.Number,
		"chainB_safe_time", statusB.SafeL2.Time,
	)

	// The safe timestamps should be relatively close since interop
	// requires all chains to have data at each timestamp
	timeDiff := int64(statusA.SafeL2.Time) - int64(statusB.SafeL2.Time)
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}

	// Allow for some variance but timestamps should be within a few blocks
	maxAllowedDiff := blockTime * 3
	t.Require().LessOrEqual(uint64(timeDiff), maxAllowedDiff,
		"safe timestamps should be within %d seconds of each other", maxAllowedDiff)
}
