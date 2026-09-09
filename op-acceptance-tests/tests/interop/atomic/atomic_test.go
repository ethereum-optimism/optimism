package atomic

import (
	"testing"

	sfp "github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/superfaultproofs"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestAtomicSynchronousCalls(gt *testing.T) {
	for _, scenario := range []sfp.AtomicCallScenario{sfp.AtomicCallsSucceed, sfp.AtomicRemoteReverts, sfp.AtomicOrphanedRemote} {
		gt.Run(string(scenario), func(gt *testing.T) {
			t := devtest.SerialT(gt)
			sys := presets.NewTwoL2SupernodeInterop(t, 0)
			sfp.RunAtomicCallVerificationTest(t, sys, scenario)
		})
	}
}
