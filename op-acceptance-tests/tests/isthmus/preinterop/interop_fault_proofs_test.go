package preinterop

import (
	"testing"

	sfp "github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/superfaultproofs"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestPreinteropFaultProofs(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t)
	sfp.RunSuperFaultProofTest(t, sys)
}

func TestPreinteropFaultProofs_TraceExtensionActivation(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t)
	sfp.RunTraceExtensionActivationTest(t, sys)
}
