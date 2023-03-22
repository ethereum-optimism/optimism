package batcher_test

import (
	"math"
	"testing"

	"github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/stretchr/testify/require"
)

// TestInputThreshold tests the [ChannelConfig.InputThreshold]
// function using a table-driven testing approach.
func TestInputThreshold(t *testing.T) {
	type testInput struct {
		TargetFrameSize  uint64
		TargetNumFrames  int
		ApproxComprRatio float64
	}
	type test struct {
		input     testInput
		assertion func(uint64)
	}

	// Construct test cases that test the boundary conditions
	tests := []test{
		{
			input: testInput{
				TargetFrameSize:  1,
				TargetNumFrames:  1,
				ApproxComprRatio: 0.4,
			},
			assertion: func(output uint64) {
				require.Equal(t, uint64(2), output)
			},
		},
		{
			input: testInput{
				TargetFrameSize:  1,
				TargetNumFrames:  100000,
				ApproxComprRatio: 0.4,
			},
			assertion: func(output uint64) {
				require.Equal(t, uint64(250_000), output)
			},
		},
		{
			input: testInput{
				TargetFrameSize:  1,
				TargetNumFrames:  1,
				ApproxComprRatio: 1,
			},
			assertion: func(output uint64) {
				require.Equal(t, uint64(1), output)
			},
		},
		{
			input: testInput{
				TargetFrameSize:  1,
				TargetNumFrames:  1,
				ApproxComprRatio: 2,
			},
			assertion: func(output uint64) {
				require.Equal(t, uint64(0), output)
			},
		},
		{
			input: testInput{
				TargetFrameSize:  100000,
				TargetNumFrames:  1,
				ApproxComprRatio: 0.4,
			},
			assertion: func(output uint64) {
				require.Equal(t, uint64(250_000), output)
			},
		},
		{
			input: testInput{
				TargetFrameSize:  1,
				TargetNumFrames:  100000,
				ApproxComprRatio: 0.4,
			},
			assertion: func(output uint64) {
				require.Equal(t, uint64(250_000), output)
			},
		},
		{
			input: testInput{
				TargetFrameSize:  100000,
				TargetNumFrames:  100000,
				ApproxComprRatio: 0.4,
			},
			assertion: func(output uint64) {
				require.Equal(t, uint64(25_000_000_000), output)
			},
		},
		{
			input: testInput{
				TargetFrameSize:  1,
				TargetNumFrames:  1,
				ApproxComprRatio: 0.000001,
			},
			assertion: func(output uint64) {
				require.Equal(t, uint64(1_000_000), output)
			},
		},
		{
			input: testInput{
				TargetFrameSize:  0,
				TargetNumFrames:  0,
				ApproxComprRatio: 0,
			},
			assertion: func(output uint64) {
				// Need to allow for NaN depending on the machine architecture
				require.True(t, output == uint64(0) || output == uint64(math.NaN()))
			},
		},
	}

	// Validate each test case
	for _, tt := range tests {
		config := batcher.ChannelConfig{
			TargetFrameSize:  tt.input.TargetFrameSize,
			TargetNumFrames:  tt.input.TargetNumFrames,
			ApproxComprRatio: tt.input.ApproxComprRatio,
		}
		got := config.InputThreshold()
		tt.assertion(got)
	}
}
