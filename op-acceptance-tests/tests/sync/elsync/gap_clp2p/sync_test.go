package gap_clp2p

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestSyncAfterInitialELSync(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithoutCheck(t)
	require := t.Require()

	sys.L2CL.Advanced(types.LocalUnsafe, 7, 30)

	// batcher down so safe not advanced
	require.Equal(uint64(0), sys.L2CL.HeadBlockRef(types.LocalSafe).Number)
	require.Equal(uint64(0), sys.L2CLB.HeadBlockRef(types.LocalSafe).Number)
	// verifier not advanced unsafe head
	require.Equal(uint64(0), sys.L2CLB.HeadBlockRef(types.LocalUnsafe).Number)

	// Finish EL sync by supplying the first block
	// EL Sync finished because underlying EL has states to validate the payload for block 1
	sys.L2CLB.SignalTarget(sys.L2EL, 1)

	// Send payloads for block 3, 4, 5, 7 which will fill in unsafe payload queue, block 2 missed
	// Non-canonical payloads will be not sent to L2EL
	// Order does not matter
	for _, target := range []uint64{5, 3, 4, 7} {
		sys.L2CLB.SignalTarget(sys.L2EL, target)
		// Canonical unsafe head never advances because of the gap
		require.Equal(uint64(1), sys.L2ELB.BlockRefByLabel(eth.Unsafe).Number)
	}

	// Send missing gap, payload 2, still not sending FCU since unsafe gap exists
	sys.L2CLB.SignalTarget(sys.L2EL, 2)

	retries := 2
	// Gap filled and payload 2, 3, 4, 5 became canonical by relaying to ELB.
	// Payload 7 is still in the unsafe payload queue because of unsafe gap
	sys.L2ELB.Reached(eth.Unsafe, 5, retries)

	// Send missing gap, payload 6
	sys.L2CLB.SignalTarget(sys.L2EL, 6)

	// Gap filled and block 6, 7 became canonical by relaying to ELB.
	sys.L2ELB.Reached(eth.Unsafe, 7, retries)
}
