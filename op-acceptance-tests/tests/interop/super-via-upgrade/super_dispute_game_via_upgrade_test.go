package super_via_upgrade

import (
	"testing"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// TestSuperRootGamesInstalledViaOPCMUpgrade verifies that opcm.upgrade installs
// the permissionless super games and makes SUPER_CANNON_KONA the respected type.
func TestSuperRootGamesInstalledViaOPCMUpgrade(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainInteropSupernodeProofs(t)

	sys.StandardBridge(sys.L2ChainA).VerifyRespectedGameType(gameTypes.SuperCannonKonaGameType)
	sys.DisputeGameFactory().VerifyGameImplPresent(gameTypes.SuperCannonKonaGameType)
	sys.DisputeGameFactory().VerifyGameImplPresent(gameTypes.SuperPermissionedGameType)
	sys.DisputeGameFactory().VerifyGameImplAbsent(gameTypes.PermissionedGameType)
}
