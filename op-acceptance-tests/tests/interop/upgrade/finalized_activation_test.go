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
// The 240s activation delay must exceed the FakePoS L1 finality lag (~120s)
// so the pre-activation FinalizedL2 baseline is non-zero.
func TestFinalizedL2DoesNotRegressAcrossActivation(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, 240)
	require := t.Require()

	interopTimeA := sys.L2A.Escape().ChainConfig().InteropTime
	interopTimeB := sys.L2B.Escape().ChainConfig().InteropTime
	require.NotNil(interopTimeA, "L2A must have InteropTime configured")
	require.NotNil(interopTimeB, "L2B must have InteropTime configured")

	dsl.CheckAll(t,
		sys.L2ACL.ReachedFn(safety.Finalized, 1, 120),
		sys.L2BCL.ReachedFn(safety.Finalized, 1, 120),
	)

	// +1 so the head must be strictly past activation.
	dsl.CheckAll(t,
		sys.L2ACL.ReachedTimeWithoutRegressionFn(safety.Finalized, *interopTimeA+1),
		sys.L2BCL.ReachedTimeWithoutRegressionFn(safety.Finalized, *interopTimeB+1),
	)
}
