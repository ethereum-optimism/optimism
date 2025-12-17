package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSundayMidnightUTC(t *testing.T) {
	result := sundayMidnightUTC()

	// Convert to time for easier assertions
	resultTime := time.Unix(int64(result), 0).UTC()

	// Should be a Sunday
	assert.Equal(t, time.Sunday, resultTime.Weekday(), "should be a Sunday")

	// Should be midnight (00:00:00)
	assert.Equal(t, 0, resultTime.Hour(), "should be midnight hour")
	assert.Equal(t, 0, resultTime.Minute(), "should be midnight minute")
	assert.Equal(t, 0, resultTime.Second(), "should be midnight second")

	// Should be in the past or now (not in the future)
	now := time.Now().UTC()
	assert.True(t, resultTime.Before(now) || resultTime.Equal(now), "should not be in the future")

	// Should be within the last 7 days
	weekAgo := now.AddDate(0, 0, -7)
	assert.True(t, resultTime.After(weekAgo), "should be within the last 7 days")
}

func TestSundayMidnightForTime(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected time.Time
	}{
		{
			name:     "Sunday returns same day midnight",
			input:    time.Date(2025, 12, 14, 15, 30, 0, 0, time.UTC), // Sunday 3:30 PM
			expected: time.Date(2025, 12, 14, 0, 0, 0, 0, time.UTC),   // Sunday midnight
		},
		{
			name:     "Monday returns previous Sunday",
			input:    time.Date(2025, 12, 15, 10, 0, 0, 0, time.UTC), // Monday 10 AM
			expected: time.Date(2025, 12, 14, 0, 0, 0, 0, time.UTC),  // Previous Sunday
		},
		{
			name:     "Saturday returns previous Sunday",
			input:    time.Date(2025, 12, 20, 23, 59, 0, 0, time.UTC), // Saturday 11:59 PM
			expected: time.Date(2025, 12, 14, 0, 0, 0, 0, time.UTC),   // Previous Sunday
		},
		{
			name:     "Wednesday returns previous Sunday",
			input:    time.Date(2025, 12, 17, 12, 0, 0, 0, time.UTC), // Wednesday noon
			expected: time.Date(2025, 12, 14, 0, 0, 0, 0, time.UTC),  // Previous Sunday
		},
		{
			name:     "Sunday midnight returns same",
			input:    time.Date(2025, 12, 14, 0, 0, 0, 0, time.UTC), // Sunday midnight
			expected: time.Date(2025, 12, 14, 0, 0, 0, 0, time.UTC), // Same
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sundayMidnightForTime(tt.input)
			assert.Equal(t, uint64(tt.expected.Unix()), result)
		})
	}
}

func TestEstimateBlock(t *testing.T) {
	tests := []struct {
		name            string
		refBlock        uint64
		refTimestamp    uint64
		blockTime       uint64
		targetTimestamp uint64
		expected        uint64
	}{
		{
			name:            "Target after reference",
			refBlock:        1000000,
			refTimestamp:    1700000000,
			blockTime:       12,
			targetTimestamp: 1700000120, // 120 seconds later
			expected:        1000010,    // 10 blocks later
		},
		{
			name:            "Target before reference",
			refBlock:        1000000,
			refTimestamp:    1700000000,
			blockTime:       12,
			targetTimestamp: 1699999880, // 120 seconds earlier
			expected:        999990,     // 10 blocks earlier
		},
		{
			name:            "Target equals reference",
			refBlock:        1000000,
			refTimestamp:    1700000000,
			blockTime:       12,
			targetTimestamp: 1700000000,
			expected:        1000000,
		},
		{
			name:            "Large time difference",
			refBlock:        21000000,
			refTimestamp:    1729345547,
			blockTime:       12,
			targetTimestamp: 1765670400, // ~36M seconds later
			expected:        24027071,   // ~3M blocks later
		},
		{
			name:            "Partial block time rounds down",
			refBlock:        1000000,
			refTimestamp:    1700000000,
			blockTime:       12,
			targetTimestamp: 1700000011, // 11 seconds (less than 1 block)
			expected:        1000000,    // No change (rounds down)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := estimateBlock(tt.refBlock, tt.refTimestamp, tt.blockTime, tt.targetTimestamp)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEstimateBlockConsistency(t *testing.T) {
	// Test that estimating forward then backward returns to the same block
	refBlock := uint64(21000000)
	refTimestamp := uint64(1729345547)
	blockTime := uint64(12)

	// Go forward 1000 blocks worth of time
	futureTimestamp := refTimestamp + (1000 * blockTime)
	futureBlock := estimateBlock(refBlock, refTimestamp, blockTime, futureTimestamp)
	assert.Equal(t, refBlock+1000, futureBlock)

	// From that future point, go back to original timestamp
	backBlock := estimateBlock(futureBlock, futureTimestamp, blockTime, refTimestamp)
	assert.Equal(t, refBlock, backBlock)
}

// Integration test - only runs if RPC is available
func TestBinarySearchBlock_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Test with a known timestamp and verify the result is accurate
	// Sunday Dec 14, 2025 00:00:00 UTC = 1765670400
	targetTimestamp := uint64(1765670400)
	estimate := estimateBlock(mainnetRefBlock, mainnetRefTimestamp, mainnetBlockTime, targetTimestamp)

	result, err := binarySearchBlock(defaultMainnetRPC, targetTimestamp, estimate, 0, false) // 0 = exact search for test
	require.NoError(t, err)

	// Verify the result by checking block timestamps
	resultTs, err := getBlockTimestamp(defaultMainnetRPC, result)
	require.NoError(t, err)
	assert.LessOrEqual(t, resultTs, targetTimestamp, "result block timestamp should be <= target")

	// Next block should be after target
	nextTs, err := getBlockTimestamp(defaultMainnetRPC, result+1)
	require.NoError(t, err)
	assert.Greater(t, nextTs, targetTimestamp, "next block timestamp should be > target")
}

func TestGetLatestBlockNumber_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	blockNum, err := getLatestBlockNumber(defaultMainnetRPC)
	require.NoError(t, err)
	assert.Greater(t, blockNum, uint64(20000000), "block number should be reasonable for mainnet")
}

func TestGetBlockTimestamp_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Test with a known block
	timestamp, err := getBlockTimestamp(defaultMainnetRPC, mainnetRefBlock)
	require.NoError(t, err)
	assert.Equal(t, mainnetRefTimestamp, timestamp, "should match known timestamp")
}

// sundayMidnightForTime is a testable version that takes a specific time
func sundayMidnightForTime(t time.Time) uint64 {
	t = t.UTC()
	daysSinceSunday := int(t.Weekday())
	sunday := t.AddDate(0, 0, -daysSinceSunday)
	sundayMidnight := time.Date(sunday.Year(), sunday.Month(), sunday.Day(), 0, 0, 0, 0, time.UTC)
	return uint64(sundayMidnight.Unix())
}
