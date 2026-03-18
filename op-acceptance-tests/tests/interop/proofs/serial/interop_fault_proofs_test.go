package serial

import (
	"testing"

	sfp "github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/superfaultproofs"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestInteropFaultProofs(gt *testing.T) {
	t := devtest.SerialT(gt)
	// TODO(#19180): Unskip this once supernode is updated.
	t.Skip("Supernode does not yet return optimistic blocks until blocks are fully validated")
	sys := presets.NewSimpleInteropSupernodeProofs(t, presets.WithChallengerCannonKonaEnabled())
	sfp.RunSuperFaultProofTest(t, sys)
}

func TestInteropFaultProofs_ConsolidateValidCrossChainMessage(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInteropSupernodeProofs(t, presets.WithChallengerCannonKonaEnabled())
	sfp.RunConsolidateValidCrossChainMessageTest(t, sys)
}
