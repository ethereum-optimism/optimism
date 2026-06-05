package cannon_kona

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/gameargs"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

func TestSmoke(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newSystem(t)
	require := t.Require()
	dgf := sys.DisputeGameFactory()

	gameArgs := dgf.GameArgs(gameTypes.PermissionedGameType)
	require.NotEmpty(gameArgs, "game args is must be set for permissioned v2 dispute games")
	_, err := gameargs.Parse(gameArgs)
	require.NoError(err, "Permissioned game args invalid")

	gameArgs = dgf.GameArgs(gameTypes.CannonKonaGameType)
	require.NotEmpty(gameArgs, "game args is must be set for cannon-kona v2 dispute games")
	_, err = gameargs.Parse(gameArgs)
	require.NoError(err, "Permissionless game args invalid")

	dgf.VerifyGameImplPresent(gameTypes.PermissionedGameType)
	dgf.VerifyGameImplPresent(gameTypes.CannonKonaGameType)
}
