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

	dgf := sys.DisputeGameFactory()
	dgf.StartSuperPermissionedGame(sys.L1Proposer)

	mon := presets.StartDisputeMon(
		t,
		sys.L1EL,
		dgf.Address(),
		presets.WithDisputeMonSupernodes(sys.SuperRoots),
	)
	mon.VerifyGameCount(gameTypes.SuperPermissionedGameType, 1)
}
