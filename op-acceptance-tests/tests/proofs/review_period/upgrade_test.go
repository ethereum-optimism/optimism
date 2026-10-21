package review_period

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-review-period/reviewperiod"
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

	// The upgrade sets SuperPermissioned as the respected game type.
	sys.StandardBridge().VerifyRespectedGameType(gameTypes.SuperPermissionedGameType)

	pre, delays := readDelaysAcrossLastUpgrade(t, sys.L1EL, sys.L2Chain.Escape().Deployment().SystemConfigProxyAddr())
	for _, d := range reviewperiod.CompareDelays(pre, delays) {
		t.Require().False(d.Changed(), "%s changed across the OPCM upgrade: %d -> %d", d.Name, d.Before, d.After)
	}
	t.Require().Equal(uint64(proofMaturityDelaySeconds), delays.ProofMaturityDelaySeconds,
		"proof maturity delay changed across the OPCM upgrade")
	t.Require().Equal(uint64(disputeGameFinalityDelaySeconds), delays.DisputeGameFinalityDelaySeconds,
		"dispute game finality delay changed across the OPCM upgrade")
	t.Require().Equal(uint64(withdrawalDelaySeconds), delays.WithdrawalDelaySeconds,
		"DelayedWETH withdrawal delay changed across the OPCM upgrade")

	// The DelayedWETH read through the SystemConfig must be the one the games use.
	args, err := gameargs.Parse(sys.DisputeGameFactory().GameArgs(gameTypes.PermissionedGameType))
	t.Require().NoError(err, "failed to parse the permissioned game args")
	t.Require().Equal(args.Weth, delays.DelayedWETH, "SystemConfig points to a different DelayedWETH than the games")
}
