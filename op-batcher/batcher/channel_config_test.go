package batcher_test

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/stretchr/testify/require"
)

// TestInputThreshold tests the [ChannelConfig.InputThreshold] function.
func TestInputThreshold(t *testing.T) {
	// Construct an empty channel config
	config := batcher.ChannelConfig{
		SeqWindowSize:      15,
		ChannelTimeout:     40,
		MaxChannelDuration: 1,
		SubSafetyMargin:    4,
		MaxFrameSize:       120000,
		TargetFrameSize:    100000,
		TargetNumFrames:    1,
		ApproxComprRatio:   0.4,
	}

	// The input threshold is calculated as: (targetNumFrames * targetFrameSize) / approxComprRatio
	// Here we see that 100,000 / 0.4 = 100,000 * 2.5 = 250,000
	inputThreshold := config.InputThreshold()
	require.Equal(t, uint64(250_000), inputThreshold)

	// Set the approximate compression ratio to 0
	// Logically, this represents infinite compression,
	// so there is no threshold on the size of the input.
	// In practice, this should never be set to 0.
	config.ApproxComprRatio = 0

	// The input threshold will overflow to the max uint64 value
	receivedThreshold := config.InputThreshold()
	max := config.TargetNumFrames * int(config.TargetFrameSize)
	require.True(t, receivedThreshold > uint64(max))
}
