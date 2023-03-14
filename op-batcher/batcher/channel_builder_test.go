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

	"github.com/stretchr/testify/require"
)

var defaultTestChannelConfig = ChannelConfig{
	SeqWindowSize:      15,
	ChannelTimeout:     40,
	MaxChannelDuration: 1,
	SubSafetyMargin:    4,
	MaxFrameSize:       120000,
	TargetFrameSize:    100000,
	TargetNumFrames:    1,
	ApproxComprRatio:   0.4,
}

// addNonsenseBlock is a helper function that adds a nonsense block
// to the channel builder using the [channelBuilder.AddBlock] method.
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

// TestBuilderNextFrame tests calling NextFrame on a ChannelBuilder with only one frame
func TestBuilderNextFrame(t *testing.T) {
	channelConfig := defaultTestChannelConfig

	// Create a new channel builder
	cb, err := newChannelBuilder(channelConfig)
	require.NoError(t, err)

	// Mock the internals of `channelBuilder.outputFrame`
	// to construct a single frame
	co := cb.co
	var buf bytes.Buffer
	fn, err := co.OutputFrame(&buf, channelConfig.MaxFrameSize)
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
	require.PanicsWithValue(t, "no next frame", func() { cb.NextFrame() })
}

// TestBuilderInvalidFrameId tests that a panic is thrown when a frame is pushed with an invalid frame id
func TestBuilderWrongFramePanic(t *testing.T) {
	channelConfig := defaultTestChannelConfig

	// Construct a channel builder
	cb, err := newChannelBuilder(channelConfig)
	require.NoError(t, err)

	// Mock the internals of `channelBuilder.outputFrame`
	// to construct a single frame
	co, err := derive.NewChannelOut()
	require.NoError(t, err)
	var buf bytes.Buffer
	fn, err := co.OutputFrame(&buf, channelConfig.MaxFrameSize)
	require.NoError(t, err)

	// The frame push should panic since we constructed a new channel out
	// so the channel out id won't match
	require.PanicsWithValue(t, "wrong channel", func() {
		tx := txID{chID: co.ID(), frameNumber: fn}
		cb.PushFrame(tx, buf.Bytes())
	})
}

// TestOutputFrames tests the OutputFrames function
func TestOutputFrames(t *testing.T) {
	channelConfig := defaultTestChannelConfig

	// Lower the max frame size so that we can test
	channelConfig.MaxFrameSize = 2

	// Construct the channel builder
	cb, err := newChannelBuilder(channelConfig)
	require.NoError(t, err)

	require.False(t, cb.IsFull())
	require.Equal(t, 0, cb.NumFrames())

	// Calling OutputFrames without having called [AddBlock]
	// should return no error
	require.NoError(t, cb.OutputFrames())

	// There should be no ready bytes yet
	readyBytes := cb.co.ReadyBytes()
	require.Equal(t, 0, readyBytes)

	// Let's add a block
	err = addNonsenseBlock(cb)
	require.NoError(t, err)

	// Check how many ready bytes
	readyBytes = cb.co.ReadyBytes()
	require.Equal(t, 2, readyBytes)

	require.Equal(t, 0, cb.NumFrames())

	// The channel should not be full
	// but we want to output the frames for testing anyways
	isFull := cb.IsFull()
	require.False(t, isFull)

	// Since we manually set the max frame size to 2,
	// we should be able to compress the two frames now
	err = cb.OutputFrames()
	require.NoError(t, err)

	// There should be one frame in the channel builder now
	require.Equal(t, 1, cb.NumFrames())

	// There should no longer be any ready bytes
	readyBytes = cb.co.ReadyBytes()
	require.Equal(t, 0, readyBytes)
}

// TestBuilderAddBlock tests the AddBlock function
func TestBuilderAddBlock(t *testing.T) {
	channelConfig := defaultTestChannelConfig

	// Lower the max frame size so that we can batch
	channelConfig.MaxFrameSize = 2

	// Configure the Input Threshold params so we observe a full channel
	// In reality, we only need the input bytes (74) below to be greater than
	// or equal to the input threshold (3 * 2) / 1 = 6
	channelConfig.TargetFrameSize = 3
	channelConfig.TargetNumFrames = 2
	channelConfig.ApproxComprRatio = 1

	// Construct the channel builder
	cb, err := newChannelBuilder(channelConfig)
	require.NoError(t, err)

	// Add a nonsense block to the channel builder
	err = addNonsenseBlock(cb)
	require.NoError(t, err)

	// Check the fields reset in the AddBlock function
	require.Equal(t, 74, cb.co.InputBytes())
	require.Equal(t, 1, len(cb.blocks))
	require.Equal(t, 0, len(cb.frames))
	require.True(t, cb.IsFull())

	// Since the channel output is full, the next call to AddBlock
	// should return the channel out full error
	err = addNonsenseBlock(cb)
	require.ErrorIs(t, err, ErrInputTargetReached)
}

