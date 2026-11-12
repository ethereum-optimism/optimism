package dsl

import (
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

const (
	scheduleTestBlockTime      = 2 * time.Second
	scheduleTestSampleInterval = scheduleTestBlockTime / scheduleSamplesPerBlock
)

var scheduleTestHead = eth.L2BlockRef{Hash: common.HexToHash("0xaa"), Number: 10, Time: 1_700_000_000}

// scheduledChild returns the block that extends scheduleTestHead, timestamped timestampOffset
// seconds off the block time after its parent.
func scheduledChild(timestampOffset int64) eth.L2BlockRef {
	return eth.L2BlockRef{
		Hash:       common.HexToHash("0xbb"),
		Number:     scheduleTestHead.Number + 1,
		ParentHash: scheduleTestHead.Hash,
		Time:       uint64(int64(scheduleTestHead.Time+blockTimeSeconds(scheduleTestBlockTime)) + timestampOffset),
	}
}

func reorgedChild() eth.L2BlockRef {
	child := scheduledChild(0)
	child.ParentHash = common.HexToHash("0xcc")
	return child
}

// childDue is the timestamp of the block that follows scheduleTestHead on schedule.
func childDue() time.Time {
	return time.Unix(int64(scheduledChild(0).Time), 0)
}

func TestAdvanceOnScheduleRejectsBlockTime(t *testing.T) {
	for _, tc := range []struct {
		name      string
		blockTime time.Duration
	}{
		{name: "unset", blockTime: 0},
		{name: "negative", blockTime: -time.Second},
		{name: "fractional seconds", blockTime: 1500 * time.Millisecond},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := (&L2ELNode{}).advanceOnSchedule(1, tc.blockTime)
			require.ErrorContains(t, err, "is not a positive whole number of seconds")
		})
	}
}

func TestBlockOnSchedule(t *testing.T) {
	due := childDue()
	slotEnd := due.Add(scheduleTestBlockTime)
	for _, tc := range []struct {
		name     string
		block    eth.L2BlockRef
		judgedAt time.Time
		seenAt   time.Time
		wantLag  time.Duration
		wantErr  string
	}{
		{name: "canonical at its timestamp", block: scheduledChild(0), judgedAt: due.Add(-time.Second), seenAt: due, wantLag: 0},
		{name: "sealed just before its timestamp", block: scheduledChild(0), judgedAt: due.Add(-time.Second), seenAt: due.Add(-50 * time.Millisecond), wantLag: -50 * time.Millisecond},
		{name: "late but within its slot", block: scheduledChild(0), judgedAt: due, seenAt: slotEnd.Add(-time.Millisecond), wantLag: scheduleTestBlockTime - time.Millisecond},
		{name: "seen past its slot end, judged inside it", block: scheduledChild(0), judgedAt: slotEnd.Add(-time.Millisecond), seenAt: slotEnd.Add(time.Second), wantLag: scheduleTestBlockTime + time.Second},
		{name: "judged at its slot end", block: scheduledChild(0), judgedAt: slotEnd, seenAt: slotEnd.Add(time.Millisecond), wantErr: "missed its slot"},
		{name: "skipped a slot", block: scheduledChild(2), judgedAt: due, seenAt: due.Add(2 * time.Second), wantErr: "slots were skipped"},
		{name: "does not extend the head", block: reorgedChild(), judgedAt: due.Add(-time.Second), seenAt: due, wantErr: "does not extend"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lag, err := blockOnSchedule(scheduleTestHead, tc.block, tc.judgedAt, tc.seenAt, scheduleTestBlockTime)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantLag, lag)
		})
	}
}

func TestNextBlockOverdue(t *testing.T) {
	due := childDue()
	for _, tc := range []struct {
		name        string
		seenAt      time.Time
		wantOverdue time.Duration
	}{
		{name: "before the next block is due", seenAt: due.Add(-time.Second), wantOverdue: -scheduleTestBlockTime - time.Second},
		{name: "when the next block is due", seenAt: due, wantOverdue: -scheduleTestBlockTime},
		{name: "last instant of the next block's slot", seenAt: due.Add(scheduleTestBlockTime - time.Millisecond), wantOverdue: -time.Millisecond},
		{name: "when the next block's slot ends", seenAt: due.Add(scheduleTestBlockTime), wantOverdue: 0},
		{name: "past the next block's slot", seenAt: due.Add(scheduleTestBlockTime + time.Second), wantOverdue: time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.wantOverdue, nextBlockOverdue(scheduleTestHead, tc.seenAt, scheduleTestBlockTime))
		})
	}
}

