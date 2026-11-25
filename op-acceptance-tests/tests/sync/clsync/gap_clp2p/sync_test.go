package gap_clp2p

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// TestSyncAfterInitialELSync tests that blocks received out of order would be processed in order when running in CL sync mode. Note that this is not going to happen when running in EL sync mode, which relies on healthy ELP2P, something that is disabled in this test.
func TestSyncAfterInitialELSync(gt *testing.T) {
	t := devtest.ParallelT(gt)
	// Example error with kona-node:
	//
	//  assertions.go:387:  ERROR[03-31|10:38:08.992]
	// "\n\tError Trace:\t/optimism/op-devstack/dsl/l2_el.go:192\n\t
	// \t\t\t\t/optimism/op-acceptance-tests/tests/sync/clsync/gap_clp2p/sync_test.go:46
	// \n\tError:
	// \tReceived unexpected error:\n\t
	// \toperation failed permanently after 2 attempts: expected head for label=latest to advance to target=5,
	// but got current=2
	// \tTest:      \tTestSyncAfterInitialELSync\n"

	sysgo.SkipOnKonaNode(t, "not supported")
	sys := newGapCLP2PSystem(t)
	require := t.Require()

	sys.L2CL.Advanced(safety.LocalUnsafe, 7, 30)

	// batcher down so safe not advanced
	require.Zero(sys.L2CL.HeadBlockRef(safety.LocalSafe).Number)
	require.Zero(sys.L2CLB.HeadBlockRef(safety.LocalSafe).Number)

	startNum := sys.L2CLB.HeadBlockRef(safety.LocalUnsafe).Number

	// Finish EL sync by supplying the first block
	// EL Sync finished because underlying EL has states to validate the payload for block startNum+1
	sys.L2CLB.SignalTarget(sys.L2EL, startNum+1)
	sys.L2ELB.WaitForBlockNumber(startNum + 1)

	// Send payloads for block startNum+3, startNum+4, startNum+5, startNum+7 which will fill in unsafe payload queue, block startNum+2, and block startNum+6 missed
	// Non-canonical payloads will be not sent to L2EL
	// Order does not matter
	for _, delta := range []uint64{5, 3, 4, 7} {
		target := startNum + delta
		sys.L2CLB.SignalTarget(sys.L2EL, target)
		// Canonical unsafe head never advances because of the gap
		require.Equal(startNum+1, sys.L2ELB.BlockRefByLabel(eth.Unsafe).Number)
	}

	// Send missing gap, payload startNum+2, still not sending FCU since unsafe gap exists
	sys.L2CLB.SignalTarget(sys.L2EL, startNum+2)

	retries := 2
	// Gap filled and payload startNum+2, startNum+3, startNum+4, startNum+5 became canonical by relaying to ELB.
	// Payload startNum+7 is still in the unsafe payload queue because of unsafe gap
	sys.L2ELB.Reached(eth.Unsafe, startNum+5, retries)

	// Send missing gap, payload startNum+6
	sys.L2CLB.SignalTarget(sys.L2EL, startNum+6)

	// Gap filled and block startNum+6, startNum+7 became canonical by relaying to ELB.
	sys.L2ELB.Reached(eth.Unsafe, startNum+7, retries)
}
