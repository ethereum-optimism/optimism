package batcher_test

import (
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
		input testInput
		want  uint64
	}

	// Construct test cases that test the boundary conditions
	tests := []test{
		{
			input: testInput{
				TargetFrameSize:  1,
				TargetNumFrames:  1,
				ApproxComprRatio: 0.4,
			},
			want: 2,
		},
		{
			input: testInput{
				TargetFrameSize:  1,
				TargetNumFrames:  1,
				ApproxComprRatio: 1,
			},
			want: 1,
		},
		{
			input: testInput{
				TargetFrameSize:  1,
				TargetNumFrames:  1,
				ApproxComprRatio: 2,
			},
			want: 0,
		},
		{
			input: testInput{
				TargetFrameSize:  100000,
				TargetNumFrames:  1,
				ApproxComprRatio: 0.4,
			},
			want: 250_000,
		},
		{
			input: testInput{
				TargetFrameSize:  1,
				TargetNumFrames:  100000,
				ApproxComprRatio: 0.4,
			},
			want: 250_000,
		},
		{
			input: testInput{
				TargetFrameSize:  100000,
				TargetNumFrames:  100000,
				ApproxComprRatio: 0.4,
			},
			want: 25_000_000_000,
		},
		// A compression ratio of 0 means there is no input threshold
		{
			input: testInput{
				TargetFrameSize:  100000,
				TargetNumFrames:  100000,
				ApproxComprRatio: 0,
			},
			want: uint64(0xffffffffffffffff),
		},
		{
			input: testInput{
				TargetFrameSize:  0,
				TargetNumFrames:  0,
				ApproxComprRatio: 0,
			},
			want: 0,
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
		require.Equal(t, tt.want, got)
	}
}
