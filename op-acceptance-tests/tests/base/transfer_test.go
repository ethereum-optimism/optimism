package base

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

func TestTransfer(gt *testing.T) {
	// Create a test environment using op-devstack
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)

	// Create two L2 wallets
	alice := sys.Funder.NewFundedEOA(eth.ThreeHundredthsEther)
	bob := sys.Wallet.NewEOA(sys.L2EL)
	bobBalance := bob.GetBalance()

	depositAmount := eth.OneHundredthEther
	bobAddr := bob.Address()
	alice.Transact(
		alice.Plan(),
		txplan.WithTo(&bobAddr),
		txplan.WithValue(depositAmount.ToBig()),
	)
	bob.VerifyBalanceExact(bobBalance.Add(depositAmount))
}
