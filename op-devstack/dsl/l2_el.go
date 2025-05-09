package dsl

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"context"
)

// L2ELNode wraps a stack.L2ELNode interface for DSL operations
type L2ELNode struct {
	commonImpl
	elNode
	inner stack.L2ELNode
}

// NewL2ELNode creates a new L2ELNode DSL wrapper
func NewL2ELNode(inner stack.L2ELNode) *L2ELNode {
	return &L2ELNode{
		commonImpl: commonFromT(inner.T()),
		elNode:     elNode{inner: inner},
		inner:      inner,
	}
}

func (el *L2ELNode) String() string {
	return el.inner.ID().String()
}

// Escape returns the underlying stack.L2ELNode
func (el *L2ELNode) Escape() stack.L2ELNode {
	return el.inner
}

func (el *L2ELNode) BlockRefByLabel(label eth.BlockLabel) eth.BlockRef {
	ctx, cancel := context.WithTimeout(el.ctx, DefaultTimeout)
	defer cancel()
	block, err := el.inner.EthClient().BlockRefByLabel(ctx, label)
	el.require.NoError(err, "block not found using block label")
	return block
}
