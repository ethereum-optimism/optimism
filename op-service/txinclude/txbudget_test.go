package txinclude

import (
	"math/big"
	"testing"

	opfees "github.com/ethereum-optimism/optimism/op-core/fees"
	"github.com/ethereum-optimism/optimism/op-service/accounting"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

type mockOPCostOracle struct {
	cost *big.Int
}

var _ OPCostOracle = mockOPCostOracle{}

func (m mockOPCostOracle) OPCost(*types.Transaction) *big.Int {
	return m.cost
}

func TestTxBudgetResubmitting(t *testing.T) {
	tx := types.NewTx(&types.BlobTx{
		Gas:        1,
		GasFeeCap:  uint256.NewInt(1),
		BlobFeeCap: uint256.NewInt(1),
		BlobHashes: []common.Hash{{}},
	})
	oracle := mockOPCostOracle{
		cost: big.NewInt(1),
	}
	// gasCost + opCost + 1 * params.BlobTxBlobGasPerBlob
	newCost := eth.WeiU64(1 + 1 + 1*params.BlobTxBlobGasPerBlob)

	t.Run("increased cost debits difference", func(t *testing.T) {
		startingBalance := eth.Ether(100)
		inner := accounting.NewBudget(startingBalance)
		tb := NewTxBudget(inner, WithOPCostOracle(oracle))
		oldCost := newCost.Sub(eth.OneWei)
		cost, err := tb.BeforeResubmit(oldCost, tx)
		require.NoError(t, err)
		require.Equal(t, newCost, cost)
		require.Equal(t, startingBalance.Sub(newCost.Sub(oldCost)), inner.Balance())
	})

	t.Run("decreased cost credits difference", func(t *testing.T) {
		startingBalance := eth.Ether(100)
		inner := accounting.NewBudget(startingBalance)
		tb := NewTxBudget(inner, WithOPCostOracle(oracle))
		oldCost := newCost.Add(eth.OneWei)
		cost, err := tb.BeforeResubmit(oldCost, tx)
		require.NoError(t, err)
		require.Equal(t, newCost, cost)
		require.Equal(t, startingBalance.Add(oldCost.Sub(newCost)), inner.Balance())
	})

	t.Run("same cost no change", func(t *testing.T) {
		startingBalance := eth.Ether(100)
		inner := accounting.NewBudget(startingBalance)
		tb := NewTxBudget(inner, WithOPCostOracle(oracle))
		cost, err := tb.BeforeResubmit(newCost, tx)
		require.NoError(t, err)
		require.Equal(t, newCost, cost)
		require.Equal(t, startingBalance, inner.Balance())
	})

	t.Run("insufficient budget for increase", func(t *testing.T) {
		tb := NewTxBudget(accounting.NewBudget(eth.OneWei), WithOPCostOracle(oracle))
		_, err := tb.BeforeResubmit(eth.OneWei, tx)
		var overdraftErr *accounting.OverdraftError
		require.ErrorAs(t, err, &overdraftErr)
	})
}

func TestTxBudgetCanceling(t *testing.T) {
	inner := accounting.NewBudget(eth.WeiU64(1000))
	tb := NewTxBudget(inner)
	tb.AfterCancel(eth.WeiU64(250), nil)
	require.Equal(t, eth.WeiU64(1250), inner.Balance())
}

func TestTxBudgetIncluded(t *testing.T) {
	const (
		gasLimit          = 30_000
		gasUsed           = 21_000 // <= gasLimit, as a real receipt reports
		effectiveGasPrice = 2
		operatorScalar    = 2
		operatorConstant  = 7
		overReserved      = 1_000 // wei the budget reserved beyond the actual cost
		startBalance      = 5_000 // balance left after the budgeted cost was debited
	)
	l1GasPrice := big.NewInt(1_000_000) // large enough that the L1 DA fee is non-zero
	l1BlobBaseFee := big.NewInt(1)

	tx := types.NewTx(&types.BlobTx{
		Gas:        gasLimit,
		GasFeeCap:  uint256.NewInt(effectiveGasPrice),
		BlobFeeCap: uint256.NewInt(1),
		BlobHashes: []common.Hash{{}},
	})
	receipt := &types.Receipt{
		EffectiveGasPrice:   eth.WeiU64(effectiveGasPrice).ToBig(),
		GasUsed:             gasUsed,
		Type:                types.DynamicFeeTxType,
		L1GasPrice:          l1GasPrice,
		L1BaseFeeScalar:     ptr(uint64(1)),
		L1BlobBaseFee:       l1BlobBaseFee,
		L1BlobBaseFeeScalar: ptr(uint64(1)),
		OperatorFeeScalar:   ptr(uint64(operatorScalar)),
		OperatorFeeConstant: ptr(uint64(operatorConstant)),
	}

	// The cost AfterIncluded must compute from the receipt: gas + L1 DA fee + Jovian operator
	// fee. (The L1 term is built from the same Fjord function because it depends on the tx's
	// FastLZ-compressed size, which isn't practical to hand-pin.)
	actualCost := new(big.Int).SetUint64(gasUsed * effectiveGasPrice)
	l1Cost, _ := types.NewL1CostFuncFjord(l1GasPrice, l1BlobBaseFee, big.NewInt(1), big.NewInt(1))(tx.RollupCostData())
	actualCost.Add(actualCost, l1Cost)
	actualCost.Add(actualCost, opfees.OperatorCostJovian(gasUsed, operatorScalar, operatorConstant))

	// The budget reserved overReserved wei beyond the actual cost; AfterIncluded must refund
	// exactly that much, so any error in how it sums gas/L1/operator shows up in the balance.
	budgetedCost := eth.WeiBig(new(big.Int).Add(actualCost, big.NewInt(overReserved)))
	inner := accounting.NewBudget(eth.WeiU64(startBalance))
	tb := NewTxBudget(inner)
	tb.AfterIncluded(budgetedCost, &IncludedTx{
		Transaction: tx,
		Receipt:     receipt,
	})
	require.Equal(t, eth.WeiU64(startBalance+overReserved), inner.Balance())
}
