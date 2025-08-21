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
		minBaseFee  uint64
		shouldClamp bool
	}{
		{"MinBaseFeeOff", 0, false},
		{"MinBaseFeeOn", 1_000_000_000, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t devtest.T) {
			minBaseFee.SetMinBaseFee(tc.minBaseFee)
			minBaseFee.WaitForL2Sync(tc.minBaseFee)
			minBaseFee.VerifyL2Config(tc.minBaseFee)

			if tc.shouldClamp {
				minBaseFee.VerifyMinBaseFeeClamp(big.NewInt(int64(tc.minBaseFee)))
			} else {
				minBaseFee.CheckBaseFeeCanDecrease()
			}

			t.Log("Test completed successfully:",
				"testCase", tc.name,
				"minBaseFee", tc.minBaseFee,
				"shouldClamp", tc.shouldClamp)
		})
	}

	minBaseFee.RestoreOriginalConfig()
}
