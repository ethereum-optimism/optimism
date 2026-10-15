package dsl

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

const scheduleTestBlockTime = 2 * time.Second

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

func TestBlockOnSchedule(t *testing.T) {
	due := childDue()
	for _, tc := range []struct {
		name    string
		block   eth.L2BlockRef
		seenAt  time.Time
		wantLag time.Duration
		wantErr string
	}{
		{name: "canonical at its timestamp", block: scheduledChild(0), seenAt: due, wantLag: 0},
		{name: "sealed just before its timestamp", block: scheduledChild(0), seenAt: due.Add(-50 * time.Millisecond), wantLag: -50 * time.Millisecond},
		{name: "late but within its slot", block: scheduledChild(0), seenAt: due.Add(scheduleTestBlockTime - time.Millisecond), wantLag: scheduleTestBlockTime - time.Millisecond},
		{name: "missed its slot", block: scheduledChild(0), seenAt: due.Add(scheduleTestBlockTime), wantErr: "missed its slot"},
		{name: "skipped a slot", block: scheduledChild(2), seenAt: due.Add(2 * time.Second), wantErr: "slots were skipped"},
		{name: "does not extend the head", block: reorgedChild(), seenAt: due, wantErr: "does not extend"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lag, err := blockOnSchedule(scheduleTestHead, tc.block, tc.seenAt, scheduleTestBlockTime)
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
