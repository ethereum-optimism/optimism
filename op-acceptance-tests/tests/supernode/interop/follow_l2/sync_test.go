package follow_l2

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestFollowSource_LocalSafeDivergesThenConverges(gt *testing.T) {
	t := devtest.SerialT(gt)
	require := t.Require()
	sys := presets.NewTwoL2SupernodeFollowL2(t, 0)

	type chainPair struct {
		name     string
		source   *dsl.L2CLNode
		follower *dsl.L2CLNode
	}

	chains := []chainPair{
		{name: "A", source: sys.L2ACL, follower: sys.L2AFollowCL},
		{name: "B", source: sys.L2BCL, follower: sys.L2BFollowCL},
	}

	// Initial sanity: followers are aligned with upstream on both local-safe and cross-safe.
	initialChecks := make([]dsl.CheckFunc, 0, len(chains)*2)
	for _, chain := range chains {
		initialChecks = append(initialChecks,
			chain.follower.MatchedFn(chain.source, types.LocalSafe, 20),
			chain.follower.MatchedFn(chain.source, types.CrossSafe, 20),
		)
	}
	dsl.CheckAll(t, initialChecks...)

	pausedAt := sys.Supernode.EnsureInteropPaused(sys.L2ACL, sys.L2BCL, 10)
	t.Logger().Info("interop paused", "timestamp", pausedAt)

	// While interop is paused, local-safe should continue to advance and lead cross-safe.
	require.Eventually(func() bool {
		for _, chain := range chains {
			sourceStatus := chain.source.SyncStatus()
			followerStatus := chain.follower.SyncStatus()

			if sourceStatus.LocalSafeL2.Number <= sourceStatus.SafeL2.Number+1 {
				return false
			}
			if followerStatus.LocalSafeL2.Number <= followerStatus.SafeL2.Number+1 {
				return false
			}
		}
		return true
	}, 2*time.Minute, 2*time.Second, "expected local-safe to lead cross-safe on source and follower")

	// Core follow-source checks: follower must match source local-safe and cross-safe independently.
	divergenceChecks := make([]dsl.CheckFunc, 0, len(chains)*2)
	for _, chain := range chains {
		divergenceChecks = append(divergenceChecks,
			chain.follower.MatchedFn(chain.source, types.LocalSafe, 20),
			chain.follower.MatchedFn(chain.source, types.CrossSafe, 20),
		)
	}
	dsl.CheckAll(t, divergenceChecks...)

	// Freeze new block production so interop can catch cross-safe up to local-safe.
	sys.L2ACL.StopSequencer()
	sys.L2BCL.StopSequencer()
	t.Cleanup(func() {
		sys.L2ACL.StartSequencer()
		sys.L2BCL.StartSequencer()
	})

	sys.Supernode.ResumeInterop()

	require.Eventually(func() bool {
		for _, chain := range chains {
			status := chain.follower.SyncStatus()
			if status.LocalSafeL2.Hash != status.SafeL2.Hash || status.LocalSafeL2.Number != status.SafeL2.Number {
				return false
			}
		}
		return true
	}, 3*time.Minute, 2*time.Second, "expected local-safe and cross-safe to converge on followers")

	// Final sanity: follower and source converge to the same local-safe and cross-safe heads.
	finalChecks := make([]dsl.CheckFunc, 0, len(chains)*2)
	for _, chain := range chains {
		finalChecks = append(finalChecks,
			chain.follower.MatchedFn(chain.source, types.LocalSafe, 20),
			chain.follower.MatchedFn(chain.source, types.CrossSafe, 20),
		)
	}
	dsl.CheckAll(t, finalChecks...)
}
