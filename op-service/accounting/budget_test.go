package accounting_test

import (
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/accounting"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"
)

func TestBudgetDebit(t *testing.T) {
	t.Run("successful debit reduces remaining balance", func(t *testing.T) {
		budget := accounting.NewBudget(eth.Ether(10))

		require.NoError(t, budget.Debit(eth.Ether(3)))
		require.Equal(t, eth.Ether(7), budget.Balance())

		require.NoError(t, budget.Debit(eth.Ether(2)))
		require.Equal(t, eth.Ether(5), budget.Balance())
	})

	t.Run("exact debit empties budget", func(t *testing.T) {
		budget := accounting.NewBudget(eth.Ether(5))

		err := budget.Debit(eth.Ether(5))
		require.NoError(t, err)
		require.Equal(t, eth.ZeroWei, budget.Balance())
	})

	t.Run("debit with insufficient funds returns error", func(t *testing.T) {
		budget := accounting.NewBudget(eth.Ether(3))

		err := budget.Debit(eth.Ether(5))
		require.Error(t, err)

		var insufficientErr *accounting.OverdraftError
		require.True(t, errors.As(err, &insufficientErr))
		require.Equal(t, &accounting.OverdraftError{
			Remaining: eth.ZeroWei,
			Requested: eth.Ether(5),
		}, insufficientErr)
		require.Equal(t, eth.ZeroWei, budget.Balance())
	})

	t.Run("debit from zero budget returns error", func(t *testing.T) {
		budget := accounting.NewBudget(eth.ZeroWei)

		err := budget.Debit(eth.OneWei)
		require.Error(t, err)

		var overdraftErr *accounting.OverdraftError
		require.True(t, errors.As(err, &overdraftErr))
		require.Equal(t, &accounting.OverdraftError{
			Remaining: eth.ZeroWei,
			Requested: eth.OneWei,
		}, overdraftErr)
		require.Equal(t, eth.ZeroWei, budget.Balance())
	})

	t.Run("multiple overdrafts maintain zero balance", func(t *testing.T) {
		budget := accounting.NewBudget(eth.Ether(1))

		require.Error(t, budget.Debit(eth.Ether(2)))
		require.Equal(t, eth.ZeroWei, budget.Balance())

		require.Error(t, budget.Debit(eth.OneWei))
		require.Equal(t, eth.ZeroWei, budget.Balance())
	})
}

func TestBudgetCredit(t *testing.T) {
	t.Run("credit increases remaining balance", func(t *testing.T) {
		budget := accounting.NewBudget(eth.Ether(5))

		budget.Credit(eth.Ether(3))
		require.Equal(t, eth.Ether(8), budget.Balance())

		budget.Credit(eth.Ether(2))
		require.Equal(t, eth.Ether(10), budget.Balance())
	})

	t.Run("credit to zero budget", func(t *testing.T) {
		budget := accounting.NewBudget(eth.ZeroWei)
		budget.Credit(eth.Ether(7))
		require.Equal(t, eth.Ether(7), budget.Balance())
	})

	t.Run("credit prevents overflow by setting to max", func(t *testing.T) {
		budget := accounting.NewBudget(eth.MaxU256Wei)
		budget.Credit(eth.OneWei)
		require.Equal(t, eth.MaxU256Wei, budget.Balance())
	})

	t.Run("credit near-max value causes overflow protection", func(t *testing.T) {
		// Start with a value close to max
		nearMax, _ := eth.MaxU256Wei.SubUnderflow(eth.Ether(1))
		budget := accounting.NewBudget(nearMax)

		// Credit more than the remaining space
		budget.Credit(eth.Ether(2))
		require.Equal(t, eth.MaxU256Wei, budget.Balance())
	})
}
