package super_via_upgrade

import (
	"testing"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// TestSuperRootGamesInstalledViaOPCMUpgrade verifies that the single-chain
// interop preset installs super-root dispute games via OPCMv2.upgrade — the
// standard upgrade entrypoint — and ends up with SUPER_CANNON respected, the
// super game impls present, and no legacy PermissionedCannon game.
func TestSuperRootGamesInstalledViaOPCMUpgrade(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainInteropSupernodeProofs(t)

	sys.StandardBridge(sys.L2ChainA).VerifyRespectedGameType(gameTypes.SuperCannonGameType)
	sys.DisputeGameFactory().VerifyGameImplPresent(gameTypes.SuperCannonGameType)
	sys.DisputeGameFactory().VerifyGameImplPresent(gameTypes.SuperPermissionedGameType)
	sys.DisputeGameFactory().VerifyGameImplAbsent(gameTypes.PermissionedGameType)
}
