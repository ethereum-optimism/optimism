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
	return el.waitForNextBlock(1)
}

func (el *elNode) WaitForBlockNumber(targetBlock uint64) eth.BlockRef {
	var newRef eth.BlockRef

	err := wait.For(el.ctx, 500*time.Millisecond, func() (bool, error) {
		newBlock, err := el.inner.EthClient().InfoByLabel(el.ctx, eth.Unsafe)
		if err != nil {
			return false, err
		}

		newRef = eth.InfoToL1BlockRef(newBlock)
		if newBlock.NumberU64() >= targetBlock {
			el.log.Info("Target block reached", "chain", el.ChainID(), "block", newRef)
			return true, nil
		}
		return false, nil
	})
	el.require.NoError(err, "Expected to reach target block")
	return newRef
}

func (el *elNode) WaitForOnline() {
	el.require.Eventually(func() bool {
		el.log.Info("Waiting for online")
		_, err := el.inner.EthClient().InfoByLabel(el.ctx, eth.Unsafe)
		return err == nil
	}, 10*time.Second, 500*time.Millisecond, "Expected to be online")
}

func (el *elNode) IsCanonical(ref eth.BlockID) bool {
	blk, err := el.inner.EthClient().BlockRefByNumber(el.t.Ctx(), ref.Number)
	el.require.NoError(err)

	return blk.Hash == ref.Hash
}

// waitForNextBlockWithTimeout waits until the specified block number is present
func (el *elNode) waitForNextBlock(blocksFromNow uint64) eth.BlockRef {
	initial, err := el.inner.EthClient().InfoByLabel(el.ctx, eth.Unsafe)
	el.require.NoError(err, "Expected to get latest block from execution client")
	targetBlock := initial.NumberU64() + blocksFromNow
	initialRef := eth.InfoToL1BlockRef(initial)
	var newRef eth.BlockRef

	err = wait.For(el.ctx, 500*time.Millisecond, func() (bool, error) {
		newBlock, err := el.inner.EthClient().InfoByLabel(el.ctx, eth.Unsafe)
		if err != nil {
			return false, err
		}

		newRef = eth.InfoToL1BlockRef(newBlock)
		if newBlock.NumberU64() >= targetBlock {
			el.log.Info("Target block reached", "block", newRef)
			return true, nil
		}

		if initialRef == newRef {
			el.log.Info("Still same block detected as initial", "block", initialRef)
			return false, nil
		} else {
			el.log.Info("New block detected", "new_block", newRef, "prev_block", initialRef)
		}
		return false, nil
	})
	el.require.NoError(err, "Expected to reach target block")
	return newRef
}

func (el *elNode) stackEL() stack.ELNode {
	return el.inner
}

func (el *elNode) WaitForFinalization() eth.BlockRef {
	// Get current block and wait for it to be finalized
	currentBlock, err := el.inner.EthClient().InfoByLabel(el.ctx, eth.Finalized)
	el.require.NoError(err, "Expected to get current block from execution client")

	var finalizedBlock eth.BlockRef
	el.require.Eventually(func() bool {
		el.log.Info("Waiting for finalization")
		block, err := el.inner.EthClient().InfoByLabel(el.ctx, eth.Finalized)
		if err != nil {
			return false
		}
		if block.NumberU64() > currentBlock.NumberU64() {
			finalizedBlock = eth.InfoToL1BlockRef(block)
			return true
		}
		return false
	}, 5*time.Minute, 500*time.Millisecond, "Expected to be online")
	return finalizedBlock
}
