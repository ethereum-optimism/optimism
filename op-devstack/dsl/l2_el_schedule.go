package dsl

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
)

const (
	// scheduleCatchUpBlockTimes bounds, in block times, how long a chain may take to be on
	// schedule before it is held to it. A sequencer that starts behind its genesis time builds
	// back-to-back blocks until it has caught up, and every one of those is late by construction.
	scheduleCatchUpBlockTimes = 60
	// scheduleSamplesPerBlock is how many times per block time the head is sampled while the
	// chain is held to its schedule; it bounds how long after a block became canonical it is
	// first seen.
	scheduleSamplesPerBlock = 8
)

// AdvancedOnScheduleFn returns a CheckFunc that waits for the unsafe head to be on schedule, then
// holds the chain to its schedule for the next blocks blocks and fails as soon as a slot is
// missed. A block is on schedule when it extends the previous head, carries the previous head's
// timestamp plus blockTime, and is canonical before that timestamp plus blockTime has passed on
// the wall clock, which is when the next block is due. A block not seen within a further block
// time of being due has missed its slot even before it arrives.
func (el *L2ELNode) AdvancedOnScheduleFn(blocks uint64, blockTime time.Duration) CheckFunc {
	return func() error {
		if blockTime%time.Second != 0 {
			return fmt.Errorf("block time %s is not a whole number of seconds, which no L2 block timestamp can follow", blockTime)
		}
		logger := el.log.With("name", el.inner.Name(), "chain", el.ChainID(), "label", eth.Unsafe, "block_time", blockTime)
		sampleInterval := blockTime / scheduleSamplesPerBlock
		previous, err := el.caughtUpToSchedule(logger, blockTime, sampleInterval)
		if err != nil {
			return err
		}
		target := previous.Number + blocks
		logger = logger.With("from", previous.Number, "target", target)
		logger.Info("Expecting L2EL to advance on schedule")
		var maxLag time.Duration
		for previous.Number < target {
			if err := clock.SystemClock.SleepCtx(el.ctx, sampleInterval); err != nil { // nosemgrep: flake-sleep-in-test -- fixed-cadence sampling of the head is what measures when each block became canonical; there is no chain event that carries that time
				return err
			}
			// Taken before the lookup, so the lookup's own latency never counts against the chain.
			seenAt := time.Now()
			head, lookupErr := el.blockRefByLabel(eth.Unsafe)
			if lookupErr != nil {
				logger.Warn("head lookup failed; will retry", "err", lookupErr)
			} else if head.Number < previous.Number || (head.Number == previous.Number && head.Hash != previous.Hash) {
				return fmt.Errorf("unsafe head went from %s to %s while the chain was held to its schedule", previous, head)
			} else {
				for number := previous.Number + 1; number <= head.Number; number++ {
					block, err := el.blockUnderHead(head, number)
					if err != nil {
						logger.Warn("block lookup failed; will retry from the last block checked", "block", number, "err", err)
						lookupErr = err
						break
					}
					lag, err := blockOnSchedule(previous, block, seenAt, blockTime)
					if err != nil {
						return err
					}
					maxLag = max(maxLag, lag)
					if lag > blockTime/2 {
						logger.Warn("Block on schedule but late", "block", block, "lag", lag.Round(time.Millisecond))
					} else {
						logger.Debug("Block on schedule", "block", block, "lag", lag.Round(time.Millisecond))
					}
					previous = block
				}
			}
			if overdue := nextBlockOverdue(previous, seenAt, blockTime); overdue > 0 {
				missed := fmt.Errorf("block %d missed its slot: not canonical on %s %s after its slot ended, the unsafe head was last seen at %s",
					previous.Number+1, el.inner.Name(), overdue.Round(time.Millisecond), previous)
				if lookupErr != nil {
					return fmt.Errorf("%w (the last lookup failed: %w)", missed, lookupErr)
				}
				return missed
			}
		}
		logger.Info("L2EL advanced on schedule", "at", previous, "max_lag", maxLag.Round(time.Millisecond))
		return nil
	}
}

