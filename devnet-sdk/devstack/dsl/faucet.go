package dsl

import "github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"

// Faucet wraps a stack.Faucet interface for DSL operations.
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
