package dsl

import (
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// L2Network wraps a stack.L2Network interface for DSL operations
type L2Network struct {
	commonImpl
	inner stack.L2Network
}

// NewL2Network creates a new L2Network DSL wrapper
func NewL2Network(inner stack.L2Network) *L2Network {
	return &L2Network{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
	}
}

func (n *L2Network) String() string {
	return n.inner.ID().String()
}

func (n *L2Network) ChainID() eth.ChainID {
	return n.inner.ChainID()
}

// Escape returns the underlying stack.L2Network
func (n *L2Network) Escape() stack.L2Network {
	return n.inner
}
