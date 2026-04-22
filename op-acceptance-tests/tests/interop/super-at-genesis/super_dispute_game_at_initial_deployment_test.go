package super_at_genesis

import (
	"testing"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// TestSuperPermissionedDisputeGameInstalledAtInitialDeployment verifies that
// when a chain is deployed via op-deployer with the SuperRootGamesMigration
// dev feature flag set, SuperPermissionedDisputeGame is installed in the
// permissioned slot as part of the initial op-deployer apply - no post-deploy
// OPCMv2 migration is required.
//
// Tracks ethereum-optimism/optimism#18729.
func TestSuperPermissionedDisputeGameInstalledAtInitialDeployment(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainInteropSuperRootAtGenesis(t)

	sys.StandardBridge(sys.L2ChainA).VerifyRespectedGameType(gameTypes.SuperPermissionedGameType)
	sys.DisputeGameFactory().VerifyGameImplPresent(gameTypes.SuperPermissionedGameType)
	sys.DisputeGameFactory().VerifyGameImplAbsent(gameTypes.PermissionedGameType)
}
