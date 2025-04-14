package dsl

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
)

// Faucet wraps a stack.Faucet interface for DSL operations.
// A Faucet is chain-specific.
// Note: Faucet wraps a stack component, to share faucet operations in kurtosis by hosting it as service,
// and prevent race-conditions with the account that sends out the faucet funds.
type Faucet struct {
	commonImpl
	inner stack.Faucet
}

// NewFaucet creates a new Faucet DSL wrapper
func NewFaucet(inner stack.Faucet) *Faucet {
	return &Faucet{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
	}
}

func (f *Faucet) String() string {
	return f.inner.ID().String()
}

// Escape returns the underlying stack.Faucet
func (f *Faucet) Escape() stack.Faucet {
	return f.inner
}

// Fund funds the given address with the given amount of ETH
func (f *Faucet) Fund(addr common.Address, amount eth.ETH) {
	if amount.IsZero() {
		return
	}
	err := retry.Do0(f.ctx, 3, retry.Exponential(), func() error {
		err := f.inner.API().RequestETH(f.ctx, addr, amount)
		if err != nil {
			f.log.Warn("Failed to fund address", "addr", addr, "amount", amount, "err", err)
		}
		return err
	})
	f.require.NoError(err, "must fund account %s with %s", addr, amount)
}
