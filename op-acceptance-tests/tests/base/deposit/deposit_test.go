package deposit

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/contract"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	supervisorTypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestMain(m *testing.M) {
	presets.DoMain(m, presets.WithMinimal())
}

func TestL1ToL2Deposit(gt *testing.T) {
	// Create a test environment using op-devstack
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)

	// Wait for L1 node to be responsive
	sys.L1Network.WaitForOnline()

	// Fund Alice on L1
	fundingAmount := eth.ThreeHundredthsEther
	alice := sys.FunderL1.NewFundedEOA(fundingAmount)
	t.Log("Alice L1 address", alice.Address())
	initialBalance := alice.GetBalance()
	t.Log("Alice L1 balance", initialBalance)

	alicel2 := alice.AsEL(sys.L2EL)
	initialL2Balance := alicel2.GetBalance()
	t.Log("Alice L2 balance", initialL2Balance)

	// Get the optimism portal address
	rollupConfig := sys.L2Chain.Escape().RollupConfig()
	portalAddr := rollupConfig.DepositContractAddress

	depositAmount := eth.OneHundredthEther

	// Build the transaction
	portal := bindings.NewBindings[bindings.OptimismPortal2](bindings.WithClient(sys.L2EL.Escape().EthClient()), bindings.WithTo(portalAddr), bindings.WithTest(t))

	args := portal.DepositTransaction(alice.Address(), depositAmount, 300_000, false, []byte{})

	receipt := contract.Write(alice, args, txplan.WithValue(depositAmount))

	gasPrice := receipt.EffectiveGasPrice

	// Verify the deposit was successful
	gasCost := new(big.Int).Mul(new(big.Int).SetUint64(receipt.GasUsed), gasPrice)
	expectedFinalL1 := new(big.Int).Sub(initialBalance.ToBig(), depositAmount.ToBig())
	expectedFinalL1.Sub(expectedFinalL1, gasCost)
	t.Log("Alice L1 expected balance", eth.WeiBig(expectedFinalL1))
	t.Log("Alice L2 expected balance", initialL2Balance.Add(depositAmount))

	// Wait for the sequencer to process the deposit
	t.Require().Eventually(func() bool {
		head := sys.L2CL.HeadBlockRef(supervisorTypes.LocalUnsafe)
		return head.L1Origin.Number >= receipt.BlockNumber.Uint64()
	}, sys.L1EL.TransactionTimeout(), time.Second, "awaiting deposit to be processed by L2")

	alicel2.WaitForBalance(initialL2Balance.Add(depositAmount))

	alice.WaitForBalance(eth.WeiBig(expectedFinalL1))
}
