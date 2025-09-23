package base

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestTransfer(gt *testing.T) {
	// Create a test environment using op-devstack
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)

	// Create two L2 wallets
	alice := sys.FunderL2.NewFundedEOA(eth.ThreeHundredthsEther)
	aliceBalance := alice.GetBalance()
	bob := sys.Wallet.NewEOA(sys.L2EL)
	bobBalance := bob.GetBalance()

	depositAmount := eth.OneHundredthEther
	bobAddr := bob.Address()
	receipt := alice.Transfer(bobAddr, depositAmount)
	bob.WaitForBalance(bobBalance.Add(depositAmount))

	gasCost := new(big.Int).Mul(new(big.Int).SetUint64(receipt.Included.Value().GasUsed), receipt.Included.Value().EffectiveGasPrice)
	expectedBalanceChange := new(big.Int).Add(gasCost, receipt.Included.Value().L1Fee)
	expectedFinalL2 := new(big.Int).Sub(aliceBalance.ToBig(), depositAmount.ToBig())
	expectedFinalL2.Sub(expectedFinalL2, expectedBalanceChange)

	alice.WaitForBalance(eth.WeiBig(expectedFinalL2))
}