// TestBuilderReset tests the Reset function
func TestBuilderReset(t *testing.T) {
	channelConfig := defaultTestChannelConfig

	// Lower the max frame size so that we can batch
	channelConfig.MaxFrameSize = 2

	cb, err := newChannelBuilder(channelConfig)
	require.NoError(t, err)

	// Add a nonsense block to the channel builder
	err = addNonsenseBlock(cb)
	require.NoError(t, err)

	// Check the fields reset in the Reset function
	require.Equal(t, 1, len(cb.blocks))
	require.Equal(t, 0, len(cb.frames))
	// Timeout should be updated in the AddBlock internal call to `updateSwTimeout`
	timeout := uint64(100) + cb.cfg.SeqWindowSize - cb.cfg.SubSafetyMargin
	require.Equal(t, timeout, cb.timeout)
	require.NoError(t, cb.fullErr)

	// Output frames so we can set the channel builder frames
	err = cb.OutputFrames()
	require.NoError(t, err)

	// Add another block to increment the block count
	err = addNonsenseBlock(cb)
	require.NoError(t, err)

	// Check the fields reset in the Reset function
	require.Equal(t, 2, len(cb.blocks))
	require.Equal(t, 1, len(cb.frames))
	require.Equal(t, timeout, cb.timeout)
	require.NoError(t, cb.fullErr)

	// Reset the channel builder
	err = cb.Reset()
	require.NoError(t, err)

	// Check the fields reset in the Reset function
	require.Equal(t, 0, len(cb.blocks))
	require.Equal(t, 0, len(cb.frames))
	require.Equal(t, uint64(0), cb.timeout)
	require.NoError(t, cb.fullErr)
	require.Equal(t, 0, cb.co.InputBytes())
	require.Equal(t, 0, cb.co.ReadyBytes())
}

// TestBuilderRegisterL1Block tests the RegisterL1Block function
func TestBuilderRegisterL1Block(t *testing.T) {
	channelConfig := defaultTestChannelConfig

	// Construct the channel builder
	cb, err := newChannelBuilder(channelConfig)
	require.NoError(t, err)

	// Assert params modified in RegisterL1Block
	require.Equal(t, uint64(1), channelConfig.MaxChannelDuration)
	require.Equal(t, uint64(0), cb.timeout)

	// Register a new L1 block
	cb.RegisterL1Block(uint64(100))

	// Assert params modified in RegisterL1Block
	require.Equal(t, uint64(1), channelConfig.MaxChannelDuration)
	require.Equal(t, uint64(101), cb.timeout)
}

// TestBuilderRegisterL1BlockZeroMaxChannelDuration tests the RegisterL1Block function
func TestBuilderRegisterL1BlockZeroMaxChannelDuration(t *testing.T) {
	channelConfig := defaultTestChannelConfig

	// Set the max channel duration to 0
	channelConfig.MaxChannelDuration = 0

	// Construct the channel builder
	cb, err := newChannelBuilder(channelConfig)
	require.NoError(t, err)

	// Assert params modified in RegisterL1Block
	require.Equal(t, uint64(0), channelConfig.MaxChannelDuration)
	require.Equal(t, uint64(0), cb.timeout)

	// Register a new L1 block
	cb.RegisterL1Block(uint64(100))

	// Since the max channel duration is set to 0,
	// the L1 block register should not update the timeout
	require.Equal(t, uint64(0), channelConfig.MaxChannelDuration)
	require.Equal(t, uint64(0), cb.timeout)
}

// TestFramePublished tests the FramePublished function
func TestFramePublished(t *testing.T) {
	channelConfig := defaultTestChannelConfig

	// Construct the channel builder
	cb, err := newChannelBuilder(channelConfig)
	require.NoError(t, err)

	// Let's say the block number is fed in as 100
	// and the channel timeout is 1000
	l1BlockNum := uint64(100)
	cb.cfg.ChannelTimeout = uint64(1000)
	cb.cfg.SubSafetyMargin = 100

	// Then the frame published will update the timeout
	cb.FramePublished(l1BlockNum)

	// Now the timeout will be 1000
	require.Equal(t, uint64(1000), cb.timeout)
}
