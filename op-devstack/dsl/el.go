package dsl

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type ELNode interface {
	ChainID() eth.ChainID
	WaitForTime(timestamp uint64) eth.BlockRef
	AwaitTime(ctx context.Context, timestamp uint64) error
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

func (el *elNode) WaitForLabel(label eth.BlockLabel, predicate func(eth.BlockInfo) (bool, error)) eth.BlockInfo {
	block, err := el.awaitLabel(el.ctx, label, predicate)
	el.require.NoError(err, "Failed to find block")
	return block
}

// awaitLabel is WaitForLabel for callers that must handle the failure themselves rather than fail
// the test.
func (el *elNode) awaitLabel(ctx context.Context, label eth.BlockLabel, predicate func(eth.BlockInfo) (bool, error)) (eth.BlockInfo, error) {
	var block eth.BlockInfo
	var lastLookupErr error
	err := wait.For(ctx, 200*time.Millisecond, func() (bool, error) {
		info, err := el.inner.EthClient().InfoByLabel(ctx, label)
		if err != nil {
			// Transient against a live chain, so keep polling; only report it if the target is
			// never reached.
			lastLookupErr = err
			el.log.Debug("Block lookup failed", "chain", el.ChainID(), "err", err)
			return false, nil
		}
		lastLookupErr = nil
		block = info
		ok, err := predicate(block)
		if ok {
			el.log.Info("Target block reached", "chain", el.ChainID(), "block", eth.ToBlockID(block))
		} else if err == nil {
			el.log.Debug("Target block not reached yet", "chain", el.ChainID(), "block", eth.ToBlockID(block))
		}
		return ok, err
	})
	if err != nil && lastLookupErr != nil {
		return block, fmt.Errorf("%w (last block lookup failed: %w)", err, lastLookupErr)
	}
	return block, err
}

func (el *elNode) WaitForLabelRef(label eth.BlockLabel, predicate func(eth.BlockInfo) (bool, error)) eth.BlockRef {
	return eth.InfoToL1BlockRef(el.WaitForLabel(label, predicate))
}

func (el *elNode) WaitForUnsafe(predicate func(eth.BlockInfo) (bool, error)) eth.BlockInfo {
	return el.WaitForLabel(eth.Unsafe, predicate)
}

func (el *elNode) WaitForUnsafeRef(predicate func(eth.BlockInfo) (bool, error)) eth.BlockRef {
	return eth.InfoToL1BlockRef(el.WaitForUnsafe(predicate))
}

func (el *elNode) WaitForBlockNumber(targetBlock uint64) eth.BlockInfo {
	info := el.WaitForUnsafe(func(info eth.BlockInfo) (bool, error) {
		return info.NumberU64() >= targetBlock, nil
	})
	if info.NumberU64() == targetBlock {
		return info
	}
	// we've gone too far
	info, err := el.inner.EthClient().InfoByNumber(el.ctx, targetBlock)
	el.require.NoError(err, "Expected to get info by number after waiting")
	return info
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

	return el.WaitForUnsafeRef(func(info eth.BlockInfo) (bool, error) {
		return info.NumberU64() >= targetBlock, nil
	})
}

// WaitForTime waits until the chain has reached or surpassed the given timestamp.
func (el *elNode) WaitForTime(timestamp uint64) eth.BlockRef {
	return el.WaitForUnsafeRef(reachedTime(timestamp))
}

// AwaitTime is WaitForTime for callers that must handle the failure themselves. See awaitLabel.
func (el *elNode) AwaitTime(ctx context.Context, timestamp uint64) error {
	_, err := el.awaitLabel(ctx, eth.Unsafe, reachedTime(timestamp))
	return err
}

func reachedTime(timestamp uint64) func(eth.BlockInfo) (bool, error) {
	return func(info eth.BlockInfo) (bool, error) {
		return info.Time() >= timestamp, nil
	}
}

func (el *elNode) stackEL() stack.ELNode {
	return el.inner
}

// WaitForFinalization waits for the current block height to be finalized. Note that it does not
// ensure that the finalized block is the same as the current unsafe block (i.e., it is not
// reorg-aware).
func (el *elNode) WaitForFinalization() eth.BlockRef {
	// Get current block and wait for it to be finalized
	currentBlock, err := el.inner.EthClient().InfoByLabel(el.ctx, eth.Unsafe)
	el.require.NoError(err, "Expected to get current block from execution client")

	return el.WaitForLabelRef(eth.Finalized, func(info eth.BlockInfo) (bool, error) {
		return info.NumberU64() >= currentBlock.NumberU64(), nil
	})
}
