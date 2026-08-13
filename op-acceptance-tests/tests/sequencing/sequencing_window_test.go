package sequencing_test

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/sequencing"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// TestSequencingWindowExpiryNativeCL runs the sequencing-window expiry and recovery scenario
// against the native CL selected by DEVSTACK_L2CL_KIND (op-node by default, or kona-node).
// The kona-node variant currently reproduces https://github.com/ethereum-optimism/optimism/issues/20279.
func TestSequencingWindowExpiryNativeCL(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimalNoFaultProofs(t, sequencing.SequencingWindowExpiryOptions()...)

	sequencing.RunSequencingWindowExpiry(t, sequencing.SequencingWindowSystem{
		L1Network: sys.L1Network,
		L1EL:      sys.L1EL,
		L2Network: sys.L2Chain,
		L2CL:      sys.L2CL,
		L2EL:      sys.L2EL,
		L2Batcher: sys.L2Batcher,
		L2Funder:  sys.FunderL2,
	})
}
