package dsl

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
)

type FlashblocksBuilderSet []*FlashblocksBuilderNode

func NewFlashblocksBuilderSet(inner []stack.FlashblocksBuilderNode) FlashblocksBuilderSet {
	flashblocksBuilders := make([]*FlashblocksBuilderNode, len(inner))
	for i, c := range inner {
		flashblocksBuilders[i] = NewFlashblocksBuilderNode(c)
	}
	return flashblocksBuilders
}

type FlashblocksBuilderNode struct {
	commonImpl
	inner stack.FlashblocksBuilderNode
}

func NewFlashblocksBuilderNode(inner stack.FlashblocksBuilderNode) *FlashblocksBuilderNode {
	return &FlashblocksBuilderNode{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
	}
}

func (c *FlashblocksBuilderNode) String() string {
	return c.inner.ID().String()
}

func (c *FlashblocksBuilderNode) Escape() stack.FlashblocksBuilderNode {
	return c.inner
}
