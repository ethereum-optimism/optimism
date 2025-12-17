package withdrawal

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/custom_gas_token"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestWithdrawalRoot(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)

	// Skip this test if CGT is enabled
	custom_gas_token.SkipIfCGT(t, sys)

	require := sys.T.Require()

	require.True(sys.L2Chain.IsForkActive(forks.Isthmus), "Isthmus fork must be active for this test")

	secondCheck, err := dsl.CheckForChainFork(t.Ctx(), sys.L2Networks(), t.Logger())
	require.NoError(err, "error checking for chain fork")
	defer func() {
		require.NoError(secondCheck(false), "error checking for chain fork")
	}()

	bridge := sys.StandardBridge()
	initialL1Balance := eth.OneThirdEther
	initialL2Balance := eth.OneThirdEther

	// l1User and l2User share same private key
	l1User := sys.FunderL1.NewFundedEOA(initialL1Balance)
	l2User := l1User.AsEL(sys.L2EL) // Only receives funds via the deposit
	sys.FunderL2.FundAtLeast(l2User, initialL2Balance)
	withdrawalAmount := eth.OneHundredthEther

	withdrawal := bridge.InitiateWithdrawal(withdrawalAmount, l2User)
	expectedL2UserBalance := initialL2Balance.Sub(withdrawalAmount).Sub(withdrawal.InitiateGasCost())
	l2User.VerifyBalanceExact(expectedL2UserBalance)

	sys.L2EL.VerifyWithdrawalHashChangedIn(withdrawal.InitiateBlockHash())
}
