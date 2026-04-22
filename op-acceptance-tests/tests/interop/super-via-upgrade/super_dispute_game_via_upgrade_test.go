package super_via_upgrade

import (
	"testing"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// TestSuperRootGamesInstalledViaOPCMUpgrade verifies that super-root dispute
// games can be installed on a vanilla-interop deployment by calling OPCMv2.upgrade
// — the standard upgrade entrypoint — instead of the one-off
// opcmMigrator.migrate path exercised by TestInteropSingleChainFaultProofs.
func TestSuperRootGamesInstalledViaOPCMUpgrade(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainInteropSupernodeProofsViaUpgrade(t)

	sys.StandardBridge(sys.L2ChainA).VerifyRespectedGameType(gameTypes.SuperCannonGameType)
	sys.DisputeGameFactory().VerifyGameImplPresent(gameTypes.SuperCannonGameType)
	sys.DisputeGameFactory().VerifyGameImplPresent(gameTypes.SuperPermissionedGameType)
	sys.DisputeGameFactory().VerifyGameImplAbsent(gameTypes.PermissionedGameType)
}
