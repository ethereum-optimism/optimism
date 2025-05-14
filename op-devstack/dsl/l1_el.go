package dsl

import "github.com/ethereum-optimism/optimism/op-devstack/stack"

// L1ELNode wraps a stack.L1ELNode interface for DSL operations
type L1ELNode struct {
	*elNode
	inner stack.L1ELNode
}

// NewL1ELNode creates a new L1ELNode DSL wrapper
func NewL1ELNode(inner stack.L1ELNode) *L1ELNode {
	return &L1ELNode{
		elNode: newELNode(commonFromT(inner.T()), inner),
		inner:  inner,
	}
}

func (el *L1ELNode) String() string {
	return el.inner.ID().String()
}

// Escape returns the underlying stack.L1ELNode
func (el *L1ELNode) Escape() stack.L1ELNode {
	return el.inner
}
