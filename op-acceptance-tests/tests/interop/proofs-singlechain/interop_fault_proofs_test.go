package proofs_singlechain

import (
	"testing"

	sfp "github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/superfaultproofs"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestInteropSingleChainFaultProofs(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainInterop(t)
	sfp.RunSingleChainSuperFaultProofSmokeTest(t, sys)
}
