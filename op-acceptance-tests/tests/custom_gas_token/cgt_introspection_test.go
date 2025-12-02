package custom_gas_token

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// TestCGT_IntrospectionViaL1Block verifies that the L2 L1Block predeploy reports
// that CGT mode is enabled and exposes non-empty token metadata (name, symbol).
func TestCGT_IntrospectionViaL1Block(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)

	name, symbol := ensureCGTOrSkip(t, sys)

	// Metadata should be non-empty.
	if name == "" {
		t.Require().Fail("gasPayingTokenName() returned empty string")
	}
	if symbol == "" {
		t.Require().Fail("gasPayingTokenSymbol() returned empty string")
	}
}
