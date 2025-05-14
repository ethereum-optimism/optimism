package dsl

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type ELNode interface {
	ChainID() eth.ChainID
	stackEL() stack.ELNode
}

// elNode implements DSL common between L1 and L2 EL nodes.
type elNode struct {
	commonImpl
	inner stack.ELNode
}

var _ ELNode = (*elNode)(nil)

func newELNode(common commonImpl, inner stack.ELNode) *elNode {
	return &elNode{
		commonImpl: common,
		inner:      inner,
	}
}

func (el *elNode) ChainID() eth.ChainID {
	return el.inner.ChainID()
}

func (el *elNode) WaitForBlock() eth.BlockRef {
	initial, err := el.inner.EthClient().InfoByLabel(el.ctx, eth.Unsafe)
	el.require.NoError(err, "Expected to get latest block from execution client")
	initialRef := eth.InfoToL1BlockRef(initial)
	var newRef eth.BlockRef
	err = wait.For(el.ctx, 500*time.Millisecond, func() (bool, error) {
		newBlock, err := el.inner.EthClient().InfoByLabel(el.ctx, eth.Unsafe)
		if err != nil {
			return false, err
		}

		newRef = eth.InfoToL1BlockRef(newBlock)
		if initialRef == newRef {
			el.log.Info("Still same block detected as initial", "block", initialRef)
			return false, nil
		}

		el.log.Info("New block detected", "new_block", newRef, "prev_block", initialRef)
		return true, nil
	})
	el.require.NoError(err, "Expected to get latest block from execution client for comparison")
	return newRef
}

func (el *elNode) stackEL() stack.ELNode {
	return el.inner
}
