package proofs_singlechain

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/sdm/sdmtest"
	sfp "github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/superfaultproofs"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestInteropSingleChainFaultProofs(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainInterop(t)
	sfp.RunSingleChainSuperFaultProofSmokeTest(t, sys)
}

func TestInteropSingleChainFaultProofsWithSDM(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := sdmtest.NewFixtureSingleChainFaultProofSystem(t)
	sfp.RunSingleChainSuperFaultProofSDMSmokeTest(t, sys)
}
