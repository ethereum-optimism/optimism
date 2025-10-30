package cannon

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"
)

func TestExecuteStep(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimal(t)

	l1User := sys.FunderL1.NewFundedEOA(eth.ThousandEther)
	blockNum := uint64(3)
	sys.L2CL.Reached(types.LocalSafe, blockNum, 30)

	game := sys.DisputeGameFactory().StartCannonGame(l1User)
	claim := game.DisputeToL2SequenceNumber(l1User, game.RootClaim(), blockNum)
	game.LogGameData()
	claim = claim.Attack(l1User, common.Hash{0x01, 0xba, 0xd0})
	claim = claim.WaitForCounterClaim()             // Wait for the honest challenger to counter
	claim = game.DisputeToStep(l1User, claim, 1000) // Skip down to max depth
	game.LogGameData()
	claim.WaitForCountered()
}
