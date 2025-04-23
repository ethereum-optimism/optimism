package dsl

import "github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"

// Superchain wraps a stack.Superchain interface for DSL operations
type Superchain struct {
	commonImpl
	inner stack.Superchain
}

// NewSuperchain creates a new Superchain DSL wrapper
func NewSuperchain(inner stack.Superchain) *Superchain {
	return &Superchain{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
	}
}

func (s *Superchain) String() string {
	return s.inner.ID().String()
}

// Escape returns the underlying stack.Superchain
func (s *Superchain) Escape() stack.Superchain {
	return s.inner
}
