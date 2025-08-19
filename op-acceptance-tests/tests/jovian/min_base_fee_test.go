package jovian

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestConfigurableMinBaseFee verifies configurable minimum base fee using devstack presets.
func TestConfigurableMinBaseFee(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	require := t.Require()

	err := dsl.RequiresL2Fork(t.Ctx(), sys, 0, rollup.Jovian)
	require.NoError(err, "Jovian fork must be active for this test")

	minBaseFee := dsl.NewMinBaseFee(t, sys.L2Chain, sys.L1EL, sys.L2EL)

	minBaseFee.CheckCompatibility()
	systemOwner := minBaseFee.GetSystemOwner()
	sys.FunderL1.FundAtLeast(systemOwner, eth.OneTenthEther)

	testCases := []struct {
		name        string
		significand uint8
		exponent    uint8
		shouldClamp bool
	}{
		{"MinBaseFeeOff", 0, 0, false},
		{"MinBaseFeeOn", 1, 9, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t devtest.T) {
			minBaseFee.SetMinBaseFeeFactors(tc.significand, tc.exponent)
			minBaseFee.WaitForL2Sync(tc.significand, tc.exponent)
			minBaseFee.VerifyL2Config(tc.significand, tc.exponent)

			if tc.shouldClamp {
				// Calculate minimum base fee: significand * 10^exponent
				minBase := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(tc.exponent)), nil)
				minBase.Mul(minBase, big.NewInt(int64(tc.significand)))
				minBaseFee.VerifyMinBaseFeeClamp(minBase)
			} else {
				minBaseFee.CheckBaseFeeCanDecrease()
			}

			t.Log("Test completed successfully:",
				"testCase", tc.name,
				"significand", tc.significand,
				"exponent", tc.exponent,
				"shouldClamp", tc.shouldClamp)
		})
	}

	minBaseFee.RestoreOriginalConfig()
}
