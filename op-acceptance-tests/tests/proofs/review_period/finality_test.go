package review_period

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-review-period/reviewperiod"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	ps "github.com/ethereum-optimism/optimism/op-proposer/proposer"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestLoweringFinalityDelayAppliesToResolvedGame checks that lowering the dispute game finality delay
// applies to a game that already resolved: a withdrawal blocked by the old delay finalizes at once.
func TestLoweringFinalityDelayAppliesToResolvedGame(gt *testing.T) {
	t := devtest.ParallelT(gt)
	gameType := gameTypes.PermissionedGameType
	sys := presets.NewMinimal(t,
		presets.WithTimeTravelEnabled(),
		presets.WithDeployerOptions(sysgo.WithFinalizationPeriodSeconds(1)),
		// A short proof maturity, so only the finality delay (the 3.5 day default) blocks finalization.
		presets.WithProofMaturityDelaySeconds(2),
		presets.WithRespectedGameTypeOverride(gameType),
		presets.WithProposerOption(func(_ sysgo.ComponentTarget, cfg *ps.CLIConfig) {
			cfg.DisputeGameType = uint32(gameType)
		}),
		presets.WithGameTypeAdded(gameType),
	)
	bridge := sys.StandardBridge()

	l1User := sys.FunderL1.NewFundedEOA(eth.OneThirdEther)
	bridge.Deposit(eth.OneTenthEther, l1User) // funds the ETHLockbox
	withdrawal := bridge.InitiateWithdrawal(eth.OneHundredthEther, l1User.AsEL(sys.L2EL))
	sys.DisputeGameFactory().WaitForGame()
	withdrawal.Prove(l1User)
	sys.AdvanceTime(bridge.GameResolutionDelay())
	withdrawal.WaitForDisputeGameResolved()

	// 13 hours after resolution the game is still inside the 3.5 day finality window.
	sys.AdvanceTime(13 * time.Hour)
	t.Require().Error(withdrawal.CheckFinalizable(l1User), "withdrawal is finalizable before the finality delay passed")

	// Lowering the delay to 12 hours puts the resolved game past it.
	systemConfig := sys.L2Chain.Escape().Deployment().SystemConfigProxyAddr()
	delays, err := reviewperiod.ReadChainDelays(t.Ctx(), sys.L1EL.EthClient(), systemConfig, rpc.LatestBlockNumber)
	t.Require().NoError(err, "failed to read the review period delays")
	newDelayOwner(t, sys.L1EL, sys.L1Network.ChainID(), sys.L2Chain, sys.FunderL1).SetFinalityDelay(delays, 43200)

	t.Require().NoError(withdrawal.CheckFinalizable(l1User), "withdrawal is not finalizable after lowering the finality delay")
	withdrawal.Finalize(l1User)
}
