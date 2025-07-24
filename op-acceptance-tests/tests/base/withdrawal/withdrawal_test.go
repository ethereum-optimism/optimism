package withdrawal

import (
	"testing"

	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestWithdrawal(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	require := sys.T.Require()

	bridge := sys.StandardBridge()
	require.EqualValues(faultTypes.FastGameType, bridge.RespectedGameType(), "Respected game type must be FastGame")

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

	t.Logger().Info("Attempting to finalize", "proofMaturity", bridge.WithdrawalDelay(), "gameResolutionDelay", bridge.GameResolutionDelay(), "gameFinalityDelay", bridge.DisputeGameFinalityDelay())
	withdrawal.Finalize(l1User)
	expectedL1UserBalance = expectedL1UserBalance.Sub(withdrawal.FinalizeGasCost()).Add(withdrawalAmount)
	l1User.VerifyBalanceExact(expectedL1UserBalance)
}
