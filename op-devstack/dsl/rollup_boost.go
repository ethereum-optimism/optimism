package dsl

import (
	opclient "github.com/ethereum-optimism/optimism/op-service/client"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
)

type RollupBoostNodesSet []*RollupBoostNode

func NewRollupBoostNodesSet(inner []stack.RollupBoostNode) RollupBoostNodesSet {
	rollupBoostNodes := make([]*RollupBoostNode, len(inner))
	for i, c := range inner {
		rollupBoostNodes[i] = NewRollupBoostNode(c)
	}
	return rollupBoostNodes
}

// RollupBoostNode wraps a stack.RollupBoostNode interface for DSL operations
type RollupBoostNode struct {
	inner stack.RollupBoostNode
}

func (r *RollupBoostNode) Escape() stack.RollupBoostNode {
	return r.inner
}

// NewRollupBoostNode creates a new RollupBoostNode DSL wrapper
func NewRollupBoostNode(inner stack.RollupBoostNode) *RollupBoostNode {
	return &RollupBoostNode{
		inner: inner,
	}
}

func (r *RollupBoostNode) FlashblocksClient() *opclient.WSClient {
	return r.inner.FlashblocksClient()
}
