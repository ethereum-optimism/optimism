package msg

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// TestInteropSystemSupervisor tests that the supervisor can provide finalized L1 block information
func TestInteropSystemSupervisor(gt *testing.T) {
	gt.Skip("Skipping Interop Acceptance Test")
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t)

	// First ensure L1 network is online and has blocks
	t.Log("Waiting for L1 network to be online...")
	sys.L1Network.WaitForOnline()
	t.Log("L1 network is online")

	t.Log("Waiting for initial L1 block...")
	initialBlock := sys.L1Network.WaitForBlock()
	t.Log("Got initial L1 block", "block", initialBlock)

	// Wait for finalization (this may take some time)
	t.Log("Waiting for L1 block finalization...")
	finalizedBlock := sys.L1Network.WaitForFinalization()
	t.Log("L1 block finalized", "block", finalizedBlock)

	// Get the finalized L1 block from the supervisor
	t.Log("Querying supervisor for finalized L1 block...")
	block, err := sys.Supervisor.Escape().QueryAPI().FinalizedL1(t.Ctx())
	t.Require().NoError(err, "Failed to get finalized block from supervisor")

	// If we get here, the supervisor has finalized L1 block information
	t.Require().NotNil(block, "Supervisor returned nil finalized block")
	t.Log("Successfully got finalized L1 block from supervisor", "block", block)
}
