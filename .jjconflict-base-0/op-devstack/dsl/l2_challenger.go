package dsl

import "github.com/ethereum-optimism/optimism/op-devstack/stack"

// L2Challenger wraps a stack.L2Challenger interface for DSL operations
type L2Challenger struct {
	commonImpl
	inner stack.L2Challenger
}

// NewL2Challenger creates a new L2Challenger DSL wrapper
func NewL2Challenger(inner stack.L2Challenger) *L2Challenger {
	return &L2Challenger{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
	}
}

func (c *L2Challenger) String() string {
	return c.inner.ID().String()
}

// Escape returns the underlying stack.L2Challenger
func (c *L2Challenger) Escape() stack.L2Challenger {
	return c.inner
}
