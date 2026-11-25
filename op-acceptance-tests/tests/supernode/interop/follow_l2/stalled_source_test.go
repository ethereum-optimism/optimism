package follow_l2

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestLightSequencerKeepsSequencingWhenFollowSourceStalls covers the driver
// event loop's independence from follow-source RPC latency: when a light CL
// sequencer's follow-source endpoint becomes unresponsive, the sequencer must
// keep producing unsafe blocks at its normal cadence.
//
// The follow-source requests run on a 2*blocktime ticker and their RPC client
// has a 10s default call timeout. If the driver services those requests on its
// main event loop (the regression this test guards against), an unresponsive
// endpoint blocks the loop for the full call timeout on every tick, freezing
// block production in >=10s gaps. Run off-loop, a stalled endpoint costs
// nothing: production must show no gap anywhere near that dead window.
func TestLightSequencerKeepsSequencingWhenFollowSourceStalls(gt *testing.T) {
	t := devtest.ParallelT(gt)

	proxies := sysgo.NewStallableFollowSourceProxies()
	sys := presets.NewTwoL2SupernodeLightSequencerInterop(t, 0,
		presets.WithSupernodeVNSequencerForBootstrap(),
		presets.WithGlobalL2CLOption(proxies.L2CLOption()),
	)
	sys.BootstrapLightSequencersViaVNHandoff()

	proxies.ForChain(t, sys.L2A.ChainID()).Stall()

	// maxGap of two block intervals requires production to stay at its normal
	// cadence, not merely avoid a full stall: it fails both the 10s dead window
	// of an on-loop follow request (the client call timeout) and any degraded
	// rhythm where individual blocks slip by more than one interval. It cannot
	// be tightened much further: KeptAdvancing samples at 500ms, so a healthy
	// 2s cadence can measure as ~3s between observed advances. The 30s window
	// spans several follow ticks, so a stalled request is guaranteed to be in
	// flight while we observe.
	sys.L2ELA.KeptAdvancing(eth.Unsafe, 30*time.Second, 4*time.Second)

	proxy := proxies.ForChain(t, sys.L2A.ChainID())
	t.Require().NotZero(proxy.StalledRequests(),
		"no follow-source request was held by the stalled proxy; the stall did not exercise the follow path")
	// The follow client keeps at most one fetch in flight, coalescing ticks
	// that fire while a request is blocked. Without that single-flight
	// behavior, every 2*blocktime tick would open another request against the
	// stalled endpoint (each surviving up to its 10s call timeout).
	t.Require().EqualValues(1, proxy.MaxConcurrentStalledRequests(),
		"expected exactly one follow-source request in flight at a time; the single-flight contract is broken")
}
