package privateinterop

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestPrivateWritesArePublishedInBatches(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeLightSequencerInterop(t, 0,
		presets.WithDeployerOptions(sysgo.WithLocalContractSources()),
		presets.WithPrivateInteropChain(),
	)
	sender := sys.FunderB.NewFundedEOA(eth.OneEther)
	probe := dsl.NewPrivateWriteProbe(sender, sys.L2ELB, sys.L2BSupernodeEL)
	probe.Write(7)
	probe.RevertWrite(9)
	probe.VerifyPublished()
}
