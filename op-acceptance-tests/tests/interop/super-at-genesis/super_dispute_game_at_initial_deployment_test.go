package super_at_genesis

import (
	"testing"
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestSuperPermissionedGameAtInitialDeploymentAdvancesAnchorStateRegistry(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainInteropSuperRootAtGenesis(t,
		presets.WithTimeTravelEnabled(),
		presets.WithDisputeGameFinalityDelaySeconds(2),
	)

	bridge := sys.StandardBridge(sys.L2ChainA)
	bridge.VerifyRespectedGameType(gameTypes.SuperPermissionedGameType)
	t.Require().Zero(bridge.GameResolutionDelay(), "SuperPermissioned games resolve during initialization")
	sys.DisputeGameFactory().VerifyGameImplPresent(gameTypes.SuperPermissionedGameType)

	game := sys.DisputeGameFactory().WaitForGame()
	t.Require().Equal(gameTypes.SuperPermissionedGameType, game.GameType())
	expectedRoot := game.RootClaimValue()
	expectedSequence := game.L2SequenceNumber()

	sys.AdvanceTime(3 * time.Second)
	sys.AnchorStateRegistry().WaitForAnchorRoot(expectedRoot, expectedSequence)
}
