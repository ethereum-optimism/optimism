package serial

import (
	"testing"

	sfp "github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/superfaultproofs"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func TestInteropFaultProofs(gt *testing.T) {
	t := devtest.SerialT(gt)
	// TODO(#19180): Unskip this once supernode is updated.
	t.Skip("Supernode does not yet return optimistic blocks until blocks are fully validated")
	sys := presets.NewSimpleInterop(t,
		presets.WithSuperInteropSupernode(),
		stack.MakeCommon(sysgo.WithChallengerCannonKonaEnabled()),
	)
	sfp.RunSuperFaultProofTest(t, sys)
}

func TestInteropFaultProofs_ConsolidateValidCrossChainMessage(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t,
		presets.WithSuperInteropSupernode(),
		stack.MakeCommon(sysgo.WithChallengerCannonKonaEnabled()),
	)
	sfp.RunConsolidateValidCrossChainMessageTest(t, sys)
}
