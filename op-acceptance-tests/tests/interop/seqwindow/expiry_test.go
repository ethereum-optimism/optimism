package seqwindow

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/sequencing"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// TestSequencingWindowExpiry runs the sequencing-window expiry and recovery scenario against an
// op-supernode virtual node.
func TestSequencingWindowExpiry(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, 0, sequencing.SequencingWindowExpiryOptions()...)

	sequencing.RunSequencingWindowExpiry(t, sequencing.SequencingWindowSystem{
		L1Network: sys.L1Network,
		L1EL:      sys.L1EL,
		L2Network: sys.L2A,
		L2CL:      sys.L2ACL,
		L2EL:      sys.L2ELA,
		L2Batcher: sys.L2BatcherA,
		L2Funder:  sys.FunderA,
	})
}
