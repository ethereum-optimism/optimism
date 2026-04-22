package super_at_genesis

import (
	"testing"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// TestSuperPermissionedDisputeGameInstalledAtInitialDeployment verifies that
// a chain deployed with the SuperRootGamesMigration dev feature flag has
// SuperPermissionedDisputeGame in the permissioned slot at initial deploy,
// without requiring a post-deploy OPCMv2 migration.
func TestSuperPermissionedDisputeGameInstalledAtInitialDeployment(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainInteropSuperRootAtGenesis(t)

	sys.StandardBridge(sys.L2ChainA).VerifyRespectedGameType(gameTypes.SuperPermissionedGameType)
	sys.DisputeGameFactory().VerifyGameImplPresent(gameTypes.SuperPermissionedGameType)
	sys.DisputeGameFactory().VerifyGameImplAbsent(gameTypes.PermissionedGameType)
}
