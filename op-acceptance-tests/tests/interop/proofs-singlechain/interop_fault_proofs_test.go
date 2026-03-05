package proofs_singlechain

import (
	"testing"

	sfp "github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/superfaultproofs"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func TestInteropSingleChainFaultProofs(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainInterop(t,
		presets.WithSingleChainSuperInteropSupernode(),
		presets.WithL2NetworkCount(1),
		stack.MakeCommon(sysgo.WithChallengerCannonKonaEnabled()),
	)
	sfp.RunSingleChainSuperFaultProofSmokeTest(t, sys)
}
