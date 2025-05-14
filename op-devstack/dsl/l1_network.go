package dsl

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// L1Network wraps a stack.L1Network interface for DSL operations
type L1Network struct {
	commonImpl
	inner stack.L1Network
}

// NewL1Network creates a new L1Network DSL wrapper
func NewL1Network(inner stack.L1Network) *L1Network {
	return &L1Network{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
	}
}

func (n *L1Network) String() string {
	return n.inner.ID().String()
}

func (n *L1Network) ChainID() eth.ChainID {
	return n.inner.ChainID()
}

// Escape returns the underlying stack.L1Network
func (n *L1Network) Escape() stack.L1Network {
	return n.inner
}

func (n *L1Network) WaitForBlock() eth.BlockRef {
	return NewL1ELNode(n.inner.L1ELNode(match.FirstL1EL)).WaitForBlock()
}
