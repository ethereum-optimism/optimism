package privateinterop

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum/go-ethereum/common"
)

func TestPrivatePublicationResumesAfterL1Reorg(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeLightSequencerInterop(t, 0,
		presets.WithDeployerOptions(sysgo.WithSequencingWindow(10)),
		presets.WithPrivateInteropChain(sysgo.WithoutRenderingInvariantCheck()))

	sys.L2BCL.Advanced(safety.CrossSafe, 20, 120)
	finalized := sys.L2ELB.BlockRefByLabel(eth.Finalized)
	sys.L1CL.Stop()
	parent := sys.L1EL.BlockRefByLabel(eth.Unsafe)

	// Freeze L1 finality while building a suffix whose origins will be removed.
	for range 4 {
		sys.TestSequencer.SequenceBlock(t, sys.L1Network.ChainID(), common.Hash{})
		sys.L2B.WaitForBlock()
		sys.L2B.WaitForBlock()
	}
	divergence := sys.L1EL.BlockRefByNumber(parent.Number + 1)
	private := sys.L2ELB.WaitForL1Origin(divergence.Number)

	sys.TestSequencer.SequenceBlock(t, sys.L1Network.ChainID(), parent.Hash)
	sys.L1CL.Start()
	sys.L1EL.ReorgTriggered(divergence, 10)
	sys.L2BCL.Reached(safety.CrossSafe, private.Number+1, 180)
	sys.L2ELB.VerifyReorgRecovery(sys.L1EL, private)
	t.Require().True(sys.L2ELB.IsCanonical(finalized.ID()), "private finalized history must survive recovery")

	// A newly submitted private transfer must become claim-backed safe again.
	// Rebuilding an unsafe tip alone does not demonstrate publication recovery.
	alice := sys.FunderB.NewFundedEOA(eth.OneEther)
	transfer := alice.Transfer(common.Address{0xee}, eth.OneGWei)
	included, err := transfer.Included.Eval(t.Ctx())
	t.Require().NoError(err)
	sys.L2BCL.ReachedRef(safety.CrossSafe, eth.BlockID{
		Hash: included.BlockHash, Number: bigs.Uint64Strict(included.BlockNumber),
	}, 180)
}
