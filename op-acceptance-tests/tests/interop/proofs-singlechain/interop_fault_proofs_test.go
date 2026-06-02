package proofs_singlechain

import (
	"testing"

	sfp "github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/superfaultproofs"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func TestInteropSingleChainFaultProofs(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainInterop(t)
	sfp.RunSingleChainSuperFaultProofSmokeTest(t, sys)
}

func TestInteropSingleChainFaultProofsWithSDM(gt *testing.T) {
	t := devtest.SerialT(gt)
	sysgo.SkipOnOpGeth(t, "SDM PostExec is op-reth only")

	sys := presets.NewSingleChainInterop(t)
	err := sys.L2ELA.Escape().L2EthClient().RPC().CallContext(t.Ctx(), nil, "admin_setSdmPostExecOptIn", true)
	t.Require().NoError(err, "admin_setSdmPostExecOptIn(true) RPC failed")
	sfp.RunSingleChainSuperFaultProofSDMSmokeTest(t, sys)
}
