package review_period

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-review-period/reviewperiod"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	ps "github.com/ethereum-optimism/optimism/op-proposer/proposer"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// resolvedWithdrawal is a proven withdrawal whose game resolved 13 hours ago: still inside the
// 3.5 day default finality window, but past the 12 hour minimum.
type resolvedWithdrawal struct {
	sys        *presets.Minimal
	delays     reviewperiod.ChainDelays
	withdrawal *dsl.Withdrawal
	user       *dsl.EOA
}

func newResolvedWithdrawal(t devtest.T) resolvedWithdrawal {
	gameType := gameTypes.PermissionedGameType
	sys := presets.NewMinimal(t,
		presets.WithTimeTravelEnabled(),
		presets.WithDeployerOptions(sysgo.WithFinalizationPeriodSeconds(1)),
		// A short proof maturity, so only the finality delay blocks finalization.
		presets.WithProofMaturityDelaySeconds(2),
		presets.WithRespectedGameTypeOverride(gameType),
		presets.WithProposerOption(func(_ sysgo.ComponentTarget, cfg *ps.CLIConfig) {
			cfg.DisputeGameType = uint32(gameType)
		}),
		presets.WithGameTypeAdded(gameType),
	)
	bridge := sys.StandardBridge()

	user := sys.FunderL1.NewFundedEOA(eth.OneThirdEther)
	bridge.Deposit(eth.OneTenthEther, user) // funds the ETHLockbox
	withdrawal := bridge.InitiateWithdrawal(eth.OneHundredthEther, user.AsEL(sys.L2EL))
	sys.DisputeGameFactory().WaitForGame()
	withdrawal.Prove(user)
	sys.AdvanceTime(bridge.GameResolutionDelay())
	withdrawal.WaitForDisputeGameResolved()
	sys.AdvanceTime(13 * time.Hour)

	systemConfig := sys.L2Chain.Escape().Deployment().SystemConfigProxyAddr()
	delays, err := reviewperiod.ReadChainDelays(t.Ctx(), sys.L1EL.EthClient(), systemConfig, rpc.LatestBlockNumber)
	t.Require().NoError(err, "failed to read the review period delays")
	return resolvedWithdrawal{sys: sys, delays: delays, withdrawal: withdrawal, user: user}
}

// TestLoweringFinalityDelayAppliesToResolvedGame checks that lowering the dispute game finality delay
// applies to a game that already resolved: a withdrawal blocked by the old delay finalizes at once.
func TestLoweringFinalityDelayAppliesToResolvedGame(gt *testing.T) {
	t := devtest.ParallelT(gt)
	r := newResolvedWithdrawal(t)
	t.Require().Error(r.withdrawal.CheckFinalizable(r.user), "withdrawal is finalizable before the finality delay passed")

	newDelayOwner(t, r.sys.L1EL, r.sys.L1Network.ChainID(), r.sys.L2Chain, r.sys.FunderL1).SetFinalityDelay(r.delays, 43200)

	t.Require().NoError(r.withdrawal.CheckFinalizable(r.user), "withdrawal is not finalizable after lowering the finality delay")
	r.withdrawal.Finalize(r.user)
}

// TestBlacklistedGameStaysUnfinalizableAfterLoweringFinalityDelay checks that a game the guardian
// blacklisted inside its finality window still can't finalize withdrawals once the delay is lowered.
func TestBlacklistedGameStaysUnfinalizableAfterLoweringFinalityDelay(gt *testing.T) {
	t := devtest.ParallelT(gt)
	r := newResolvedWithdrawal(t)
	game := r.sys.DisputeGameFactory().GameAtIndex(r.withdrawal.ProvenDisputeGameIndex().Int64())

	newGuardian(t, r.sys.L1EL, r.sys.L1Network.ChainID(), r.sys.L2Chain, r.sys.FunderL1).BlacklistGame(r.delays, game.Address)
	newDelayOwner(t, r.sys.L1EL, r.sys.L1Network.ChainID(), r.sys.L2Chain, r.sys.FunderL1).SetFinalityDelay(r.delays, 43200)

	t.Require().Error(r.withdrawal.CheckFinalizable(r.user), "withdrawal against a blacklisted game is finalizable")
}
