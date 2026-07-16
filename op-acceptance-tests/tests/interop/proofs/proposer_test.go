package proofs

import (
	"testing"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestProposer(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t)

	dgf := sys.DisputeGameFactory()

	newGame := dgf.WaitForGame()
	rootClaim := newGame.RootClaim().Value()
	l2SequenceNumber := newGame.L2SequenceNumber()
	sys.SuperRoots.AssertSuperRootAtTimestamp(l2SequenceNumber, rootClaim)
}

func TestSuperPermissionedProposerCreatesRepeatedGames(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t, presets.WithProposerGameType(gameTypes.SuperPermissionedGameType))

	dgf := sys.DisputeGameFactory()
	firstGame := dgf.WaitForGame()
	firstGame.VerifyGameType(gameTypes.SuperPermissionedGameType)
	firstGame.WaitForGameStatus(gameTypes.GameStatusDefenderWon)

	secondGame := dgf.WaitForGame()
	secondGame.VerifyGameType(gameTypes.SuperPermissionedGameType)
	secondGame.WaitForGameStatus(gameTypes.GameStatusDefenderWon)
}
