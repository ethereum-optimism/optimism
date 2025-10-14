package disputegame_v2

import (
	"testing"

	challengerTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestSmoke(gt *testing.T) {
	gt.Skip("TODO(#17257): Re-enable once opcm.deploy supports v2 dispute games")
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	require := t.Require()
	dgf := sys.DisputeGameFactory()

	gameArgs := dgf.GameArgs(challengerTypes.PermissionedGameType)
	require.NotEmpty(gameArgs, "game args is must be set for permissioned v2 dispute games")
	gameArgs = dgf.GameArgs(challengerTypes.CannonGameType)
	require.NotEmpty(gameArgs, "game args is must be set for cannon v2 dispute games")

	permissionedGame := dgf.GameImpl(challengerTypes.PermissionedGameType)
	require.NotEmpty(permissionedGame.Address, "permissioned game impl must be set")
	cannonGame := dgf.GameImpl(challengerTypes.CannonGameType)
	require.NotEmpty(cannonGame.Address, "cannon game impl must be set")
}
