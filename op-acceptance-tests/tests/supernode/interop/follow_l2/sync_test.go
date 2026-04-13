package follow_l2

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestFollowSource_HeadsDivergeThenConverge(gt *testing.T) {
	t := devtest.ParallelT(gt)
	require := t.Require()
	sys := presets.NewTwoL2SupernodeFollowL2(t, 0)

	type chainPair struct {
		name     string
		source   *dsl.L2CLNode
		follower *dsl.L2CLNode
	}

	type headSnapshot struct {
		sourceUnsafe      uint64
		sourceLocalSafe   uint64
		sourceCrossSafe   uint64
		followerUnsafe    uint64
		followerLocalSafe uint64
		followerCrossSafe uint64
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

	baselines := make(map[string]headSnapshot, len(chains))
	for _, chain := range chains {
		sourceStatus := chain.source.SyncStatus()
		followerStatus := chain.follower.SyncStatus()
		baselines[chain.name] = headSnapshot{
			sourceUnsafe:      sourceStatus.UnsafeL2.Number,
			sourceLocalSafe:   sourceStatus.LocalSafeL2.Number,
			sourceCrossSafe:   sourceStatus.SafeL2.Number,
			followerUnsafe:    followerStatus.UnsafeL2.Number,
			followerLocalSafe: followerStatus.LocalSafeL2.Number,
			followerCrossSafe: followerStatus.SafeL2.Number,
		}
	}

	// While interop is paused, unsafe should advance independently ahead of local-safe, and local-safe ahead of cross-safe.
	require.Eventually(func() bool {
		for _, chain := range chains {
			baseline := baselines[chain.name]
			sourceStatus := chain.source.SyncStatus()
			followerStatus := chain.follower.SyncStatus()

			if sourceStatus.UnsafeL2.Number <= baseline.sourceUnsafe {
				return false
			}
			if followerStatus.UnsafeL2.Number <= baseline.followerUnsafe {
				return false
			}
			if sourceStatus.UnsafeL2.Number <= sourceStatus.LocalSafeL2.Number {
				return false
			}
			if followerStatus.UnsafeL2.Number <= followerStatus.LocalSafeL2.Number {
				return false
			}
			if sourceStatus.LocalSafeL2.Number <= sourceStatus.SafeL2.Number {
				return false
			}
			if followerStatus.LocalSafeL2.Number <= followerStatus.SafeL2.Number {
				return false
			}
			if sourceStatus.LocalSafeL2.Number <= baseline.sourceLocalSafe {
				return false
			}
			if followerStatus.LocalSafeL2.Number <= baseline.followerLocalSafe {
				return false
			}
			if sourceStatus.SafeL2.Number < baseline.sourceCrossSafe {
				return false
			}
			if followerStatus.SafeL2.Number < baseline.followerCrossSafe {
				return false
			}
		}
		return true
	}, 2*time.Minute, 2*time.Second, "expected unsafe > local-safe > cross-safe with unsafe advancing on source and follower")

	// Core follow-source checks: follower must match source unsafe, local-safe, and cross-safe independently.
	divergenceChecks := make([]dsl.CheckFunc, 0, len(chains)*3)
	for _, chain := range chains {
		divergenceChecks = append(divergenceChecks,
			chain.follower.MatchedFn(chain.source, types.LocalUnsafe, 20),
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

	// Final sanity: follower and source converge to the same unsafe, local-safe, and cross-safe heads.
	finalChecks := make([]dsl.CheckFunc, 0, len(chains)*3)
	for _, chain := range chains {
		finalChecks = append(finalChecks,
			chain.follower.MatchedFn(chain.source, types.LocalUnsafe, 20),
			chain.follower.MatchedFn(chain.source, types.LocalSafe, 20),
			chain.follower.MatchedFn(chain.source, types.CrossSafe, 20),
		)
	}
	dsl.CheckAll(t, finalChecks...)
}
