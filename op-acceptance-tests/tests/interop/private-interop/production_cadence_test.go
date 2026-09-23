package privateinterop

import (
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum/go-ethereum/common"
)

// This wall-clock soak takes about twenty minutes. Keep the normal acceptance
// suite fast; operators enable it explicitly before a deployment.
func TestPrivatePublicationCatchesUpAtProductionCadence(gt *testing.T) {
	if os.Getenv("PRIVATE_INTEROP_SOAK") != "1" {
		gt.Skip("set PRIVATE_INTEROP_SOAK=1 for the production-cadence soak")
	}
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeLightSequencerInterop(t, 0,
		presets.WithDeployerOptions(sysgo.WithSequencingWindow(3600)),
		presets.WithPrivateInteropChain(sysgo.WithoutRenderingInvariantCheck(), sysgo.WithPrivateInteropCadence(300)))
	sys.L2BatcherB.Stop()
	alice := sys.FunderB.NewFundedEOA(eth.OneEther)
	transfer := alice.Transfer(common.Address{0xea}, eth.OneGWei)
	rec, err := transfer.Included.Eval(t.Ctx())
	t.Require().NoError(err)
	included := eth.BlockID{Hash: rec.BlockHash, Number: bigs.Uint64Strict(rec.BlockNumber)}
	// Accumulate two complete ranges while L1 and private sequencing continue.
	// This exercises catch-up from retained private execution, not prebuilt payloads.
	sys.L2BCL.Reached(safety.LocalUnsafe, 600, 1400)
	sys.L2BatcherB.Start()
	sys.L2BCL.Reached(safety.CrossSafe, 600, 240)
	t.Require().True(sys.L2ELB.IsCanonical(included), "publication must preserve the private transfer")
	t.Require().GreaterOrEqual(sys.L2ELB.BlockRefByLabel(eth.Safe).Number, uint64(600))
}
