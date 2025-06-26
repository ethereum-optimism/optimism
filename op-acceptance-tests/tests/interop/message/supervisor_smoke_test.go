package msg

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// TestInteropSystemSupervisor tests that the supervisor can provide finalized L1 block information
func TestInteropSystemSupervisor(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t)

	sys.L1Network.WaitForFinalization()

	// Get the finalized L1 block from the supervisor
	block, err := sys.Supervisor.Escape().QueryAPI().FinalizedL1(t.Ctx())
	t.Require().NoError(err)

	// If we get here, the supervisor has finalized L1 block information
	t.Require().NotNil(block)
	t.Log("finalized l1 block", "block", block)
}
