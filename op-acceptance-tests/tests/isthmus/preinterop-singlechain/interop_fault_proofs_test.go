package preinterop_singlechain

import (
	"testing"

	sfp "github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/superfaultproofs"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestPreinteropSingleChainFaultProofs(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleChainInteropIsthmusSuper(
		t,
		presets.WithChallengerCannonKonaEnabled(),
	)
	sfp.RunSingleChainSuperFaultProofSmokeTest(t, sys)
}
