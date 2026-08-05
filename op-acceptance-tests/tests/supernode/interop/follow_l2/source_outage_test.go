package follow_l2

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"

	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum/go-ethereum/common"
)

// TestFollowSourceOutageIsolation verifies that an outage of one chain's
// follow-source route does not stall or rewrite either chain. Follow-L2 nodes
// retain their independent gossip and EL P2P data paths during this control-
// plane outage; after the route resumes, every safety head must reconverge.
func TestFollowSourceOutageIsolation(gt *testing.T) {
	t := devtest.SerialT(gt)
	require := t.Require()

	proxies := sysgo.NewStallableFollowSourceProxies()
	sys := presets.NewTwoL2SupernodeFollowL2(t, 0,
		presets.WithGlobalL2CLOption(proxies.L2CLOption()),
	)

	dsl.CheckAll(t,
		sys.L2AFollowCL.InSyncFn(sys.L2ACL, safety.LocalUnsafe, 30),
		sys.L2BFollowCL.InSyncFn(sys.L2BCL, safety.LocalUnsafe, 30),
	)

	proxyA := proxies.ForChain(t, sys.L2A.ChainID())
	proxyA.Stall()
	t.Cleanup(proxyA.Resume)
	require.Eventually(func() bool {
		return proxyA.StalledRequests() > 0
	}, 30*time.Second, 250*time.Millisecond,
		"chain A follower never exercised its stalled follow-source route")

	sourceABefore := sys.L2ACL.HeadBlockRef(safety.LocalUnsafe)
	followerABefore := sys.L2AFollowCL.HeadBlockRef(safety.LocalUnsafe)
	sourceBBefore := sys.L2BCL.HeadBlockRef(safety.LocalUnsafe)
	followerBBefore := sys.L2BFollowCL.HeadBlockRef(safety.LocalUnsafe)
	targetA := max(sourceABefore.Number, followerABefore.Number) + 4
	targetB := max(sourceBBefore.Number, followerBBefore.Number) + 4

	dsl.CheckAll(t,
		sys.L2ACL.ReachedFn(safety.LocalUnsafe, targetA, 30),
		sys.L2AFollowCL.ReachedFn(safety.LocalUnsafe, targetA, 30),
		sys.L2BCL.ReachedFn(safety.LocalUnsafe, targetB, 30),
		sys.L2BFollowCL.ReachedFn(safety.LocalUnsafe, targetB, 30),
	)

	sourceAAfter := sys.L2ACL.HeadBlockRef(safety.LocalUnsafe)
	followerAAfter := sys.L2AFollowCL.HeadBlockRef(safety.LocalUnsafe)
	sourceBAfter := sys.L2BCL.HeadBlockRef(safety.LocalUnsafe)
	followerBAfter := sys.L2BFollowCL.HeadBlockRef(safety.LocalUnsafe)
	firstCanonicalA := max(sourceABefore.Number, followerABefore.Number) + 1
	lastCanonicalA := min(sourceAAfter.Number, followerAAfter.Number)
	canonicalA := make(map[uint64]common.Hash, lastCanonicalA-firstCanonicalA+1)
	for height := firstCanonicalA; height <= lastCanonicalA; height++ {
		sourceRef := sys.L2ELA.BlockRefByNumber(height)
		followerRef := sys.L2AFollowEL.BlockRefByNumber(height)
		require.Equal(sourceRef.Hash, followerRef.Hash,
			"chain A follower disagreed with its source at outage height %d", height)
		canonicalA[height] = followerRef.Hash
	}

	firstCanonicalB := max(sourceBBefore.Number, followerBBefore.Number) + 1
	lastCanonicalB := min(sourceBAfter.Number, followerBAfter.Number)
	canonicalB := make(map[uint64]common.Hash, lastCanonicalB-firstCanonicalB+1)
	for height := firstCanonicalB; height <= lastCanonicalB; height++ {
		sourceRef := sys.L2ELB.BlockRefByNumber(height)
		followerRef := sys.L2BFollowEL.BlockRefByNumber(height)
		require.Equal(sourceRef.Hash, followerRef.Hash,
			"chain B follower disagreed with its source at outage height %d", height)
		canonicalB[height] = followerRef.Hash
	}

	proxyA.Resume()
	dsl.CheckAll(t,
		sys.L2AFollowCL.InSyncFn(sys.L2ACL, safety.LocalUnsafe, 60),
		sys.L2AFollowCL.InSyncFn(sys.L2ACL, safety.LocalSafe, 60),
		sys.L2AFollowCL.InSyncFn(sys.L2ACL, safety.CrossSafe, 60),
		sys.L2BFollowCL.InSyncFn(sys.L2BCL, safety.LocalUnsafe, 60),
		sys.L2BFollowCL.InSyncFn(sys.L2BCL, safety.LocalSafe, 60),
		sys.L2BFollowCL.InSyncFn(sys.L2BCL, safety.CrossSafe, 60),
	)

	for height, expected := range canonicalA {
		require.Equal(expected, sys.L2ELA.BlockRefByNumber(height).Hash,
			"chain A source rewrote outage height %d", height)
		require.Equal(expected, sys.L2AFollowEL.BlockRefByNumber(height).Hash,
			"chain A follower rewrote outage height %d", height)
	}
	for height, expected := range canonicalB {
		require.Equal(expected, sys.L2ELB.BlockRefByNumber(height).Hash,
			"chain B source rewrote outage height %d", height)
		require.Equal(expected, sys.L2BFollowEL.BlockRefByNumber(height).Hash,
			"chain B follower rewrote outage height %d", height)
	}

	require.Greater(proxyA.StalledRequests(), int64(0))
	require.LessOrEqual(proxyA.MaxConcurrentStalledRequests(), int64(1),
		"chain A follower opened concurrent follow-source requests during the outage")
}
