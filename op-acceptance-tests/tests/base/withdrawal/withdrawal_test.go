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

	initialL1Balance, initialL2Balance := eth.OneThirdEther, eth.OneTenthEther

	// l1User and l2User share same private key
	l2User := sys.FunderL2.NewFundedEOA(initialL2Balance)
	l1User := l2User.AsEL(sys.L1EL)
	sys.FunderL1.Fund(l1User, initialL1Balance)
	depositAmount := eth.OneTenthEther
	withdrawalAmount := eth.OneTenthEther

	// The max amount of withdrawal is limited to the total amount of deposit
	// We trigger deposit first to fund the L1 ETHLockbox to satisfy the invariant
	deposit := bridge.Deposit(depositAmount, l1User)
	expectedL1UserBalance := initialL1Balance.Sub(depositAmount).Sub(deposit.GasCost())
	l1User.VerifyBalanceExact(expectedL1UserBalance)
	expectedL2UserBalance := initialL2Balance.Add(depositAmount)
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
