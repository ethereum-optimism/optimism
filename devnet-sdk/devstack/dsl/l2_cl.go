package dsl

import (
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// L2CLNode wraps a stack.L2CLNode interface for DSL operations
type L2CLNode struct {
	commonImpl
	inner stack.L2CLNode
}

// NewL2CLNode creates a new L2CLNode DSL wrapper
func NewL2CLNode(inner stack.L2CLNode) *L2CLNode {
	return &L2CLNode{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
	}
}

func (cl *L2CLNode) String() string {
	return cl.inner.ID().String()
}

// Escape returns the underlying stack.L2CLNode
func (cl *L2CLNode) Escape() stack.L2CLNode {
	return cl.inner
}

func (cl *L2CLNode) SafeL2BlockRef() eth.L2BlockRef {
	syncStatus, err := cl.Escape().RollupAPI().SyncStatus(cl.ctx)
	cl.require.NoError(err, "Expected to get sync status")

	return syncStatus.SafeL2
}
