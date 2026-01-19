package cannon

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/gameargs"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestSmoke(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	require := t.Require()
	dgf := sys.DisputeGameFactory()

	gameArgs := dgf.GameArgs(gameTypes.PermissionedGameType)
	require.NotEmpty(gameArgs, "game args is must be set for permissioned v2 dispute games")
	_, err := gameargs.Parse(gameArgs)
	require.NoError(err, "Permissioned game args invalid")

	gameArgs = dgf.GameArgs(gameTypes.CannonGameType)
	require.NotEmpty(gameArgs, "game args is must be set for cannon v2 dispute games")
	_, err = gameargs.Parse(gameArgs)
	require.NoError(err, "Permissionless game args invalid")

	permissionedGame := dgf.GameImpl(gameTypes.PermissionedGameType)
	require.NotEmpty(permissionedGame.Address, "permissioned game impl must be set")
	cannonGame := dgf.GameImpl(gameTypes.CannonGameType)
	require.NotEmpty(cannonGame.Address, "cannon game impl must be set")
}
