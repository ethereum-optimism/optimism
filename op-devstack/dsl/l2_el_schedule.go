package dsl

import (
	"errors"
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
	// chain is held to its schedule. The time between two samples, at least the sample interval,
	// is credited to the chain: a block that becomes canonical within it past its slot end still
	// passes.
	scheduleSamplesPerBlock = 8
)

// AdvancedOnScheduleFn returns a CheckFunc that waits for the unsafe head to be on schedule, then
// holds the chain to its schedule for the next blocks blocks and fails as soon as a slot is
// missed: each block must extend the previous head, carry the previous head's timestamp plus
// blockTime, and be canonical before its own timestamp plus blockTime, when the next block is due.
func (el *L2ELNode) AdvancedOnScheduleFn(blocks uint64, blockTime time.Duration) CheckFunc {
	return func() error {
		_, err := el.advanceOnSchedule(blocks, blockTime)
		return err
	}
}

// AdvancedOnSchedule waits for the unsafe head to be on schedule and then to advance blocks
// blocks without missing a slot, see AdvancedOnScheduleFn, and returns the head the chain was
// held to its schedule from: the blocks held to it are the blocks numbered after it.
func (el *L2ELNode) AdvancedOnSchedule(blocks uint64, blockTime time.Duration) eth.L2BlockRef {
	from, err := el.advanceOnSchedule(blocks, blockTime)
	el.require.NoError(err)
	return from
}

func (el *L2ELNode) advanceOnSchedule(blocks uint64, blockTime time.Duration) (eth.L2BlockRef, error) {
	if blockTime <= 0 || blockTime%time.Second != 0 {
		return eth.L2BlockRef{}, fmt.Errorf("block time %s is not a positive whole number of seconds, which no L2 block timestamp can follow", blockTime)
	}
	logger := el.log.With("name", el.inner.Name(), "chain", el.ChainID(), "label", eth.Unsafe, "block_time", blockTime)
	sampleInterval := blockTime / scheduleSamplesPerBlock
	caughtUp, err := el.caughtUpToSchedule(logger, blockTime, sampleInterval)
	if err != nil {
		return eth.L2BlockRef{}, err
	}
	tracker := newScheduleTracker(caughtUp, blockTime)
	target := caughtUp.head.Number + blocks
	logger = logger.With("from", caughtUp.head.Number, "target", target)
	logger.Info("Expecting L2EL to advance on schedule")
	for tracker.previous.Number < target {
		if err := clock.SystemClock.SleepCtx(el.ctx, sampleInterval); err != nil { // nosemgrep: flake-sleep-in-test -- fixed-cadence sampling of the head is what measures when each block became canonical; there is no chain event that carries that time
			return eth.L2BlockRef{}, err
		}
		sample, headErr := el.sampleHead()
		if stalledFor := sample.stalledFor(blockTime); stalledFor > 0 {
			logger.Warn("Head lookup took longer than a block time; not crediting the time past it to the chain",
				"took", sample.answeredAt.Sub(sample.requestedAt).Round(time.Millisecond), "err", headErr)
		}
		if headErr != nil {
			logger.Warn("head lookup failed; will retry", "err", headErr)
			if err := tracker.lookupFailed(el.inner.Name(), sample, headErr); err != nil {
				return eth.L2BlockRef{}, err
			}
		} else if err := tracker.observe(sample); err != nil {
			return eth.L2BlockRef{}, err
		}
		blockErr, violation := tracker.judgePending(el.blockUnderHead, logger)
		if violation != nil {
			return eth.L2BlockRef{}, violation
		}
		if blockErr != nil {
			logger.Warn("block lookup failed; will retry from the last block checked", "block", tracker.previous.Number+1, "err", blockErr)
		}
		if err := tracker.checkNextBlock(el.inner.Name(), sample.requestedAt, errors.Join(headErr, blockErr)); err != nil {
			return eth.L2BlockRef{}, err
		}
	}
	logger.Info("L2EL advanced on schedule", "at", tracker.previous, "max_lag", tracker.maxLag.Round(time.Millisecond))
	return caughtUp.head, nil
}

// sampleHead looks up the unsafe head and records when the lookup was requested and answered, also
// when it failed.
func (el *L2ELNode) sampleHead() (headSample, error) {
	requestedAt := time.Now()
	head, err := el.blockRefByLabel(eth.Unsafe)
	if err != nil {
		return headSample{answeredAt: time.Now(), requestedAt: requestedAt}, err
	}
	return headSample{answeredAt: time.Now(), head: head, requestedAt: requestedAt}, nil
}

// blockUnderHead returns the block at number, which is head itself or one of its ancestors.
func (el *L2ELNode) blockUnderHead(head eth.L2BlockRef, number uint64) (eth.L2BlockRef, error) {
	if number == head.Number {
		return head, nil
	}
	block, err := el.blockRefByNumber(number)
	if err != nil {
		return eth.L2BlockRef{}, fmt.Errorf("block %d is below the unsafe head %s but could not be fetched: %w", number, head, err)
	}
	return block, nil
}

