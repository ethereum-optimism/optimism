package conductor

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// TestSequencerStopsAndRestartsAtPreviousHead verifies an operator can stop
// block production and later resume from the exact unsafe head returned by the
// stop operation.
func TestSequencerStopsAndRestartsAtPreviousHead(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnKonaNode(t, "kona-node admin RPC compatibility is tracked by #21906")

	sys := presets.NewMinimalNoFaultProofs(t)
	sys.L2CL.AdvancedUnsafe(1, 30)

	stoppedAt := sys.L2CL.StopSequencer()
	sys.L2CL.NotAdvancedUnsafe(2)

	t.Require().NoError(sys.L2CL.StartSequencerAt(stoppedAt))
	sys.L2CL.AwaitSequencerActive()
	sys.L2CL.AdvancedUnsafe(1, 30)
}
