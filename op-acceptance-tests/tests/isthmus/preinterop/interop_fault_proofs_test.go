package preinterop

import (
	"testing"

	sfp "github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/superfaultproofs"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

func TestPreinteropFaultProofs(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newSimpleInteropPreinterop(t)
	sfp.RunSuperFaultProofTest(t, sys)
}

func TestPreinteropFaultProofs_TraceExtensionActivation(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newSimpleInteropPreinterop(t)
	sfp.RunTraceExtensionActivationTest(t, sys)
}

func TestPreinteropFaultProofs_UnsafeProposal(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newSimpleInteropPreinterop(t)
	sfp.RunUnsafeProposalTest(t, sys)
}
