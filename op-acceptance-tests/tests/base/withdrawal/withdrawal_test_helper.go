package withdrawal

import (
	"testing"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	opforks "github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	ps "github.com/ethereum-optimism/optimism/op-proposer/proposer"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func withdrawalOpts(gameType gameTypes.GameType, extra ...presets.Option) []presets.Option {
	opts := []presets.Option{
		presets.WithTimeTravelEnabled(),
		presets.WithDeployerOptions(
			sysgo.WithFinalizationPeriodSeconds(1),
			// Satisfy OptimismPortal2 PROOF_MATURITY_DELAY_SECONDS check, avoid OptimismPortal_ProofNotOldEnough() revert
			sysgo.WithProofMaturityDelaySeconds(2),
			// Satisfy AnchorStateRegistry DISPUTE_GAME_FINALITY_DELAY_SECONDS check, avoid OptimismPortal_InvalidRootClaim() revert
			sysgo.WithDisputeGameFinalityDelaySeconds(2),
		),
		presets.WithGameTypeAdded(gameType),
		presets.WithRespectedGameTypeOverride(gameType),
		presets.WithProposerOption(func(_ sysgo.ComponentTarget, cfg *ps.CLIConfig) {
			cfg.DisputeGameType = uint32(gameType)
		}),
	}
	return append(opts, extra...)
}

func newSystem(t devtest.T, gameType gameTypes.GameType, extra ...presets.Option) *presets.Minimal {
	return presets.NewMinimal(t, withdrawalOpts(gameType, extra...)...)
}

func TestWithdrawal(gt *testing.T, gameType gameTypes.GameType, extra ...presets.Option) {
	t := devtest.ParallelT(gt)
	sys := newSystem(t, gameType, extra...)

	bridge := sys.StandardBridge()
	bridge.VerifyRespectedGameType(gameType)

	initialL1Balance := eth.OneThirdEther

	// l1User and l2User share same private key
	l1User := sys.FunderL1.NewFundedEOA(initialL1Balance)
	l2User := l1User.AsEL(sys.L2EL) // Only receives funds via the deposit
	depositAmount := eth.OneTenthEther
	withdrawalAmount := eth.OneHundredthEther

	// The max amount of withdrawal is limited to the total amount of deposit
	// We trigger deposit first to fund the L1 ETHLockbox to satisfy the invariant
	deposit := bridge.Deposit(depositAmount, l1User)
	expectedL1UserBalance := initialL1Balance.Sub(depositAmount).Sub(deposit.GasCost())
	l1User.VerifyBalanceExact(expectedL1UserBalance)
	expectedL2UserBalance := depositAmount
	l2User.VerifyBalanceExact(expectedL2UserBalance)

	withdrawal := bridge.InitiateWithdrawal(withdrawalAmount, l2User)
	expectedL2UserBalance = expectedL2UserBalance.Sub(withdrawalAmount).Sub(withdrawal.InitiateGasCost())
	l2User.VerifyBalanceExact(expectedL2UserBalance)

	withdrawal.Prove(l1User)
	expectedL1UserBalance = expectedL1UserBalance.Sub(withdrawal.ProveGasCost())
	l1User.VerifyBalanceExact(expectedL1UserBalance)

	// Advance time until game is resolvable
	sys.AdvanceTime(bridge.GameResolutionDelay())
	withdrawal.WaitForDisputeGameResolved()

	// Advance time to when game finalization and proof finalization delay has expired
	sys.AdvanceTime(max(bridge.WithdrawalDelay()-bridge.GameResolutionDelay(), bridge.DisputeGameFinalityDelay()))

	t.Logger().Info("Attempting to finalize", "proofMaturity", bridge.WithdrawalDelay(), "gameResolutionDelay", bridge.GameResolutionDelay(), "gameFinalityDelay", bridge.DisputeGameFinalityDelay())
	withdrawal.Finalize(l1User)
	expectedL1UserBalance = expectedL1UserBalance.Sub(withdrawal.FinalizeGasCost()).Add(withdrawalAmount)
	l1User.VerifyBalanceExact(expectedL1UserBalance)
}

// TestWithdrawalAfterUpgrade is like TestWithdrawal but waits for the given fork to activate
// before initiating the withdrawal, exercising the upgrade path rather than genesis activation.
func TestWithdrawalAfterUpgrade(gt *testing.T, gameType gameTypes.GameType, fork opforks.Name, extra ...presets.Option) {
	t := devtest.ParallelT(gt)
	sys := newSystem(t, gameType, extra...)

	sys.L2Chain.AwaitActivation(t, fork)

	bridge := sys.StandardBridge()
	bridge.VerifyRespectedGameType(gameType)

	initialL1Balance := eth.OneThirdEther

	l1User := sys.FunderL1.NewFundedEOA(initialL1Balance)
	l2User := l1User.AsEL(sys.L2EL)
	depositAmount := eth.OneTenthEther
	withdrawalAmount := eth.OneHundredthEther

	deposit := bridge.Deposit(depositAmount, l1User)
	expectedL1UserBalance := initialL1Balance.Sub(depositAmount).Sub(deposit.GasCost())
	l1User.VerifyBalanceExact(expectedL1UserBalance)
	expectedL2UserBalance := depositAmount
	l2User.VerifyBalanceExact(expectedL2UserBalance)

	withdrawal := bridge.InitiateWithdrawal(withdrawalAmount, l2User)
	expectedL2UserBalance = expectedL2UserBalance.Sub(withdrawalAmount).Sub(withdrawal.InitiateGasCost())
	l2User.VerifyBalanceExact(expectedL2UserBalance)

	withdrawal.Prove(l1User)
	expectedL1UserBalance = expectedL1UserBalance.Sub(withdrawal.ProveGasCost())
	l1User.VerifyBalanceExact(expectedL1UserBalance)

	sys.AdvanceTime(bridge.GameResolutionDelay())
	withdrawal.WaitForDisputeGameResolved()

	sys.AdvanceTime(max(bridge.WithdrawalDelay()-bridge.GameResolutionDelay(), bridge.DisputeGameFinalityDelay()))

	t.Logger().Info("Attempting to finalize", "proofMaturity", bridge.WithdrawalDelay(), "gameResolutionDelay", bridge.GameResolutionDelay(), "gameFinalityDelay", bridge.DisputeGameFinalityDelay())
	withdrawal.Finalize(l1User)
	expectedL1UserBalance = expectedL1UserBalance.Sub(withdrawal.FinalizeGasCost()).Add(withdrawalAmount)
	l1User.VerifyBalanceExact(expectedL1UserBalance)
}
