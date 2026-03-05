package preinterop

import (
	"testing"

	sfp "github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/superfaultproofs"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func TestPreinteropFaultProofs(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t,
		presets.WithIsthmusSuperSupernode(),
		stack.MakeCommon(sysgo.WithChallengerCannonKonaEnabled()),
	)
	sfp.RunSuperFaultProofTest(t, sys)
}

func TestPreinteropFaultProofs_TraceExtensionActivation(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t,
		presets.WithIsthmusSuperSupernode(),
		stack.MakeCommon(sysgo.WithChallengerCannonKonaEnabled()),
	)
	sfp.RunTraceExtensionActivationTest(t, sys)
}

func TestPreinteropFaultProofs_UnsafeProposal(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t,
		presets.WithIsthmusSuperSupernode(),
		stack.MakeCommon(sysgo.WithChallengerCannonKonaEnabled()),
	)
	sfp.RunUnsafeProposalTest(t, sys)
}