// caughtUpToSchedule waits until the unsafe head's slot has not ended yet, that is until the
// chain has caught up with the wall clock, and returns the sample of that head.
func (el *L2ELNode) caughtUpToSchedule(logger log.Logger, blockTime, sampleInterval time.Duration) (headSample, error) {
	timeout := scheduleCatchUpBlockTimes * blockTime
	deadline := time.Now().Add(timeout)
	var lastLogged eth.L2BlockRef
	for {
		sample, err := el.sampleHead()
		if err != nil {
			logger.Warn("head lookup failed; will retry", "err", err)
		} else if behind := time.Since(slotEnd(sample.head.Time, blockTime)); behind <= 0 {
			return sample, nil
		} else if sample.head.Hash != lastLogged.Hash {
			logger.Info("Chain is behind its schedule; waiting for it to catch up", "head", sample.head, "behind", behind.Round(time.Millisecond))
			lastLogged = sample.head
		}
		if time.Now().After(deadline) {
			return headSample{}, fmt.Errorf("%s did not catch up with its schedule within %s", el.inner.Name(), timeout)
		}
		if err := clock.SystemClock.SleepCtx(el.ctx, sampleInterval); err != nil { // nosemgrep: flake-sleep-in-test -- the head is sampled until its slot is current; no chain event marks the catch-up
			return headSample{}, err
		}
	}
}

// headSample is one lookup of the unsafe head and the wall-clock times it was requested and
// answered at. The chain had not advanced past head when the lookup was requested, and head was
// canonical by the time it was answered.
type headSample struct {
	answeredAt  time.Time
	head        eth.L2BlockRef
	requestedAt time.Time
}

// stalledFor is how much longer than blockTime the lookup took, or zero.
func (s headSample) stalledFor(blockTime time.Duration) time.Duration {
	return max(0, s.answeredAt.Sub(s.requestedAt)-blockTime)
}

// blockUnderHeadFetcher returns the block at number, which is head itself or one of its ancestors.
type blockUnderHeadFetcher func(head eth.L2BlockRef, number uint64) (eth.L2BlockRef, error)

// coveringSample is a sampled head that rose above every head sampled before it.
type coveringSample struct {
	// judgedAt is when the sample before it was requested, when no block above that sample's
	// head and up to this head was canonical yet, plus the time past a block time that each head
	// lookup from that sample through this one took. Each of those blocks fails when its slot had
	// ended by judgedAt. The gap between two samples is credited to the chain, since a slow
	// poller says nothing about the chain, but a lookup outstanding for longer than a block time
	// may be held up by the very stall it would otherwise hide.
	judgedAt time.Time
	sample   headSample
}

// scheduleTracker holds a chain to its schedule from the heads sampled off it.
type scheduleTracker struct {
	blockTime time.Duration

	// headLookupsFailingSince is when the first of the head lookups failing since the last
	// sample was requested, or zero while the last head lookup succeeded.
	headLookupsFailingSince time.Time
	// lastSampledAt is when the last sample was requested; a failed head lookup is no sample.
	lastSampledAt time.Time
	maxLag        time.Duration
	// pending are the sampled heads above previous, oldest first. Each block is judged against
	// the first of them it is at or under, the earliest sample that proved it canonical, even
	// when fetching the block failed at that sample and succeeded at a later one.
	pending  []coveringSample
	previous eth.L2BlockRef
	// stalledSinceLastSample is the time past a block time that each head lookup took, from the
	// last sample's on.
	stalledSinceLastSample time.Duration
}

func newScheduleTracker(start headSample, blockTime time.Duration) *scheduleTracker {
	return &scheduleTracker{
		blockTime: blockTime,

		lastSampledAt:          start.requestedAt,
		previous:               start.head,
		stalledSinceLastSample: start.stalledFor(blockTime),
	}
}

// latestHead is the highest head sampled so far.
func (s *scheduleTracker) latestHead() eth.L2BlockRef {
	if len(s.pending) == 0 {
		return s.previous
	}
	return s.pending[len(s.pending)-1].sample.head
}

// observe records a sampled head, or fails when the head went back or was replaced.
func (s *scheduleTracker) observe(sample headSample) error {
	latest := s.latestHead()
	if sample.head.Number < latest.Number || (sample.head.Number == latest.Number && sample.head.Hash != latest.Hash) {
		return fmt.Errorf("unsafe head went from %s to %s while the chain was held to its schedule", latest, sample.head)
	}
	stalledFor := sample.stalledFor(s.blockTime)
	if sample.head.Number > latest.Number {
		judgedAt := s.lastSampledAt.Add(s.stalledSinceLastSample + stalledFor)
		s.pending = append(s.pending, coveringSample{judgedAt: judgedAt, sample: sample})
	}
	s.headLookupsFailingSince = time.Time{}
	s.lastSampledAt = sample.requestedAt
	s.stalledSinceLastSample = stalledFor
	return nil
}

