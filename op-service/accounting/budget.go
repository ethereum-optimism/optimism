package accounting

import (
	"fmt"
	"sync"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type OverdraftError struct {
	Remaining eth.ETH
	Requested eth.ETH
}

var _ error = (*OverdraftError)(nil)

func (e *OverdraftError) Error() string {
	return fmt.Sprintf("budget overdraft: requested %s, remaining %s", e.Requested, e.Remaining)
}

type Budget struct {
	balanceMu sync.RWMutex
	balance   eth.ETH
}

func NewBudget(amount eth.ETH) *Budget {
	return &Budget{
		balance: amount,
	}
}

func (b *Budget) Balance() eth.ETH {
	b.balanceMu.RLock()
	defer b.balanceMu.RUnlock()
	return b.balance
}

func (b *Budget) Debit(amount eth.ETH) error {
	b.balanceMu.Lock()
	defer b.balanceMu.Unlock()
	result, underflow := b.balance.SubUnderflow(amount)
	if underflow {
		b.balance = eth.ZeroWei
		return &OverdraftError{
			Remaining: b.balance,
			Requested: amount,
		}
	}
	b.balance = result
	return nil
}

func (b *Budget) Credit(amount eth.ETH) {
	b.balanceMu.Lock()
	defer b.balanceMu.Unlock()
	var overflow bool
	b.balance, overflow = b.balance.AddOverflow(amount)
	if overflow {
		b.balance = eth.MaxU256Wei
	}
}
