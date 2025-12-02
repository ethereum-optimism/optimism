package custom_gas_token

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestCGT_ValueTransferPaysGasInToken verifies that on CGT chains a simple L2
// value transfer charges gas in the native ERC-20, and balances reflect
// recipient +amount and sender > amount decrease (amount + gas).
func TestCGT_ValueTransferPaysGasInToken(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)

	ensureCGTOrSkip(t, sys)

	sender := sys.FunderL2.NewFundedEOA(eth.OneTenthEther)
	recipient := sys.Wallet.NewEOA(sys.L2EL)

	amount := eth.OneHundredthEther
	beforeS := sender.GetBalance()
	beforeR := recipient.GetBalance()

	// This sends L2 native (CGT) value.
	sender.Transfer(recipient.Address(), amount)

	// Wait until recipient reflects the transfer.
	// We don't wait on sender balance; it includes gas and is non-deterministic.
	recipient.WaitForBalance(beforeR.Add(amount))

	afterS := sender.GetBalance()
	afterR := recipient.GetBalance()

	// Recipient increased by amount
	wantR := beforeR.Add(amount)
	if afterR != wantR {
		t.Require().Fail("recipient balance mismatch: got %s, want %s", afterR, wantR)
	}

	// Sender decreased by at least amount (amount + gas). Strict inequality:
	if !(beforeS.Sub(afterS)).Gt(amount) {
		t.Require().Fail("sender delta must exceed transferred amount (gas must be paid): before=%s after=%s amount=%s",
			beforeS, afterS, amount)
	}
}