// scheduledChain returns length blocks that extend scheduleTestHead on schedule, keyed by number.
func scheduledChain(length int) map[uint64]eth.L2BlockRef {
	chain := make(map[uint64]eth.L2BlockRef, length)
	parent := scheduleTestHead
	for range length {
		block := eth.L2BlockRef{
			Hash:       common.BigToHash(new(big.Int).SetUint64(parent.Number + 1)),
			Number:     parent.Number + 1,
			ParentHash: parent.Hash,
			Time:       parent.Time + blockTimeSeconds(scheduleTestBlockTime),
		}
		chain[block.Number] = block
		parent = block
	}
	return chain
}

// flakyBlockFetcher serves blocks from a chain, failing each block's first failures[number] lookups.
type flakyBlockFetcher struct {
	chain    map[uint64]eth.L2BlockRef
	failures map[uint64]int
}

func (f *flakyBlockFetcher) fetch(_ eth.L2BlockRef, number uint64) (eth.L2BlockRef, error) {
	if f.failures[number] > 0 {
		f.failures[number]--
		return eth.L2BlockRef{}, errors.New("block lookup failed")
	}
	return f.chain[number], nil
}

func sampleOf(head eth.L2BlockRef, requestedAt time.Time, latency time.Duration) headSample {
	return headSample{answeredAt: requestedAt.Add(latency), head: head, requestedAt: requestedAt}
}

// headLookup is one lookup of the head as the schedule check makes it: a sample, or a failure.
type headLookup struct {
	err    error
	sample headSample
}

func lookupOf(head eth.L2BlockRef, requestedAt time.Time, latency time.Duration) headLookup {
	return headLookup{sample: sampleOf(head, requestedAt, latency)}
}

func failedLookup(requestedAt time.Time, latency time.Duration) headLookup {
	return headLookup{err: errors.New("head lookup failed"), sample: headSample{answeredAt: requestedAt.Add(latency), requestedAt: requestedAt}}
}

// trackerFrom returns a tracker held to the schedule from scheduleTestHead, sampled at startedAt.
func trackerFrom(startedAt time.Time) *scheduleTracker {
	return newScheduleTracker(sampleOf(scheduleTestHead, startedAt, 0), scheduleTestBlockTime)
}

// trackLookups feeds lookups to the tracker in order, as the schedule check does, and returns the
// first failure the tracker reports.
func trackLookups(t *testing.T, tracker *scheduleTracker, fetcher *flakyBlockFetcher, lookups ...headLookup) error {
	logger := testlog.Logger(t, log.LevelDebug)
	for _, lookup := range lookups {
		if lookup.err != nil {
			if err := tracker.lookupFailed("sequencer", lookup.sample, lookup.err); err != nil {
				return err
			}
		} else if err := tracker.observe(lookup.sample); err != nil {
			return err
		}
		blockErr, violation := tracker.judgePending(fetcher.fetch, logger)
		if violation != nil {
			return violation
		}
		if err := tracker.checkNextBlock("sequencer", lookup.sample.requestedAt, errors.Join(lookup.err, blockErr)); err != nil {
			return err
		}
	}
	return nil
}

// sampleRepeatedly samples the same head count times, one sample interval apart from
// firstRequestedAt, and returns the first failure the tracker reports.
func sampleRepeatedly(t *testing.T, tracker *scheduleTracker, fetcher *flakyBlockFetcher, head eth.L2BlockRef, firstRequestedAt time.Time, count int) error {
	lookups := make([]headLookup, count)
	for i := range lookups {
		lookups[i] = lookupOf(head, firstRequestedAt.Add(time.Duration(i)*scheduleTestSampleInterval), 0)
	}
	return trackLookups(t, tracker, fetcher, lookups...)
}

func TestHeadSampleStalledFor(t *testing.T) {
	for _, tc := range []struct {
		name    string
		latency time.Duration
		want    time.Duration
	}{
		{name: "lookup answered at once", latency: 0, want: 0},
		{name: "lookup answered in a block time", latency: scheduleTestBlockTime, want: 0},
		{name: "lookup answered past a block time", latency: scheduleTestBlockTime + time.Millisecond, want: time.Millisecond},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, sampleOf(scheduledChild(0), childDue(), tc.latency).stalledFor(scheduleTestBlockTime))
		})
	}
}

