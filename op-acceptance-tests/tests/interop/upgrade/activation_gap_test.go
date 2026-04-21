//go:build !ci

package upgrade

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	stypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// TestActivationGap asserts that SafeL2 (CrossSafe via optimism_syncStatus)
// and FinalizedL2 never regress across the interop activation boundary on
// either chain. Boots with activation 60s out, snapshots both heads
// pre-activation, awaits activation, then polls optimism_syncStatus at 1Hz
// for 60s asserting neither head drops below its snapshot on either chain.
func TestActivationGap(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, 60)
	require := t.Require()

	interopTimeA := sys.L2A.Escape().ChainConfig().InteropTime
	interopTimeB := sys.L2B.Escape().ChainConfig().InteropTime
	require.NotNil(interopTimeA, "interop activation must be configured on chain A")
	require.NotNil(interopTimeB, "interop activation must be configured on chain B")

	upgradeTime := time.Unix(int64(*interopTimeA), 0)
	if deadline, hasDeadline := t.Deadline(); hasDeadline {
		t.Gate().True(upgradeTime.Add(90*time.Second).Before(deadline),
			"test must have headroom past activation")
	}

	dsl.CheckAll(t,
		sys.L2ACL.AdvancedFn(stypes.CrossSafe, 3, 60),
		sys.L2BCL.AdvancedFn(stypes.CrossSafe, 3, 60),
	)

	preA := sys.L2ACL.SyncStatus()
	preB := sys.L2BCL.SyncStatus()
	require.Greater(preA.SafeL2.Number, uint64(0),
		"chain A CrossSafe must be advancing pre-activation")
	require.Greater(preB.SafeL2.Number, uint64(0),
		"chain B CrossSafe must be advancing pre-activation")

	t.Logger().Info("pre-activation snapshot",
		"A.CrossSafe", preA.SafeL2.Number, "A.Finalized", preA.FinalizedL2.Number,
		"B.CrossSafe", preB.SafeL2.Number, "B.Finalized", preB.FinalizedL2.Number)

	activationA := sys.L2A.AwaitActivation(t, forks.Interop)
	activationB := sys.L2B.AwaitActivation(t, forks.Interop)
	t.Logger().Info("activation reached", "A", activationA, "B", activationB)

	pollDeadline := time.Now().Add(60 * time.Second)
	samples := 0
	for time.Now().Before(pollDeadline) {
		a := sys.L2ACL.SyncStatus()
		b := sys.L2BCL.SyncStatus()

		require.GreaterOrEqualf(a.SafeL2.Number, preA.SafeL2.Number,
			"chain A CrossSafe regressed from %d to %d across activation",
			preA.SafeL2.Number, a.SafeL2.Number)
		require.GreaterOrEqualf(b.SafeL2.Number, preB.SafeL2.Number,
			"chain B CrossSafe regressed from %d to %d across activation",
			preB.SafeL2.Number, b.SafeL2.Number)
		require.GreaterOrEqualf(a.FinalizedL2.Number, preA.FinalizedL2.Number,
			"chain A Finalized regressed from %d to %d across activation",
			preA.FinalizedL2.Number, a.FinalizedL2.Number)
		require.GreaterOrEqualf(b.FinalizedL2.Number, preB.FinalizedL2.Number,
			"chain B Finalized regressed from %d to %d across activation",
			preB.FinalizedL2.Number, b.FinalizedL2.Number)

		samples++
		time.Sleep(1 * time.Second)
	}
	t.Logger().Info("monotonicity window completed", "samples", samples)

	dsl.CheckAll(t,
		sys.L2ACL.AdvancedFn(stypes.CrossSafe, 3, 60),
		sys.L2BCL.AdvancedFn(stypes.CrossSafe, 3, 60),
	)
}
