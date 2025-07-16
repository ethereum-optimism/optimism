package withdrawal

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"
)

func TestSuperRootWithdrawal(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t)
	sys.L1Network.WaitForOnline()

	initialL1Balance := eth.HalfEther
	initialL2Balance := eth.ZeroWei // L2 only gets funds from the deposit
	depositAmount := eth.OneThirdEther
	withdrawalAmount := eth.OneTenthEther

	l1User := sys.FunderL1.NewFundedEOA(initialL1Balance)
	l2User := l1User.AsEL(sys.L2ELA)

	bridge := sys.StandardBridge(sys.L2ChainA)
	require.True(t, bridge.UsesSuperRoots(), "Expected interop system to be using super roots")

	deposit := bridge.Deposit(depositAmount, l1User)
	l1User.VerifyBalanceExact(initialL1Balance.Sub(depositAmount).Sub(deposit.GasCost()))
	l2User.VerifyBalanceExact(initialL2Balance.Add(depositAmount))

	withdrawal := bridge.InitiateWithdrawal(withdrawalAmount, l2User)
	withdrawal.Prove(l1User)
	l2User.VerifyBalanceExact(initialL2Balance.Add(depositAmount).Sub(withdrawalAmount).Sub(withdrawal.InitiateGasCost()))

	// Advance time until game is resolvable
	sys.AdvanceTime(bridge.GameResolutionDelay())
	withdrawal.WaitForDisputeGameResolved()

	// Advance time to when game finalization and proof finalization delay has expired
	sys.AdvanceTime(max(bridge.WithdrawalDelay()-bridge.GameResolutionDelay(), bridge.DisputeGameFinalityDelay()))
	withdrawal.Finalize(l1User)

	l1User.VerifyBalanceExact(initialL1Balance.
		// Less cost of deposit
		Sub(depositAmount).
		Sub(deposit.GasCost()).
		// Less withdrawal L1 gas costs
		Sub(withdrawal.ProveGasCost()).
		Sub(withdrawal.FinalizeGasCost()).
		// Plus received withdrawal amount
		Add(withdrawalAmount))
}
