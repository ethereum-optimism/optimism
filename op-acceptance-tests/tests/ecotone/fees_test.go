package ecotone

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestFees(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimal(t, presets.WithDeployerOptions(sysgo.WithEcotoneAtGenesis))

	alice := sys.FunderL2.NewFundedEOA(eth.OneTenthEther)
	bob := sys.Wallet.NewEOA(sys.L2EL)

	ecotoneFees := dsl.NewEcotoneFees(t, sys.L2Chain)

	result := ecotoneFees.ValidateTransaction(alice, bob, big.NewInt(42000000000))

	ecotoneFees.LogResults(result)

	t.Log("Ecotone fees test completed successfully",
		"gasUsed", result.TransactionReceipt.GasUsed,
		"l1Fee", result.L1Fee.String(),
		"l2Fee", result.L2Fee.String(),
		"baseFee", result.BaseFee.String(),
		"priorityFee", result.PriorityFee.String(),
		"totalFee", result.TotalFee.String(),
		"walletBalanceDiff", result.WalletBalanceDiff.String())
}
