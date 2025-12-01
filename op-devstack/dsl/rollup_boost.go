package dsl

import "github.com/ethereum-optimism/optimism/op-devstack/stack"

type RollupBoostNodesSet []*RollupBoostNode

func NewRollupBoostNodesSet(inner []stack.RollupBoostNode, control stack.ControlPlane) RollupBoostNodesSet {
	rollupBoostNodes := make([]*RollupBoostNode, len(inner))
	for i, c := range inner {
		rollupBoostNodes[i] = NewRollupBoostNode(c, control)
	}
	return rollupBoostNodes
}

// RollupBoostNode wraps a stack.RollupBoostNode interface for DSL operations
type RollupBoostNode struct {
	inner   stack.RollupBoostNode
	control stack.ControlPlane
}

func (r *RollupBoostNode) Escape() stack.RollupBoostNode {
	return r.inner
}

// NewRollupBoostNode creates a new RollupBoostNode DSL wrapper
func NewRollupBoostNode(inner stack.RollupBoostNode, control stack.ControlPlane) *RollupBoostNode {
	return &RollupBoostNode{
		inner,
		control,
	}
}

func (r *RollupBoostNode) FlashblocksClient() *FlashblocksWSClient {
	return NewFlashblocksWSClient(r.inner.FlashblocksClient())
}
