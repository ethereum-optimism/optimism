package batcher

import (
	"bytes"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/stretchr/testify/require"
)

var defaultTestConfig = ChannelConfig{
	SeqWindowSize:      15,
	ChannelTimeout:     40,
	MaxChannelDuration: 1,
	SubSafetyMargin:    4,
	MaxFrameSize:       120000,
	TargetFrameSize:    100000,
	TargetNumFrames:    1,
	ApproxComprRatio:   0.4,
}

// TestBuilderNextFrame tests calling NextFrame on a ChannelBuilder with only one frame
func TestBuilderNextFrame(t *testing.T) {
	cb, err := NewChannelBuilder(defaultTestConfig)
	require.NoError(t, err)

	// Mock the internals of `channelBuilder.outputFrame`
	// to construct a single frame
	co := cb.co
	var buf bytes.Buffer
	fn, err := co.OutputFrame(&buf, defaultTestConfig.MaxFrameSize)
	require.NoError(t, err)

	// Push one frame into to the channel builder
	expectedTx := txID{chID: co.ID(), frameNumber: fn}
	expectedBytes := buf.Bytes()
	cb.PushFrame(expectedTx, expectedBytes)

	// There should only be 1 frame in the channel builder
	require.Equal(t, 1, cb.NumFrames())

	// We should be able to increment to the next frame
	constructedTx, constructedBytes := cb.NextFrame()
	require.Equal(t, expectedTx, constructedTx)
	require.Equal(t, expectedBytes, constructedBytes)
	require.Equal(t, 0, cb.NumFrames())

	// The next call should panic since the length of frames is 0
	defer func() { _ = recover() }()
	cb.NextFrame()

	// If we get here, `NextFrame` did not panic as expected
	t.Errorf("did not panic")
}

// TestBuilderInvalidFrameId tests that a panic is thrown when a frame is pushed with an invalid frame id
func TestBuilderWrongFramePanic(t *testing.T) {
	cb, err := NewChannelBuilder(defaultTestConfig)
	require.NoError(t, err)

	// Mock the internals of `channelBuilder.outputFrame`
	// to construct a single frame
	co, err := derive.NewChannelOut()
	require.NoError(t, err)
	var buf bytes.Buffer
	fn, err := co.OutputFrame(&buf, defaultTestConfig.MaxFrameSize)
	require.NoError(t, err)

	// The frame push should panic since we constructed a new channel out
	// so the channel out id won't match
	defer func() { _ = recover() }()

	// Push one frame into to the channel builder
	tx := txID{chID: co.ID(), frameNumber: fn}
	cb.PushFrame(tx, buf.Bytes())

	// If we get here, `PushFrame` did not panic as expected
	t.Errorf("did not panic")
}
