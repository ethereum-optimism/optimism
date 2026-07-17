package disputemon

import (
	"testing"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestDisputeMonitorCountsSuperPermissionedGames(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleChainInterop(t, presets.WithoutHonestProposer())

	sys.DisputeGameFactory().StartSuperPermissionedGame()

	mon := sys.StartDisputeMon()
	mon.VerifyGameCount(gameTypes.SuperPermissionedGameType, 1)
}
