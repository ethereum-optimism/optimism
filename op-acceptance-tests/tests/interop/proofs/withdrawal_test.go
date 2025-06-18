package proofs

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"
)

func TestSuperRootWithdrawal(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t)
	sys.L1Network.WaitForOnline()

	initialL1Balance := eth.OneThirdEther
	initialL2Balance := eth.Ether(1)
	depositAmount := eth.OneTenthEther
	withdrawalAmount := eth.OneTenthEther

	l2User := sys.FunderA.NewFundedEOA(initialL2Balance)
	l1User := l2User.AsEL(sys.L1EL)
	sys.FunderL1.FundAtLeast(l1User, initialL1Balance)
	// Use a separate account to prove the withdrawal to make balance checks simpler
	l1Prover := sys.FunderL1.NewFundedEOA(eth.OneTenthEther)

	bridge := sys.StandardBridge(sys.L2ChainA)
	require.True(t, bridge.UsesSuperRoots(), "Expected interop system to be using super roots")

	deposit := bridge.Deposit(depositAmount, l1User)
	l1User.VerifyBalanceExact(initialL1Balance.Sub(depositAmount).Sub(deposit.GasCost()))
	l2User.VerifyBalanceExact(initialL2Balance.Add(depositAmount))

	withdrawal := bridge.InitiateWithdrawal(withdrawalAmount, l2User)
	withdrawal.Prove(l1Prover)
	l2User.VerifyBalanceExact(initialL2Balance.Add(depositAmount).Sub(withdrawalAmount).Sub(withdrawal.InitiateGasCost()))

	// Advance time until game is resolvable
	sys.AdvanceTime(bridge.GameResolutionDelay())
	withdrawal.WaitForDisputeGameResolved()

	// Advance time to when game finalization and proof finalization delay has expired
	sys.AdvanceTime(max(bridge.WithdrawalDelay(), bridge.DisputeGameFinalityDelay()))
	withdrawal.Finalize(l1Prover)

	l1User.VerifyBalanceExact(initialL1Balance.Sub(depositAmount).Sub(deposit.GasCost()).Add(withdrawalAmount))
}
