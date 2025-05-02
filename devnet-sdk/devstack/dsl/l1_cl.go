package dsl

import "github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"

// L1CLNode wraps a stack.L1CLNode interface for DSL operations
type L1CLNode struct {
	commonImpl
	inner stack.L1CLNode
}

// NewL1CLNode creates a new L1CLNode DSL wrapper
func NewL1CLNode(inner stack.L1CLNode) *L1CLNode {
	return &L1CLNode{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
	}
}

func (cl *L1CLNode) String() string {
	return cl.inner.ID().String()
}

// Escape returns the underlying stack.L1CLNode
func (cl *L1CLNode) Escape() stack.L1CLNode {
	return cl.inner
}