// TestScheduleTrackerJudgesBlockByLastSampleWithoutIt samples scheduleTestHead a last time at
// unseenAt, relative to the end of its child's slot and answered after unseenLatency, then samples
// the child, requested at seenAt and answered after seenLatency. The child fails only when the
// sample without it was requested at or after its slot end, once each lookup's time past a block
// time is added to that request.
func TestScheduleTrackerJudgesBlockByLastSampleWithoutIt(t *testing.T) {
	for _, tc := range []struct {
		name          string
		unseenAt      time.Duration
		unseenLatency time.Duration
		seenAt        time.Duration
		seenLatency   time.Duration
		wantErr       string
	}{
		{name: "seen inside its slot", unseenAt: -200 * time.Millisecond, seenAt: -75 * time.Millisecond},
		{name: "sealed between samples straddling its slot end", unseenAt: -50 * time.Millisecond, seenAt: 75 * time.Millisecond},
		{name: "seen after the sampler stalled past its slot end", unseenAt: -scheduleTestBlockTime + 100*time.Millisecond, seenAt: 900 * time.Millisecond},
		{name: "lookup of it answered in a block time, past its slot end", unseenAt: -200 * time.Millisecond, seenAt: -75 * time.Millisecond, seenLatency: scheduleTestBlockTime},
		{name: "lookup of it stalled, the time past a block time inside its slot", unseenAt: -500 * time.Millisecond, seenAt: -375 * time.Millisecond, seenLatency: scheduleTestBlockTime + 300*time.Millisecond},
		{name: "lookup of it stalled past its slot end and a block time", unseenAt: -200 * time.Millisecond, seenAt: -75 * time.Millisecond, seenLatency: scheduleTestBlockTime + 300*time.Millisecond, wantErr: "missed its slot"},
		{name: "lookup of the sample before answered in a block time", unseenAt: -time.Second, unseenLatency: scheduleTestBlockTime - 100*time.Millisecond, seenAt: time.Second},
		{name: "lookup of the sample before stalled past its slot end and a block time", unseenAt: -time.Second, unseenLatency: scheduleTestBlockTime + 1500*time.Millisecond, seenAt: 2625 * time.Millisecond, wantErr: "missed its slot"},
		{name: "unseen at its slot end", unseenAt: 0, seenAt: scheduleTestSampleInterval, wantErr: "missed its slot"},
		{name: "unseen past its slot end", unseenAt: time.Millisecond, seenAt: scheduleTestSampleInterval, wantErr: "missed its slot"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			chain := scheduledChain(1)
			child := chain[scheduleTestHead.Number+1]
			slotEnd := childDue().Add(scheduleTestBlockTime)
			tracker := trackerFrom(childDue().Add(-scheduleTestBlockTime))

			err := trackLookups(t, tracker, &flakyBlockFetcher{chain: chain},
				lookupOf(scheduleTestHead, slotEnd.Add(tc.unseenAt), tc.unseenLatency),
				lookupOf(child, slotEnd.Add(tc.seenAt), tc.seenLatency))

			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, child, tracker.previous)
		})
	}
}

// TestScheduleTrackerHoldsChainToFailedHeadLookups samples scheduleTestHead a last time 200ms
// before the end of its child's slot, then fails a head lookup requested as that sample is
// answered and taking failedLatency, then samples the child, requested at seenAt relative to the
// child's slot end. The time past a block time the failed lookup took is not credited to the
// chain.
func TestScheduleTrackerHoldsChainToFailedHeadLookups(t *testing.T) {
	for _, tc := range []struct {
		name          string
		failedLatency time.Duration
		seenAt        time.Duration
		wantErr       string
	}{
		{name: "failed lookup answered in a block time", failedLatency: scheduleTestBlockTime - 100*time.Millisecond, seenAt: 1800 * time.Millisecond},
		{name: "failed lookup stalled, the time past a block time inside the slot", failedLatency: scheduleTestBlockTime + 100*time.Millisecond, seenAt: 2 * time.Second},
		{name: "failed lookup stalled past the slot end and a block time", failedLatency: scheduleTestBlockTime + 300*time.Millisecond, seenAt: 2200 * time.Millisecond, wantErr: "missed its slot"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			chain := scheduledChain(1)
			child := chain[scheduleTestHead.Number+1]
			slotEnd := childDue().Add(scheduleTestBlockTime)
			unseenAt := slotEnd.Add(-200 * time.Millisecond)
			tracker := trackerFrom(childDue().Add(-scheduleTestBlockTime))

			err := trackLookups(t, tracker, &flakyBlockFetcher{chain: chain},
				lookupOf(scheduleTestHead, unseenAt, 0),
				failedLookup(unseenAt, tc.failedLatency),
				lookupOf(child, slotEnd.Add(tc.seenAt), 0))

			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, child, tracker.previous)
		})
	}
}

