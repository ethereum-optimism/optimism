package gap_clp2p

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"

	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

func TestReachUnsafeTipByAppendingUnsafePayload(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newGapCLP2PSystem(t)
	logger := t.Logger()

	sys.L2CL.Advanced(safety.LocalUnsafe, 7, 30)

	// First make verifier reach unsafe tip
	logger.Info("Initial trial for appending payload until tip")
	sys.L2CLB.AppendUnsafePayloadUntilTip(sys.L2ELB, sys.L2EL, 400)

	sys.L2CL.Advanced(safety.LocalUnsafe, 7, 30)

	// Try once more to check that filling in the gap works again
	logger.Info("Second trial for appending payload until tip")
	sys.L2CLB.AppendUnsafePayloadUntilTip(sys.L2ELB, sys.L2EL, 400)
}
