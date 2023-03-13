package batcher

import (
	"bytes"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/stretchr/testify/suite"
)

// ChannelBuilderTestSuite encapsulates testing on the ChannelBuilder.
type ChannelBuilderTestSuite struct {
	suite.Suite
	channelConfig ChannelConfig
}

// SetupTest sets up the test suite.
func (testSuite *ChannelBuilderTestSuite) SetupTest() {
	testSuite.channelConfig = ChannelConfig{
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

// TestChannelBuilder runs the ChannelBuilderTestSuite.
func TestChannelBuilder(t *testing.T) {
	suite.Run(t, new(ChannelBuilderTestSuite))
}

// TestBuilderNextFrame tests calling NextFrame on a ChannelBuilder with only one frame
func (testSuite *ChannelBuilderTestSuite) TestBuilderNextFrame() {
	cb, err := newChannelBuilder(testSuite.channelConfig)
	testSuite.NoError(err)

	// Mock the internals of `channelBuilder.outputFrame`
	// to construct a single frame
	co := cb.co
	var buf bytes.Buffer
	fn, err := co.OutputFrame(&buf, testSuite.channelConfig.MaxFrameSize)
	testSuite.NoError(err)

	// Push one frame into to the channel builder
	expectedTx := txID{chID: co.ID(), frameNumber: fn}
	expectedBytes := buf.Bytes()
	cb.PushFrame(expectedTx, expectedBytes)

	// There should only be 1 frame in the channel builder
	testSuite.Equal(1, cb.NumFrames())

	// We should be able to increment to the next frame
	constructedTx, constructedBytes := cb.NextFrame()
	testSuite.Equal(expectedTx, constructedTx)
	testSuite.Equal(expectedBytes, constructedBytes)
	testSuite.Equal(0, cb.NumFrames())

	// The next call should panic since the length of frames is 0
	defer func() { _ = recover() }()
	cb.NextFrame()

	// If we get here, `NextFrame` did not panic as expected
	testSuite.T().Errorf("did not panic")
}

// TestBuilderInvalidFrameId tests that a panic is thrown when a frame is pushed with an invalid frame id
func (testSuite *ChannelBuilderTestSuite) TestBuilderWrongFramePanic() {
	cb, err := newChannelBuilder(testSuite.channelConfig)
	testSuite.NoError(err)

	// Mock the internals of `channelBuilder.outputFrame`
	// to construct a single frame
	co, err := derive.NewChannelOut()
	testSuite.NoError(err)
	var buf bytes.Buffer
	fn, err := co.OutputFrame(&buf, testSuite.channelConfig.MaxFrameSize)
	testSuite.NoError(err)

	// The frame push should panic since we constructed a new channel out
	// so the channel out id won't match
	defer func() { _ = recover() }()

	// Push one frame into to the channel builder
	tx := txID{chID: co.ID(), frameNumber: fn}
	cb.PushFrame(tx, buf.Bytes())

	// If we get here, `PushFrame` did not panic as expected
	testSuite.T().Errorf("did not panic")
}
