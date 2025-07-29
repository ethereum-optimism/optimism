package operatorfee

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestOperatorFeeDevstack(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	require := t.Require()

	err := dsl.RequiresL2Fork(t.Ctx(), sys, 0, rollup.Isthmus)
	require.NoError(err, "Isthmus fork must be active for this test")

	alice := sys.FunderL2.NewFundedEOA(eth.OneTenthEther)
	bob := sys.Wallet.NewEOA(sys.L2EL)

	operatorFee := dsl.NewOperatorFee(t, sys.L2Chain, sys.L1EL)

	operatorFee.CheckCompatibility()
	systemOwner := operatorFee.GetSystemOwner()
	sys.FunderL1.FundAtLeast(systemOwner, eth.OneTenthEther)

	// First, ensure L2 is synced with current L1 state before starting tests
	t.Log("Ensuring L2 is synced with current L1 state...")
	operatorFee.WaitForL2SyncWithCurrentL1State()

	testCases := []struct {
		name     string
		scalar   uint32
		constant uint64
	}{
		{"ZeroFees", 0, 0},
		{"NonZeroFees", 300, 400},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t devtest.T) {
			operatorFee.SetOperatorFee(tc.scalar, tc.constant)
			operatorFee.WaitForL2Sync(tc.scalar, tc.constant)
			operatorFee.VerifyL2Config(tc.scalar, tc.constant)

			result := operatorFee.ValidateTransactionFees(alice, bob, big.NewInt(1000), tc.scalar, tc.constant)

			t.Log("Test completed successfully:",
				"testCase", tc.name,
				"gasUsed", result.TransactionReceipt.GasUsed,
				"actualTotalFee", result.ActualTotalFee.String(),
				"expectedOperatorFee", result.ExpectedOperatorFee.String(),
				"vaultBalanceIncrease", result.VaultBalanceIncrease.String())
		})
	}

	operatorFee.RestoreOriginalConfig()
}
