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

	startNum := sys.L2CLB.HeadBlockRef(types.LocalUnsafe).Number

	// Finish EL sync by supplying the first block
	// EL Sync finished because underlying EL has states to validate the payload for block startNum+1
	sys.L2CLB.SignalTarget(sys.L2EL, startNum+1)

	// Send payloads for block startNum+3, startNum+4, startNum+5, startNum+7 which will fill in unsafe payload queue, block startNum+2 missed
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

func TestReachUnsafeTipByAppendingUnsafePayload(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithoutCheck(t)
	logger := t.Logger()

	sys.L2CL.Advanced(types.LocalUnsafe, 7, 30)

	// First make verifier reach unsafe tip
	logger.Info("Initial trial for appending payload until tip")
	sys.L2CLB.AppendUnsafePayloadUntilTip(sys.L2ELB, sys.L2EL, 400)

	sys.L2CL.Advanced(types.LocalUnsafe, 7, 30)

	// Try once more to check that filling in the gap works again
	logger.Info("Second trial for appending payload until tip")
	sys.L2CLB.AppendUnsafePayloadUntilTip(sys.L2ELB, sys.L2EL, 400)
}
