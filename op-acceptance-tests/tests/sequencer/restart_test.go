package sequencer

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// TestSequencerResumesAfterRestart checks that the devstack sequencer builds
// blocks again after its rollup node restarts. A kona-node sequencer always
// boots stopped, so the harness must start it after every start of the node,
// to match an op-node sequencer, which boots active.
func TestSequencerResumesAfterRestart(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimal(t)

	sys.L2CL.AdvancedUnsafe(2, 30)
	sys.L2CL.Stop()
	sys.L2CL.Start()
	t.Require().True(sys.L2CL.SequencerActive(), "sequencer must be active after a restart")
	sys.L2CL.AdvancedUnsafe(2, 30)
}
