package batcher

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// defaultChannelConfig returns a valid, default [ChannelConfig] struct.
func defaultChannelConfig() ChannelConfig {
	return ChannelConfig{
		SeqWindowSize:      15,
		ChannelTimeout:     40,
		MaxChannelDuration: 1,
		SubSafetyMargin:    4,
		MaxFrameSize:       120000,
		TargetFrameSize:    100000,
		TargetNumFrames:    1,
		ApproxComprRatio:   0.4,
	}
}

// TestConfigValidation tests the validation of the [ChannelConfig] struct.
func TestConfigValidation(t *testing.T) {
	// Construct a valid config.
	validChannelConfig := defaultChannelConfig()
	require.NoError(t, validChannelConfig.Check())

	// Set the config to have a zero max frame size.
	validChannelConfig.MaxFrameSize = 0
	require.ErrorIs(t, validChannelConfig.Check(), ErrInvalidMaxFrameSize)

	// Reset the config and test the Timeout error.
	// NOTE: We should be fuzzing these values with the constraint that
	// 		 SubSafetyMargin > ChannelTimeout to ensure validation.
	validChannelConfig = defaultChannelConfig()
	validChannelConfig.ChannelTimeout = 0
	validChannelConfig.SubSafetyMargin = 1
	require.ErrorIs(t, validChannelConfig.Check(), ErrInvalidChannelTimeout)
}
