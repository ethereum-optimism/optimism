package dsl

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type ELNode interface {
	ChainID() eth.ChainID
	stackEL() stack.ELNode
}

// elNode implements DSL common between L1 and L2 EL nodes.
type elNode struct {
	inner stack.ELNode
}

var _ ELNode = (*elNode)(nil)

func (el *elNode) ChainID() eth.ChainID {
	return el.inner.ChainID()
}

func (el *elNode) stackEL() stack.ELNode {
	return el.inner
}