// TestScheduleTrackerJudgesFailedHeadLookupsByNextSample samples scheduleTestHead a last time
// 100ms before the end of its child's slot, then fails failedLookups head lookups one sample
// interval apart from 25ms past that slot end, then samples the block seenBlocks above
// scheduleTestHead one sample interval after the last failure.
func TestScheduleTrackerJudgesFailedHeadLookupsByNextSample(t *testing.T) {
	for _, tc := range []struct {
		name          string
		failedLookups int
		seenBlocks    uint64
		wantErr       string
	}{
		{name: "one failed lookup past its slot end, then seen", failedLookups: 1, seenBlocks: 1},
		{name: "lookups failing for a block time, then it and its child seen", failedLookups: scheduleSamplesPerBlock + 1, seenBlocks: 2},
		{name: "lookups failing past a block time", failedLookups: scheduleSamplesPerBlock + 2, seenBlocks: 2, wantErr: "unsafe head lookups on sequencer failed"},
		{name: "one failed lookup past its slot end, then unseen", failedLookups: 1, seenBlocks: 0, wantErr: "missed its slot"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			chain := scheduledChain(2)
			chain[scheduleTestHead.Number] = scheduleTestHead
			seen := chain[scheduleTestHead.Number+tc.seenBlocks]
			slotEnd := childDue().Add(scheduleTestBlockTime)
			tracker := trackerFrom(childDue().Add(-scheduleTestBlockTime))
			lookups := []headLookup{lookupOf(scheduleTestHead, slotEnd.Add(-100*time.Millisecond), 0)}
			lookupAt := slotEnd.Add(25 * time.Millisecond)
			for range tc.failedLookups {
				lookups = append(lookups, failedLookup(lookupAt, 0))
				lookupAt = lookupAt.Add(scheduleTestSampleInterval)
			}
			lookups = append(lookups, lookupOf(seen, lookupAt, 0))

			err := trackLookups(t, tracker, &flakyBlockFetcher{chain: chain}, lookups...)

			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, seen, tracker.previous)
		})
	}
}

// TestScheduleTrackerRestartsHeadLookupFailuresAtEachSample fails two head lookups more than a
// block time apart with a sample between them, which must not count as failing for that long.
func TestScheduleTrackerRestartsHeadLookupFailuresAtEachSample(t *testing.T) {
	chain := scheduledChain(1)
	child := chain[scheduleTestHead.Number+1]
	tracker := trackerFrom(childDue().Add(-scheduleTestBlockTime))

	err := trackLookups(t, tracker, &flakyBlockFetcher{chain: chain},
		failedLookup(childDue().Add(-scheduleTestBlockTime+scheduleTestSampleInterval), 0),
		lookupOf(child, childDue().Add(100*time.Millisecond), 0),
		failedLookup(childDue().Add(time.Second), 0))

	require.NoError(t, err)
	require.Equal(t, child, tracker.previous)
}

// TestScheduleTrackerRetriesFailedBlockLookups samples a head two blocks above the tracked one,
// sealed just before its timestamp and so inside its parent's slot, while the lookup of that
// parent fails; later samples fall past the parent's slot end.
func TestScheduleTrackerRetriesFailedBlockLookups(t *testing.T) {
	for _, tc := range []struct {
		name          string
		fetchFailures int
		wantErr       string
	}{
		{name: "one failed lookup", fetchFailures: 1},
		{name: "lookups failing for a block time", fetchFailures: scheduleSamplesPerBlock},
		{name: "lookups failing past a block time", fetchFailures: scheduleSamplesPerBlock + 2, wantErr: "could not be fetched"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			chain := scheduledChain(2)
			parent, head := chain[scheduleTestHead.Number+1], chain[scheduleTestHead.Number+2]
			fetcher := &flakyBlockFetcher{chain: chain, failures: map[uint64]int{parent.Number: tc.fetchFailures}}
			sealedAt := time.Unix(int64(head.Time), 0).Add(-50 * time.Millisecond)
			tracker := trackerFrom(sealedAt.Add(-scheduleTestSampleInterval))

			err := sampleRepeatedly(t, tracker, fetcher, head, sealedAt, scheduleSamplesPerBlock+2)

			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				require.NotContains(t, err.Error(), "missed its slot")
				return
			}
			require.NoError(t, err)
			require.Equal(t, head, tracker.previous)
		})
	}
}
