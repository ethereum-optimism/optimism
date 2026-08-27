package dsl

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// WaitForSiblingBlocks waits until the chain contains a run of at least minGroupSize consecutive
// blocks that share one timestamp, and returns the first and last block of that run.
// Only blocks built after the call are considered, so a group found here was produced under the
// load the test just applied. A reorg discards the run in progress, since blocks from two branches
// are not a group.
func (el *L2ELNode) WaitForSiblingBlocks(minGroupSize int, timeout time.Duration) (first, last eth.L2BlockRef) {
	el.require.Greater(minGroupSize, 1, "a group of one is any ordinary block")

	ctx, cancel := context.WithTimeout(el.ctx, timeout)
	defer cancel()

	logger := el.log.With("name", el.inner.Name(), "chain", el.ChainID(), "min_group_size", minGroupSize)
	logger.Info("Waiting for sibling blocks")

	next := el.BlockRefByLabel(eth.Unsafe).Number + 1
	var run []eth.L2BlockRef
	longest := 0
	err := wait.For(ctx, 100*time.Millisecond, func() (bool, error) {
		head, err := el.blockRefByLabel(eth.Unsafe)
		if err != nil {
			logger.Warn("Head lookup failed; will retry", "err", err)
			return false, nil
		}
		if head.Number < next-1 {
			// the chain reorged below what we already scanned: rescan from the new head
			logger.Info("Head moved backwards, restarting scan", "head", head)
			next = head.Number + 1
			run = run[:0]
		}
		for ; next <= head.Number; next++ {
			block := el.BlockRefByNumber(next)
			if len(run) > 0 {
				prev := run[len(run)-1]
				if prev.Hash != block.ParentHash || prev.Time != block.Time {
					run = run[:0]
				}
			}
			run = append(run, block)
			longest = max(longest, len(run))
			if len(run) >= minGroupSize {
				first, last = run[0], run[len(run)-1]
				return true, nil
			}
		}
		return false, nil
	})
	el.require.NoError(err,
		"expected %d blocks sharing a timestamp on chain %s within %s, longest run seen was %d",
		minGroupSize, el.ChainID(), timeout, longest)
	logger.Info("Found sibling blocks", "first", first, "last", last, "timestamp", first.Time)
	return first, last
}

// VerifyTimestampGroups checks the multi-blocks chain rules over the inclusive block range
// [from, to]: every block links to its predecessor by parent hash, timestamps never decrease, they
// step by exactly the chain's block time when they do increase, blocks only share a timestamp
// where the config allows siblings, at most MaxMultiBlocks consecutive blocks share one, and all
// blocks of a group share one L1 origin.
// A group that starts before `from` is only counted from `from` on, so pick a range that starts at
// a group boundary to have the group limit fully checked.
func (el *L2ELNode) VerifyTimestampGroups(from, to uint64, cfg *rollup.Config) {
	el.require.LessOrEqual(from, to, "block range must not be empty")
	el.require.NotZero(cfg.BlockTime, "block time must be set")

	maxGroup := cfg.MaxMultiBlocksOrDefault()
	prev := el.BlockRefByNumber(from)
	groupLen := uint64(1)
	for num := from + 1; num <= to; num++ {
		block := el.BlockRefByNumber(num)
		el.require.Equal(prev.Hash, block.ParentHash,
			"block %d must link to block %d by parent hash", num, num-1)
		switch block.Time {
		case prev.Time:
			el.require.Truef(cfg.SiblingsAllowed(block.Time),
				"blocks %d and %d share timestamp %d, where siblings are not allowed", num-1, num, block.Time)
			groupLen++
			el.require.LessOrEqualf(groupLen, maxGroup,
				"blocks up to %d share timestamp %d, exceeding the group limit", num, block.Time)
			el.require.Equalf(prev.L1Origin, block.L1Origin,
				"blocks %d and %d share timestamp %d but not their L1 origin", num-1, num, block.Time)
		case prev.Time + cfg.BlockTime:
			groupLen = 1
		default:
			el.require.FailNow(fmt.Sprintf(
				"block %d has timestamp %d, which is neither its parent's %d nor one block time later",
				num, block.Time, prev.Time))
		}
		prev = block
	}
}
