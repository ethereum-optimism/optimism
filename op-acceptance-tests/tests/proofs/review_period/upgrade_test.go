package review_period

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/gameargs"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// The 1DW minimum bounds, which differ from the standard defaults.
const (
	proofMaturityDelaySeconds       = 86400
	disputeGameFinalityDelaySeconds = 43200
	withdrawalDelaySeconds          = 43200
)

// TestOPCMUpgradePreservesReviewPeriodDelays checks that an OPCM upgrade, run during setup by
// WithGameTypeAdded, leaves the review period delays unchanged.
func TestOPCMUpgradePreservesReviewPeriodDelays(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimal(t,
		presets.WithProofMaturityDelaySeconds(proofMaturityDelaySeconds),
		presets.WithDisputeGameFinalityDelaySeconds(disputeGameFinalityDelaySeconds),
		presets.WithWithdrawalDelaySeconds(withdrawalDelaySeconds),
		presets.WithGameTypeAdded(gameTypes.PermissionedGameType),
	)
	bridge := sys.StandardBridge()
	factory := sys.DisputeGameFactory()

	// The upgrade sets SuperPermissioned as the respected game type.
	bridge.VerifyRespectedGameType(gameTypes.SuperPermissionedGameType)

	t.Require().Equal(proofMaturityDelaySeconds*time.Second, bridge.WithdrawalDelay(),
		"proof maturity delay changed across the OPCM upgrade")
	t.Require().Equal(disputeGameFinalityDelaySeconds*time.Second, bridge.DisputeGameFinalityDelay(),
		"dispute game finality delay changed across the OPCM upgrade")

	// Game implementations reach DelayedWETH through the factory's game args.
	args, err := gameargs.Parse(factory.GameArgs(gameTypes.PermissionedGameType))
	t.Require().NoError(err, "failed to parse the permissioned game args")
	t.Require().Equal(uint64(withdrawalDelaySeconds), factory.DelayedWETH(args.Weth).Delay().Uint64(),
		"DelayedWETH withdrawal delay changed across the OPCM upgrade")
}
