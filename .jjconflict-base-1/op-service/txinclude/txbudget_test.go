package txinclude

import (
	"math/big"
	"testing"

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
	tx := types.NewTx(&types.BlobTx{
		Gas:        1,
		GasFeeCap:  uint256.NewInt(1),
		BlobFeeCap: uint256.NewInt(1),
		BlobHashes: []common.Hash{{}},
	})

	l1Cost, _ := types.NewL1CostFuncFjord(big.NewInt(1), big.NewInt(1), big.NewInt(1), big.NewInt(1))(tx.RollupCostData())
	l1Cost.Add(l1Cost, big.NewInt(1)) // operator fee
	oracle := mockOPCostOracle{
		cost: l1Cost,
	}
	// gasCost + opCost + 1 * params.BlobTxBlobGasPerBlob
	cost := big.NewInt(1) // gas cost
	cost.Add(cost, oracle.cost)
	cost.Add(cost, big.NewInt(params.BlobTxBlobGasPerBlob))
	budgetedCost := eth.WeiBig(cost)

	receipt := &types.Receipt{
		EffectiveGasPrice: eth.WeiU64(1).ToBig(),
		GasUsed:           budgetedCost.ToBig().Uint64(),
		Type:              types.DynamicFeeTxType,

		L1GasPrice:          big.NewInt(1),
		L1BaseFeeScalar:     ptr(uint64(1)),
		L1BlobBaseFee:       big.NewInt(1),
		L1BlobBaseFeeScalar: ptr(uint64(1)),
		OperatorFeeScalar:   ptr(uint64(1)),
		OperatorFeeConstant: ptr(uint64(0)),
	}

	startingBalance := eth.WeiU64(100)
	inner := accounting.NewBudget(startingBalance)
	tb := NewTxBudget(inner, WithOPCostOracle(oracle))
	tb.AfterIncluded(budgetedCost, &IncludedTx{
		Transaction: tx,
		Receipt:     receipt,
	})
	require.Equal(t, startingBalance, inner.Balance())
}