// lookupFailed records a head lookup that failed, so the time past a block time it took is not
// credited to the chain. The blocks the lookup might have seen are judged by the next sample, and
// head lookups failing for longer than a block time fail the check.
func (s *scheduleTracker) lookupFailed(elName string, sample headSample, lookupErr error) error {
	s.stalledSinceLastSample += sample.stalledFor(s.blockTime)
	if s.headLookupsFailingSince.IsZero() {
		s.headLookupsFailingSince = sample.requestedAt
	}
	if failingFor := sample.requestedAt.Sub(s.headLookupsFailingSince); failingFor > s.blockTime {
		return fmt.Errorf("unsafe head lookups on %s failed for %s: %w", elName, failingFor.Round(time.Millisecond), lookupErr)
	}
	return nil
}

// judgePending judges every block under the pending heads against the sample that first proved
// it canonical and the sample before it. It returns a violation for the first block off its
// schedule, or stops at the first block it could not fetch and returns that lookup's error,
// keeping the block pending.
func (s *scheduleTracker) judgePending(fetch blockUnderHeadFetcher, logger log.Logger) (lookupErr error, violation error) {
	for len(s.pending) > 0 {
		covering := s.pending[0]
		for s.previous.Number < covering.sample.head.Number {
			block, err := fetch(covering.sample.head, s.previous.Number+1)
			if err != nil {
				return err, nil
			}
			seenAt := covering.sample.answeredAt
			lag, err := blockOnSchedule(s.previous, block, covering.judgedAt, seenAt, s.blockTime)
			if err != nil {
				return nil, err
			}
			s.maxLag = max(s.maxLag, lag)
			switch {
			case lag >= s.blockTime:
				logger.Warn("Block first seen past its slot end; crediting the time between samples, from inside its slot, to the chain",
					"block", block, "lag", lag.Round(time.Millisecond), "credited", seenAt.Sub(covering.judgedAt).Round(time.Millisecond))
			case lag > s.blockTime/2:
				logger.Warn("Block on schedule but late", "block", block, "lag", lag.Round(time.Millisecond))
			default:
				logger.Debug("Block on schedule", "block", block, "lag", lag.Round(time.Millisecond))
			}
			s.previous = block
		}
		s.pending = s.pending[1:]
	}
	return nil, nil
}

// checkNextBlock fails once the block after previous is overdue at the last sample, which had not
// seen it; a failed head lookup since then is judged by the next sample instead. A block a sampled
// head already proved canonical is never overdue; its fetch is retried for a block time after
// that sample, until now, then the check fails with the lookup's error.
func (s *scheduleTracker) checkNextBlock(elName string, now time.Time, lookupErr error) error {
	next := s.previous.Number + 1
	if len(s.pending) > 0 {
		covering := s.pending[0].sample
		if unfetchedFor := now.Sub(covering.requestedAt); unfetchedFor > s.blockTime {
			return fmt.Errorf("block %d is canonical on %s under the unsafe head %s but could not be fetched for %s: %w",
				next, elName, covering.head, unfetchedFor.Round(time.Millisecond), lookupErr)
		}
		return nil
	}
	overdue := nextBlockOverdue(s.previous, s.lastSampledAt, s.blockTime)
	if overdue <= 0 {
		return nil
	}
	return fmt.Errorf("block %d missed its slot: not canonical on %s %s after its slot ended, the unsafe head was last seen at %s",
		next, elName, overdue.Round(time.Millisecond), s.previous)
}

// blockOnSchedule returns how long after its timestamp block was seen canonical, or an error when
// block does not follow previous on the chain's schedule: it must extend previous, carry
// previous's timestamp plus blockTime, and its slot must not have ended by judgedAt, a time no
// sample had seen block canonical by.
func blockOnSchedule(previous, block eth.L2BlockRef, judgedAt, seenAt time.Time, blockTime time.Duration) (time.Duration, error) {
	if block.Number != previous.Number+1 || block.ParentHash != previous.Hash {
		return 0, fmt.Errorf("block %s does not extend %s", block, previous)
	}
	if expected := previous.Time + blockTimeSeconds(blockTime); block.Time != expected {
		return 0, fmt.Errorf("block %s has timestamp %d, not its parent's %d plus %s: either slots were skipped or %s is not the chain's block time",
			block, block.Time, previous.Time, blockTime, blockTime)
	}
	timestamp := time.Unix(int64(block.Time), 0)
	if !judgedAt.Before(slotEnd(block.Time, blockTime)) {
		return 0, fmt.Errorf("block %s missed its slot: not seen canonical yet %s after its timestamp, at or past the block time %s",
			block, judgedAt.Sub(timestamp).Round(time.Millisecond), blockTime)
	}
	return seenAt.Sub(timestamp), nil
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
