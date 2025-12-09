package outputs

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestTopGame(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimal(t)

	l1User := sys.FunderL1.NewFundedEOA(eth.ThousandEther)
	blockNum := sys.L2CL.SafeHead().BlockRef.Number

	game := sys.DisputeGameFactory().StartCannonGame(l1User, proofs.WithL2SequenceNumber(blockNum))
	claim := game.DisputeL2SequenceNumber(l1User, game.RootClaim(), blockNum)
	game.LogGameData()
	_ = claim.WaitForCounterClaim() // Wait for the honest challenger to counter
}
