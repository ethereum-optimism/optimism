//go:build !ci

package upgrade

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"

	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// TestFinalizedL2DoesNotRegressAcrossActivation reproduces #20365.
// The FinalizedL2 baseline is taken while LocalSafe is still before
// activation, so the watch spans the whole window where #20365 dropped
// FinalizedL2 to genesis. Time travel finalizes the baseline in seconds
// instead of the FakePoS L1 finality lag (20 blocks); the 150s activation
// delay leaves the rest of the old margin for slow setup. The crossing itself
// waits for L1 finality in real time.
func TestFinalizedL2DoesNotRegressAcrossActivation(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, 150, presets.WithTimeTravelEnabled())
	require := t.Require()

	interopTimeA := sys.L2A.Escape().RollupConfig().LagoonTime
	interopTimeB := sys.L2B.Escape().RollupConfig().LagoonTime
	require.NotNil(interopTimeA, "L2A must have LagoonTime configured")
	require.NotNil(interopTimeB, "L2B must have LagoonTime configured")

	// Once both chains have a block on L1, jump L1 past FakePoS finality
	// (20 blocks, plus 5 of margin) so that block finalizes.
	dsl.CheckAll(t,
		sys.L2ACL.ReachedFn(safety.LocalSafe, 1, 60),
		sys.L2BCL.ReachedFn(safety.LocalSafe, 1, 60),
	)
	sys.AdvanceTime(25 * sys.L1CL.BlockTime())

	dsl.CheckAll(t,
		sys.L2ACL.ReachedFn(safety.Finalized, 1, 120),
		sys.L2BCL.ReachedFn(safety.Finalized, 1, 120),
	)
	require.Less(sys.L2ACL.HeadBlockRef(safety.LocalSafe).Time, *interopTimeA,
		"L2A LocalSafe crossed activation before the finalized baseline; setup was slower than the activation delay")
	require.Less(sys.L2BCL.HeadBlockRef(safety.LocalSafe).Time, *interopTimeB,
		"L2B LocalSafe crossed activation before the finalized baseline; setup was slower than the activation delay")

	// +1 so the head must be strictly past activation.
	dsl.CheckAll(t,
		sys.L2ACL.ReachedTimeWithoutRegressionFn(safety.Finalized, *interopTimeA+1),
		sys.L2BCL.ReachedTimeWithoutRegressionFn(safety.Finalized, *interopTimeB+1),
	)
}
