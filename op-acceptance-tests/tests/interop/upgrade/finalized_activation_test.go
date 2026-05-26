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
//
// During interop activation, `optimism_syncStatus.finalized_l2` could briefly
// snap back to genesis (block 0) before recovering. Once LocalSafeL2 crosses
// activation, super_authority stops falling back to local-finalized, but the
// verifier has nothing useful to say about finalization until its first
// post-activation entry is both written and L1-finalized. In that gap, the
// layer above interprets the verifier's empty answer as "no information" and
// resolves FinalizedL2 to genesis.
//
// The test trap, also documented in #20365: the FakePoS preset finalizes L1
// with a 20-block lag (~120s at the default 6s L1 block time). If the
// activation delay sits inside that lag, the pre-activation FinalizedL2
// snapshot is itself 0, and a naive `now >= before` monotonicity check passes
// vacuously even when FinalizedL2 regresses to genesis — 0 >= 0. To avoid this:
//
//  1. Use an activation delay (240s) that comfortably exceeds the FakePoS L1
//     finality lag (120s), so a real post-genesis block becomes L1-finalized
//     before activation lands. A future preset change that shrinks the
//     activation delay back into the finality window would re-vacuum the
//     assertion; the explicit value here is the load-bearing precondition.
//  2. Wait for FinalizedL2 > 0 on every chain before snapshotting the
//     baseline. This guarantees the monotonicity check has something real to
//     compare against.
//  3. Drive monitoring with the interop activation timestamp as the explicit
//     target. ReachedTimeWithoutRegressionFn refuses to run with a zero
//     baseline, refuses to run if the head is already past the boundary, and
//     fails immediately on any regression below the baseline while waiting
//     for the head to cross activation.
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

	// +1 so the head must be strictly past activation, not merely at the
	// activation block. Stopping at the activation block could miss the
	// regression window that opens once LocalSafeL2 crosses the boundary.
	dsl.CheckAll(t,
		sys.L2ACL.ReachedTimeWithoutRegressionFn(safety.Finalized, *interopTimeA+1),
		sys.L2BCL.ReachedTimeWithoutRegressionFn(safety.Finalized, *interopTimeB+1),
	)
}
