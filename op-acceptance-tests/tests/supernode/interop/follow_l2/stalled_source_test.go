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

	// maxGap sits strictly below the 10s follow-source call timeout (the
	// guaranteed production dead window when follow requests block the driver
	// loop) and well above the healthy block cadence. The 30s window spans
	// several follow ticks, so a stalled request is guaranteed to be in flight
	// while we observe.
	sys.L2ELA.KeptAdvancing(eth.Unsafe, 30*time.Second, 8*time.Second)

	t.Require().NotZero(proxies.ForChain(t, sys.L2A.ChainID()).StalledRequests(),
		"no follow-source request was held by the stalled proxy; the stall did not exercise the follow path")
}