// AdvancedOnSchedule waits for the unsafe head to be on schedule and then to advance blocks
// blocks without missing a slot; see AdvancedOnScheduleFn.
func (el *L2ELNode) AdvancedOnSchedule(blocks uint64, blockTime time.Duration) {
	el.require.NoError(el.AdvancedOnScheduleFn(blocks, blockTime)())
}

// blockUnderHead returns the block at number, which is head itself or one of its ancestors.
func (el *L2ELNode) blockUnderHead(head eth.L2BlockRef, number uint64) (eth.L2BlockRef, error) {
	if number == head.Number {
		return head, nil
	}
	ctx, cancel := context.WithTimeout(el.ctx, DefaultTimeout)
	defer cancel()
	block, err := el.inner.L2EthClient().L2BlockRefByNumber(ctx, number)
	if err != nil {
		return eth.L2BlockRef{}, fmt.Errorf("block %d is below the unsafe head %s but could not be fetched: %w", number, head, err)
	}
	return block, nil
}

// caughtUpToSchedule waits until the unsafe head's slot has not ended yet, that is until the
// chain has caught up with the wall clock, and returns that head.
func (el *L2ELNode) caughtUpToSchedule(logger log.Logger, blockTime, sampleInterval time.Duration) (eth.L2BlockRef, error) {
	timeout := scheduleCatchUpBlockTimes * blockTime
	deadline := time.Now().Add(timeout)
	var lastLogged eth.L2BlockRef
	for {
		head, err := el.blockRefByLabel(eth.Unsafe)
		if err != nil {
			logger.Warn("head lookup failed; will retry", "err", err)
		} else if behind := time.Since(slotEnd(head.Time, blockTime)); behind <= 0 {
			return head, nil
		} else if head.Hash != lastLogged.Hash {
			logger.Info("Chain is behind its schedule; waiting for it to catch up", "head", head, "behind", behind.Round(time.Millisecond))
			lastLogged = head
		}
		if time.Now().After(deadline) {
			return eth.L2BlockRef{}, fmt.Errorf("%s did not catch up with its schedule within %s", el.inner.Name(), timeout)
		}
		if err := clock.SystemClock.SleepCtx(el.ctx, sampleInterval); err != nil { // nosemgrep: flake-sleep-in-test -- the head is sampled until its slot is current; no chain event marks the catch-up
			return eth.L2BlockRef{}, err
		}
	}
}

// blockOnSchedule returns how long after its timestamp block was seen canonical, or an error when
// block does not follow previous on the chain's schedule: it must extend previous, carry
// previous's timestamp plus blockTime, and be seen before its slot ends.
func blockOnSchedule(previous, block eth.L2BlockRef, seenAt time.Time, blockTime time.Duration) (time.Duration, error) {
	if block.Number != previous.Number+1 || block.ParentHash != previous.Hash {
		return 0, fmt.Errorf("block %s does not extend %s", block, previous)
	}
	if expected := previous.Time + blockTimeSeconds(blockTime); block.Time != expected {
		return 0, fmt.Errorf("block %s has timestamp %d, not its parent's %d plus %s: either slots were skipped or %s is not the chain's block time",
			block, block.Time, previous.Time, blockTime, blockTime)
	}
	lag := seenAt.Sub(time.Unix(int64(block.Time), 0))
	if !seenAt.Before(slotEnd(block.Time, blockTime)) {
		return lag, fmt.Errorf("block %s missed its slot: canonical %s after its timestamp, at or past the block time %s",
			block, lag.Round(time.Millisecond), blockTime)
	}
	return lag, nil
}

// nextBlockOverdue returns how long past the end of its slot the block after previous is at
// seenAt, or zero or less while that slot is still open.
func nextBlockOverdue(previous eth.L2BlockRef, seenAt time.Time, blockTime time.Duration) time.Duration {
	return seenAt.Sub(slotEnd(previous.Time+blockTimeSeconds(blockTime), blockTime))
}

// slotEnd is when the next block is due after a block with the given timestamp.
func slotEnd(timestamp uint64, blockTime time.Duration) time.Time {
	return time.Unix(int64(timestamp), 0).Add(blockTime)
}

func blockTimeSeconds(blockTime time.Duration) uint64 {
	return uint64(blockTime / time.Second)
}
