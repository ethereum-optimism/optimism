package withdrawal

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestWithdrawalRoot(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	require := sys.T.Require()

	err := dsl.RequiresL2Fork(t.Ctx(), sys, 0, rollup.Isthmus)
	require.NoError(err, "Isthmus fork must be active for this test")

	secondCheck, err := dsl.CheckForChainFork(t.Ctx(), sys.L2Networks(), t.Logger())
	require.NoError(err, "error checking for chain fork")
	defer func() {
		require.NoError(secondCheck(false), "error checking for chain fork")
	}()

	bridge := sys.StandardBridge()
	// 333_333_333 gwei (1 billion wei)
	initialL1Balance := eth.OneThirdEther
	initialL2Balance := eth.OneThirdEther

	// l1User and l2User share same private key
	l1User := sys.FunderL1.NewFundedEOA(initialL1Balance)
	l2User := l1User.AsEL(sys.L2EL) // Only receives funds via the deposit
	sys.FunderL2.FundAtLeast(l2User, initialL2Balance)
	withdrawalAmount := eth.OneHundredthEther

	withdrawal := bridge.InitiateWithdrawal(withdrawalAmount, l2User)
	expectedL2UserBalance := initialL2Balance.Sub(withdrawalAmount).Sub(withdrawal.InitiateGasCost())
	// divergence here
	// actual_wrong = initialL2Balance - eth.OneHundredthEther - withdrawal.InitiateGasCost()
	// expected 323203245609236609 = 333333333 * 1000000000 - 10000000 * 1000000000 - gasCost
	// gasCost = 130087390763391
	// expected - actual = 422
	// actual = 323203245609236187 = (delta + 233408941113112216) - 10000000 * 1000000000 - 130087390763391
	// delta = 99924391886887362
	// 100000000 * 1000000000 - 99924391886887362 = 75608113112638

	l2User.VerifyBalanceExact(expectedL2UserBalance)

	sys.L2EL.VerifyWithdrawalHashChangedIn(withdrawal.InitiateBlockHash())
}
