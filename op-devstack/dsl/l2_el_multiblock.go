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
	defer el.pinBranch(from, to)()
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

// VerifyNoSiblingBlocks checks that no two blocks in the inclusive range [from, to] share a
// timestamp: every block links to its predecessor by parent hash and sits exactly one block time
// later. That is the chain shape before the multi-blocks activation, and the shape an idle
// sequencer keeps producing after it.
func (el *L2ELNode) VerifyNoSiblingBlocks(from, to uint64, cfg *rollup.Config) {
	el.require.LessOrEqual(from, to, "block range must not be empty")
	el.require.NotZero(cfg.BlockTime, "block time must be set")

	el.log.Info("Verifying no sibling blocks", "name", el.inner.Name(), "chain", el.ChainID(), "from", from, "to", to)
	defer el.pinBranch(from, to)()
	prev := el.BlockRefByNumber(from)
	for num := from + 1; num <= to; num++ {
		block := el.BlockRefByNumber(num)
		el.require.Equalf(prev.Hash, block.ParentHash,
			"block %d must link to block %d by parent hash", num, num-1)
		el.require.Equalf(prev.Time+cfg.BlockTime, block.Time,
			"block %d must sit one block time after block %d at timestamp %d, but sits at %d",
			num, num-1, prev.Time, block.Time)
		prev = block
	}
}

// WaitForHeadPastTime waits until the head at the given label carries a timestamp strictly greater
// than the given one, and returns that head. Past the multi-blocks activation a timestamp no longer
// maps to a single block number, so tests reaching for "the chain got beyond this point in time"
// have to ask by timestamp rather than by block number.
func (el *L2ELNode) WaitForHeadPastTime(label eth.BlockLabel, timestamp uint64, timeout time.Duration) eth.L2BlockRef {
	ctx, cancel := context.WithTimeout(el.ctx, timeout)
	defer cancel()

	logger := el.log.With("name", el.inner.Name(), "chain", el.ChainID(), "label", label, "target_time", timestamp)
	logger.Info("Waiting for head past timestamp")

	var head eth.L2BlockRef
	err := wait.For(ctx, 250*time.Millisecond, func() (bool, error) {
		next, err := el.blockRefByLabel(label)
		if err != nil {
			logger.Warn("Head lookup failed; will retry", "err", err)
			return false, nil
		}
		head = next
		logger.Info("Head", "head", head, "time", head.Time)
		return head.Time > timestamp, nil
	})
	el.require.NoError(err,
		"expected the %s head on chain %s to pass timestamp %d within %s, last saw %s at timestamp %d",
		label, el.ChainID(), timestamp, timeout, head, head.Time)
	return head
}

// VerifyMatchesChain waits until this node's head at the given label covers block `to`, then checks
// that every block in the inclusive range [from, to] is hash-identical to the source node's.
// With eth.Unsafe it asserts a node followed the sequencer over P2P; with eth.Safe it asserts a
// node derived the same chain from what the batcher submitted to L1.
func (el *L2ELNode) VerifyMatchesChain(source *L2ELNode, label eth.BlockLabel, from, to uint64, timeout time.Duration) {
	el.require.LessOrEqual(from, to, "block range must not be empty")

	ctx, cancel := context.WithTimeout(el.ctx, timeout)
	defer cancel()

	logger := el.log.With("name", el.inner.Name(), "source", source.inner.Name(),
		"chain", el.ChainID(), "label", label, "target", to)
	logger.Info("Waiting for head to cover the source range", "from", from)

	var head eth.L2BlockRef
	err := wait.For(ctx, 250*time.Millisecond, func() (bool, error) {
		next, err := el.blockRefByLabel(label)
		if err != nil {
			logger.Warn("Head lookup failed; will retry", "err", err)
			return false, nil
		}
		head = next
		logger.Info("Head", "head", head)
		return head.Number >= to, nil
	})
	el.require.NoError(err,
		"expected the %s head on chain %s to reach block %d within %s, last saw %s",
		label, el.ChainID(), to, timeout, head)

	defer el.pinBranch(from, to)()
	defer source.pinBranch(from, to)()
	for num := from; num <= to; num++ {
		want := source.BlockRefByNumber(num)
		got := el.BlockRefByNumber(num)
		el.require.Equalf(want.Hash, got.Hash,
			"block %d must match %s: expected %s at timestamp %d, got %s at timestamp %d",
			num, source.inner.Name(), want.Hash, want.Time, got.Hash, got.Time)
	}
	logger.Info("Chain range matches source", "from", from, "to", to)
}

// pinBranch records the block at `to` and returns a check to run once the caller has finished
// walking [from, to]. A chain that moved under the walk makes every rule the caller checked
// meaningless, and the resulting failures read as consensus violations rather than as a reorg, so
// name the reorg instead.
func (el *L2ELNode) pinBranch(from, to uint64) func() {
	pinned := el.BlockRefByNumber(to)
	return func() {
		el.require.Equalf(pinned.Hash, el.BlockRefByNumber(to).Hash,
			"chain on %s reorged at block %d while verifying [%d, %d]; the checks above describe a chain that no longer exists",
			el.inner.Name(), to, from, to)
	}
}
