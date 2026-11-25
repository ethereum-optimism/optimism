package cannon_kona

import (
	"testing"
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

func TestExecuteStep_CannonKona(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newSystem(t)

	l1User := sys.FunderL1.NewFundedEOA(eth.ThousandEther)
	blockNum := uint64(3)
	sys.L2CL.Reached(safety.LocalSafe, blockNum, 30)

	game := sys.DisputeGameFactory().StartCannonKonaGame(l1User, proofs.WithL2SequenceNumber(blockNum))
	claim := game.DisputeL2SequenceNumber(l1User, game.RootClaim(), blockNum)
	game.LogGameData()
	claim = claim.WaitForCounterClaim()             // Wait for the honest challenger to counter
	claim = game.DisputeToStep(l1User, claim, 1000) // Skip down to max depth
	game.LogGameData()
	claim.WaitForCountered()
	sys.AdvanceTime(game.MaxClockDuration() + time.Second)
	game.WaitForGameStatus(gameTypes.GameStatusDefenderWon)
}
