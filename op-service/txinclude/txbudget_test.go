package txinclude

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/accounting"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestTxBudgetResubmitting(t *testing.T) {
	t.Run("increased cost debits difference", func(t *testing.T) {
		inner := accounting.NewBudget(eth.WeiU64(1000))
		tb := newTxBudget(inner)
		require.NoError(t, tb.resubmitting(eth.WeiU64(100), eth.WeiU64(150)))
		require.Equal(t, eth.WeiU64(950), inner.Balance())
	})

	t.Run("decreased cost credits difference", func(t *testing.T) {
		inner := accounting.NewBudget(eth.WeiU64(1000))
		tb := newTxBudget(inner)
		require.NoError(t, tb.resubmitting(eth.WeiU64(200), eth.WeiU64(150)))
		require.Equal(t, eth.WeiU64(1050), inner.Balance())
	})

	t.Run("same cost no change", func(t *testing.T) {
		inner := accounting.NewBudget(eth.WeiU64(1000))
		tb := newTxBudget(inner)
		cost := eth.WeiU64(100)
		require.NoError(t, tb.resubmitting(cost, cost))
		require.Equal(t, eth.WeiU64(1000), inner.Balance())
	})

	t.Run("insufficient budget for increase", func(t *testing.T) {
		inner := accounting.NewBudget(eth.WeiU64(30))
		tb := newTxBudget(inner)
		err := tb.resubmitting(eth.WeiU64(100), eth.WeiU64(150))
		require.Error(t, err)
		var overdraftErr *accounting.OverdraftError
		require.ErrorAs(t, err, &overdraftErr)
	})
}

func TestTxBudgetCanceling(t *testing.T) {
	inner := accounting.NewBudget(eth.WeiU64(1000))
	tb := newTxBudget(inner)
	tb.canceling(eth.WeiU64(250))
	require.Equal(t, eth.WeiU64(1250), inner.Balance())
}

func TestTxBudgetIncluded(t *testing.T) {
	t.Run("dynamic fee tx", func(t *testing.T) {
		startingBalance := eth.ZeroWei
		inner := accounting.NewBudget(startingBalance)
		tb := newTxBudget(inner)
		tb.included(&IncludedTx{
			Transaction: types.NewTx(&types.DynamicFeeTx{
				GasFeeCap: eth.GWei(1).ToBig(),
				Gas:       21000,
			}),
			Receipt: &types.Receipt{
				EffectiveGasPrice: eth.GWei(1).ToBig(),
				GasUsed:           21000,
				Type:              types.DynamicFeeTxType,
			},
		})
		require.Equal(t, startingBalance, inner.Balance())
	})

	t.Run("dynamic fee tx less gas", func(t *testing.T) {
		startingBalance := eth.ZeroWei
		inner := accounting.NewBudget(startingBalance)
		tb := newTxBudget(inner)
		tb.included(&IncludedTx{
			Transaction: types.NewTx(&types.DynamicFeeTx{
				GasFeeCap: eth.GWei(1).ToBig(),
				Gas:       100_000,
			}),
			Receipt: &types.Receipt{
				EffectiveGasPrice: eth.GWei(1).ToBig(),
				GasUsed:           75_000,
				Type:              types.DynamicFeeTxType,
			},
		})
		require.Equal(t, startingBalance.Add(eth.GWei(25_000)), inner.Balance())
	})

	t.Run("dynamic fee tx cheaper gas", func(t *testing.T) {
		startingBalance := eth.ZeroWei
		inner := accounting.NewBudget(startingBalance)
		tb := newTxBudget(inner)
		tb.included(&IncludedTx{
			Transaction: types.NewTx(&types.DynamicFeeTx{
				GasFeeCap: eth.GWei(2).ToBig(),
				Gas:       100_000,
			}),
			Receipt: &types.Receipt{
				EffectiveGasPrice: eth.GWei(1).ToBig(),
				GasUsed:           100_000,
				Type:              types.DynamicFeeTxType,
			},
		})
		require.Equal(t, startingBalance.Add(eth.GWei(100_000)), inner.Balance())
	})

	t.Run("blob tx", func(t *testing.T) {
		startingBalance := eth.ZeroWei
		inner := accounting.NewBudget(startingBalance)
		tb := newTxBudget(inner)
		tb.included(&IncludedTx{
			Transaction: types.NewTx(&types.BlobTx{
				GasFeeCap: eth.GWei(1).ToU256(),
				Gas:       21_000,

				BlobFeeCap: eth.GWei(1).ToU256(),
				BlobHashes: []common.Hash{{}},
			}),
			Receipt: &types.Receipt{
				EffectiveGasPrice: eth.GWei(1).ToBig(),
				GasUsed:           21_000,
				Type:              types.BlobTxType,

				BlobGasPrice: eth.GWei(1).ToBig(),
				BlobGasUsed:  params.BlobTxBlobGasPerBlob,
			},
		})
		require.Equal(t, startingBalance, inner.Balance())
	})

	t.Run("blob transaction smaller fee", func(t *testing.T) {
		startingBalance := eth.ZeroWei
		inner := accounting.NewBudget(startingBalance)
		tb := newTxBudget(inner)
		tb.included(&IncludedTx{
			Transaction: types.NewTx(&types.BlobTx{
				GasFeeCap: eth.GWei(30).ToU256(),
				Gas:       22_000,

				BlobFeeCap: eth.GWei(2).ToU256(),
				BlobHashes: []common.Hash{{}},
			}),
			Receipt: &types.Receipt{
				EffectiveGasPrice: eth.GWei(30).ToBig(),
				GasUsed:           22_000,
				Type:              types.BlobTxType,

				BlobGasPrice: eth.GWei(1).ToBig(),
				BlobGasUsed:  params.BlobTxBlobGasPerBlob,
			},
		})

		require.Equal(t, startingBalance.Add(eth.GWei(params.BlobTxBlobGasPerBlob)), inner.Balance())
	})
}
