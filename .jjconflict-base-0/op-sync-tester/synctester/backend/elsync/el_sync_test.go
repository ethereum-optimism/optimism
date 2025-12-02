package elsync

import (
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"
)

func runStatuses(p eth.ELSyncPolicy, nums ...uint64) []eth.ExecutePayloadStatus {
	out := make([]eth.ExecutePayloadStatus, 0, len(nums))
	for _, n := range nums {
		out = append(out, p.ELSyncStatus(n))
	}
	return out
}

func TestWindowSyncPolicy_NewWindowSyncPolicy_PanicsOnInvalidArgs(t *testing.T) {
	tests := []struct {
		name          string
		cnt           uint64
		maxSize       uint64
		wantSubstring string
	}{
		{
			name:          "cnt greater than maxSize",
			cnt:           uint64(3),
			maxSize:       uint64(2),
			wantSubstring: "less than window size",
		},
		{
			name:          "cnt is zero",
			cnt:           uint64(0),
			maxSize:       uint64(5),
			wantSubstring: "not positive",
		},
		{
			name:          "maxSize is zero",
			cnt:           uint64(2),
			maxSize:       uint64(0),
			wantSubstring: "not positive",
		},
		{
			name:          "both zero",
			cnt:           uint64(0),
			maxSize:       uint64(0),
			wantSubstring: "not positive",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				r := recover()
				require.NotNil(t, r, "expected panic")
				require.Contains(t, fmt.Sprint(r), tc.wantSubstring)
			}()
			NewWindowSyncPolicy(tc.cnt, tc.maxSize)
		})
	}
}

func TestWindowSyncPolicy_MonotonicConsecutive_DefaultCnt2(t *testing.T) {
	p := DefaultELSyncPolicy()

	// Need two consecutive observations ending at current num to become VALID.
	got := runStatuses(p, 10, 11, 12)
	require.Equal(t, []eth.ExecutePayloadStatus{
		eth.ExecutionSyncing, // only 10 observed
		eth.ExecutionValid,   // [10,11] consecutive window ending at 11
		eth.ExecutionValid,   // [11,12] consecutive window ending at 12
	}, got)
}

func TestWindowSyncPolicy_GapThenFill_Cnt3(t *testing.T) {
	p := NewWindowSyncPolicy(3, 5)

	// 10 -> SYNC, 12 -> SYNC (gap), 11 -> still SYNC (len < 3),
	// 12 -> now have [10,11,12] ending at 12 => VALID
	got := runStatuses(p, 10, 12, 11, 12)
	require.Equal(t, []eth.ExecutePayloadStatus{
		eth.ExecutionSyncing,
		eth.ExecutionSyncing,
		eth.ExecutionSyncing,
		eth.ExecutionValid,
	}, got)
}

func TestWindowSyncPolicy_Reorg_DropGreaterOrEqual_Current(t *testing.T) {
	p := NewWindowSyncPolicy(2, 5)

	// Build up to 12 => VALID, then reorg to 9 (drop >= 9) => SYNC,
	// then 10 => VALID with window [9,10].
	got := runStatuses(p, 10, 11, 12, 9, 10)
	require.Equal(t, []eth.ExecutePayloadStatus{
		eth.ExecutionSyncing, // 10
		eth.ExecutionValid,   // 11
		eth.ExecutionValid,   // 12
		eth.ExecutionSyncing, // 9 (reorg down)
		eth.ExecutionValid,   // 10 [9,10]
	}, got)
}

func TestWindowSyncPolicy_Duplicates_DoNotAdvance(t *testing.T) {
	p := NewWindowSyncPolicy(2, 5)

	// Duplicate 10 should not create validity; need 11 next.
	got := runStatuses(p, 10, 10, 11)
	require.Equal(t, []eth.ExecutePayloadStatus{
		eth.ExecutionSyncing,
		eth.ExecutionSyncing, // duplicate doesn't satisfy window [9,10] or [10,11]
		eth.ExecutionValid,   // 11 with prior 10
	}, got)
}

func TestWindowSyncPolicy_MaxSizeTrimming_DoesNotBreakValidity(t *testing.T) {
	p := NewWindowSyncPolicy(3, 3) // window == maxSize

	// 5,6,7 => VALID at 7 (have [5,6,7])
	// 8 => cache should trim oldest (effectively [6,7,8]), still VALID
	got := runStatuses(p, 5, 6, 7, 8)
	require.Equal(t, []eth.ExecutePayloadStatus{
		eth.ExecutionSyncing, // 5
		eth.ExecutionSyncing, // 6
		eth.ExecutionValid,   // 7 (have 5,6,7)
		eth.ExecutionValid,   // 8 (have 6,7,8 after trim)
	}, got)
}

func TestWindowSyncPolicy_LongerRun_StabilityAcrossAdvances(t *testing.T) {
	p := NewWindowSyncPolicy(3, 5)
	seq := []uint64{100, 101, 102, 103, 104}
	want := []eth.ExecutePayloadStatus{
		eth.ExecutionSyncing, // 100
		eth.ExecutionSyncing, // 101
		eth.ExecutionValid,   // 102
		eth.ExecutionValid,   // 103
		eth.ExecutionValid,   // 104
	}
	require.Equal(t, want, runStatuses(p, seq...))
}

func TestWindowSyncPolicy_Cnt1_AlwaysValidAfterSingleObservation(t *testing.T) {
	p := NewWindowSyncPolicy(1, 5)

	// With cnt=1, a single observation of the current head is enough to be VALID,
	// regardless of gaps or reorg-like regressions.
	got := runStatuses(p, 10, 12, 11, 50, 49)
	require.Equal(t, []eth.ExecutePayloadStatus{
		eth.ExecutionValid, // 10
		eth.ExecutionValid, // 12
		eth.ExecutionValid, // 11
		eth.ExecutionValid, // 50
		eth.ExecutionValid, // 49
	}, got)
}

func TestWindowSyncPolicy_MaxSize1_WithCnt1(t *testing.T) {
	p := NewWindowSyncPolicy(1, 1)

	// Cache can only hold the last number, but with cnt=1 that's sufficient.
	got := runStatuses(p, 5, 6, 4, 4, 7)
	require.Equal(t, []eth.ExecutePayloadStatus{
		eth.ExecutionValid, // 5
		eth.ExecutionValid, // 6
		eth.ExecutionValid, // 4 (reorg-like; still VALID with cnt=1)
		eth.ExecutionValid, // 4 (duplicate)
		eth.ExecutionValid, // 7
	}, got)
}
