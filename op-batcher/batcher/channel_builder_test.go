package batcher

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"

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

func addNonsenseBlock(cb *channelBuilder) error {
	lBlock := types.NewBlock(&types.Header{
		BaseFee:    big.NewInt(10),
		Difficulty: common.Big0,
		Number:     big.NewInt(100),
	}, nil, nil, nil, trie.NewStackTrie(nil))
	l1InfoTx, err := derive.L1InfoDeposit(0, lBlock, eth.SystemConfig{}, false)
	if err != nil {
		return err
	}
	txs := []*types.Transaction{types.NewTx(l1InfoTx)}
	a := types.NewBlock(&types.Header{
		Number: big.NewInt(0),
	}, txs, nil, nil, trie.NewStackTrie(nil))
	err = cb.AddBlock(a)
	return err
}

// TestOutputFrames tests the OutputFrames function
func (testSuite *ChannelBuilderTestSuite) TestOutputFrames() {
	// Lower the max frame size so that we can test
	testSuite.channelConfig.MaxFrameSize = 2

	// Construct the channel builder
	cb, err := newChannelBuilder(testSuite.channelConfig)
	testSuite.NoError(err)

	testSuite.False(cb.IsFull())
	testSuite.Equal(0, cb.NumFrames())

	// Calling OutputFrames without having called [AddBlock]
	// should return `nil`.
	testSuite.Nil(cb.OutputFrames())

	// There should be no ready bytes yet
	readyBytes := cb.co.ReadyBytes()
	testSuite.Equal(0, readyBytes)

	// Let's add a block
	err = addNonsenseBlock(cb)
	testSuite.NoError(err)

	// Check how many ready bytes
	readyBytes = cb.co.ReadyBytes()
	testSuite.Equal(2, readyBytes)

	testSuite.Equal(0, cb.NumFrames())

	// The channel should not be full
	// but we want to output the frames for testing anyways
	isFull := cb.IsFull()
	testSuite.False(isFull)

	// Since we manually set the max frame size to 2,
	// we should be able to compress the two frames now
	err = cb.OutputFrames()
	testSuite.NoError(err)

	// There should be one frame in the channel builder now
	testSuite.Equal(1, cb.NumFrames())

	// There should no longer be any ready bytes
	readyBytes = cb.co.ReadyBytes()
	testSuite.Equal(0, readyBytes)
}

// TestBuilderAddBlock tests the AddBlock function
func (testSuite *ChannelBuilderTestSuite) TestBuilderAddBlock() {
	// Lower the max frame size so that we can batch
	testSuite.channelConfig.MaxFrameSize = 2

	// Configure the Input Threshold params so we observe a full channel
	// In reality, we only need the input bytes (74) below to be greater than
	// or equal to the input threshold (3 * 2) / 1 = 6
	testSuite.channelConfig.TargetFrameSize = 3
	testSuite.channelConfig.TargetNumFrames = 2
	testSuite.channelConfig.ApproxComprRatio = 1

	// Construct the channel builder
	cb, err := newChannelBuilder(testSuite.channelConfig)
	testSuite.NoError(err)

	// Add a nonsense block to the channel builder
	err = addNonsenseBlock(cb)
	testSuite.NoError(err)

	// Check the fields reset in the AddBlock function
	testSuite.Equal(74, cb.co.InputBytes())
	testSuite.Equal(1, len(cb.blocks))
	testSuite.Equal(0, len(cb.frames))
	testSuite.True(cb.IsFull())

	// Since the channel output is full, the next call to AddBlock
	// should return the channel out full error
	err = addNonsenseBlock(cb)
	testSuite.ErrorIs(err, ErrInputTargetReached)
}

// TestBuilderReset tests the Reset function
func (testSuite *ChannelBuilderTestSuite) TestBuilderReset() {
	// Lower the max frame size so that we can batch
	testSuite.channelConfig.MaxFrameSize = 2

	cb, err := newChannelBuilder(testSuite.channelConfig)
	testSuite.NoError(err)

	// Add a nonsense block to the channel builder
	err = addNonsenseBlock(cb)
	testSuite.NoError(err)

	// Check the fields reset in the Reset function
	testSuite.Equal(1, len(cb.blocks))
	testSuite.Equal(0, len(cb.frames))
	// Timeout should be updated in the AddBlock internal call to `updateSwTimeout`
	timeout := uint64(100) + cb.cfg.SeqWindowSize - cb.cfg.SubSafetyMargin
	testSuite.Equal(timeout, cb.timeout)
	testSuite.Nil(cb.fullErr)

	// Output frames so we can set the channel builder frames
	err = cb.OutputFrames()
	testSuite.NoError(err)

	// Add another block to increment the block count
	err = addNonsenseBlock(cb)
	testSuite.NoError(err)

	// Check the fields reset in the Reset function
	testSuite.Equal(2, len(cb.blocks))
	testSuite.Equal(1, len(cb.frames))
	testSuite.Equal(timeout, cb.timeout)
	testSuite.Nil(cb.fullErr)

	// Reset the channel builder
	err = cb.Reset()
	testSuite.NoError(err)

	// Check the fields reset in the Reset function
	testSuite.Equal(0, len(cb.blocks))
	testSuite.Equal(0, len(cb.frames))
	testSuite.Equal(uint64(0), cb.timeout)
	testSuite.Nil(cb.fullErr)
	testSuite.Equal(0, cb.co.InputBytes())
	testSuite.Equal(0, cb.co.ReadyBytes())
}
