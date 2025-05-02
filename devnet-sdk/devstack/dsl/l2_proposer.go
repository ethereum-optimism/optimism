package dsl

import "github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"

// L2Proposer wraps a stack.L2Proposer interface for DSL operations
type L2Proposer struct {
	commonImpl
	inner stack.L2Proposer
}

// NewL2Proposer creates a new L2Proposer DSL wrapper
func NewL2Proposer(inner stack.L2Proposer) *L2Proposer {
	return &L2Proposer{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
	}
}

func (p *L2Proposer) String() string {
	return p.inner.ID().String()
}

// Escape returns the underlying stack.L2Proposer
func (p *L2Proposer) Escape() stack.L2Proposer {
	return p.inner
}
