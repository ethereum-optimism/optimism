package follow_l2

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// TestFollowL2_FollowerWaitsForStalledSource checks that L2CLC takes its local-safe and finalized
// heads only from its follow source: while the source's RPC is stalled, both heads stay put though
// L2CLB keeps advancing them, so L2CLC is not deriving from L1. Once the source answers again,
// L2CLC catches up.
func TestFollowL2_FollowerWaitsForStalledSource(gt *testing.T) {
	forEachFollowSource(gt, followL2FollowerWaitsForStalledSource)
}

func followL2FollowerWaitsForStalledSource(t devtest.T, source followSource) {
	proxies := sysgo.NewStallableFollowSourceProxies()
	sys := newSingleChainTwoVerifiersFollowL2(t, source, presets.WithGlobalL2CLOption(proxies.L2CLOption()))
	require := t.Require()
	proxy := proxies.ForChain(t, sys.L2CLC.ChainID())

	// L2CLC follows its source past genesis first, so heads that stay put during the stall cannot
	// just be heads that never moved.
	require.NoError(sys.L2CLC.ReachedWithProgressFn(
		safety.Finalized, safety.LocalSafe, 1,
		5*time.Minute, time.Minute,
	)())

	proxy.Stall()
	// Both CLs keep one follow-source request in flight, so a second stalled request means the first
	// timed out. op-node applied any earlier response before sending the first; kona-node has had a
	// full request timeout to apply what it forwarded to its engine.
	proxy.WaitForStalledRequests(t, 2)
	stalledSafe := sys.L2CLC.HeadBlockRef(safety.LocalSafe)
	stalledFinalized := sys.L2CLC.HeadBlockRef(safety.Finalized)

	// L2CLB kept advancing while the stall was confirmed, so targets past L2CLC's heads can already
	// be met. Make L2CLB advance from its own heads, so that the stall spans real progress: about an
	// L1 block of batches for local-safe, and any finality step for finalized.
	sourceFinalized := sys.L2CLB.HeadBlockRef(safety.Finalized)
	dsl.CheckAll(t,
		sys.L2CLB.AdvancedFn(safety.LocalSafe, 3, 30),
		sys.L2CLB.ReachedWithProgressFn(
			safety.Finalized, safety.LocalSafe, sourceFinalized.Number+1,
			5*time.Minute, time.Minute,
		),
	)
	require.Equal(stalledSafe, sys.L2CLC.HeadBlockRef(safety.LocalSafe),
		"local-safe head moved while the follow source was stalled")
	require.Equal(stalledFinalized, sys.L2CLC.HeadBlockRef(safety.Finalized),
		"finalized head moved while the follow source was stalled")

	proxy.Resume()
	attempts := 30
	dsl.CheckAll(t,
		sys.L2CLC.InSyncFn(sys.L2CLB, safety.LocalSafe, attempts),
		sys.L2CLC.InSyncFn(sys.L2CLB, safety.Finalized, attempts),
	)
}
