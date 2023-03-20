package batcher

import (
	"bytes"
	"errors"
	"math"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	dtest "github.com/ethereum-optimism/optimism/op-node/rollup/derive/test"

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

// TestConfigValidation tests the validation of the [ChannelConfig] struct.
func TestConfigValidation(t *testing.T) {
	// Construct a valid config.
	validChannelConfig := defaultTestChannelConfig
	require.NoError(t, validChannelConfig.Check())

	// Set the config to have a zero max frame size.
	validChannelConfig.MaxFrameSize = 0
	require.ErrorIs(t, validChannelConfig.Check(), ErrZeroMaxFrameSize)

	// Set the config to have a max frame size less than 23.
	validChannelConfig.MaxFrameSize = 22
	require.ErrorIs(t, validChannelConfig.Check(), ErrSmallMaxFrameSize)

	// Reset the config and test the Timeout error.
	// NOTE: We should be fuzzing these values with the constraint that
	// 		 SubSafetyMargin > ChannelTimeout to ensure validation.
	validChannelConfig = defaultTestChannelConfig
	validChannelConfig.ChannelTimeout = 0
	validChannelConfig.SubSafetyMargin = 1
	require.ErrorIs(t, validChannelConfig.Check(), ErrInvalidChannelTimeout)
}

// addMiniBlock adds a minimal valid L2 block to the channel builder using the
// channelBuilder.AddBlock method.
func addMiniBlock(cb *channelBuilder) error {
	a := newMiniL2Block(0)
	_, err := cb.AddBlock(a)
	return err
}

// newMiniL2Block returns a minimal L2 block with a minimal valid L1InfoDeposit
// transaction as first transaction. Both blocks are minimal in the sense that
// most fields are left at defaults or are unset.
//
// If numTx > 0, that many empty DynamicFeeTxs will be added to the txs.
func newMiniL2Block(numTx int) *types.Block {
	return newMiniL2BlockWithNumberParent(numTx, new(big.Int), (common.Hash{}))
}

// newMiniL2Block returns a minimal L2 block with a minimal valid L1InfoDeposit
// transaction as first transaction. Both blocks are minimal in the sense that
// most fields are left at defaults or are unset. Block number and parent hash
// will be set to the given parameters number and parent.
//
// If numTx > 0, that many empty DynamicFeeTxs will be added to the txs.
func newMiniL2BlockWithNumberParent(numTx int, number *big.Int, parent common.Hash) *types.Block {
	l1Block := types.NewBlock(&types.Header{
		BaseFee:    big.NewInt(10),
		Difficulty: common.Big0,
		Number:     big.NewInt(100),
	}, nil, nil, nil, trie.NewStackTrie(nil))
	l1InfoTx, err := derive.L1InfoDeposit(0, l1Block, eth.SystemConfig{}, false)
	if err != nil {
		panic(err)
	}

	txs := make([]*types.Transaction, 0, 1+numTx)
	txs = append(txs, types.NewTx(l1InfoTx))
	for i := 0; i < numTx; i++ {
		txs = append(txs, types.NewTx(&types.DynamicFeeTx{}))
	}

	return types.NewBlock(&types.Header{
		Number:     number,
		ParentHash: parent,
	}, txs, nil, nil, trie.NewStackTrie(nil))
}

// addTooManyBlocks adds blocks to the channel until it hits an error,
// which is presumably ErrTooManyRLPBytes.
func addTooManyBlocks(cb *channelBuilder) error {
	for i := 0; i < 10_000; i++ {
		block := newMiniL2Block(100)
		_, err := cb.AddBlock(block)
		if err != nil {
			return err
		}
	}

	return nil
}

// FuzzDurationTimeoutZeroMaxChannelDuration ensures that when whenever the MaxChannelDuration
// is set to 0, the channel builder cannot have a duration timeout.
func FuzzDurationTimeoutZeroMaxChannelDuration(f *testing.F) {
	for i := range [10]int{} {
		f.Add(uint64(i))
	}
	f.Fuzz(func(t *testing.T, l1BlockNum uint64) {
		channelConfig := defaultTestChannelConfig
		channelConfig.MaxChannelDuration = 0
		cb, err := newChannelBuilder(channelConfig)
		require.NoError(t, err)
		cb.timeout = 0
		cb.updateDurationTimeout(l1BlockNum)
		require.False(t, cb.TimedOut(l1BlockNum))
	})
}

// FuzzDurationZero ensures that when whenever the MaxChannelDuration
// is not set to 0, the channel builder will always have a duration timeout
// as long as the channel builder's timeout is set to 0.
func FuzzDurationZero(f *testing.F) {
	for i := range [10]int{} {
		f.Add(uint64(i), uint64(i))
	}
	f.Fuzz(func(t *testing.T, l1BlockNum uint64, maxChannelDuration uint64) {
		if maxChannelDuration == 0 {
			t.Skip("Max channel duration cannot be 0")
		}

		// Create the channel builder
		channelConfig := defaultTestChannelConfig
		channelConfig.MaxChannelDuration = maxChannelDuration
		cb, err := newChannelBuilder(channelConfig)
		require.NoError(t, err)

		// Whenever the timeout is set to 0, the channel builder should have a duration timeout
		cb.timeout = 0
		cb.updateDurationTimeout(l1BlockNum)
		cb.checkTimeout(l1BlockNum + maxChannelDuration)
		require.ErrorIs(t, cb.FullErr(), ErrMaxDurationReached)
	})
}

// FuzzDurationTimeoutMaxChannelDuration ensures that when whenever the MaxChannelDuration
// is not set to 0, the channel builder will always have a duration timeout
// as long as the channel builder's timeout is greater than the target block number.
func FuzzDurationTimeoutMaxChannelDuration(f *testing.F) {
	// Set multiple seeds in case fuzzing isn't explicitly used
	for i := range [10]int{} {
		f.Add(uint64(i), uint64(i), uint64(i))
	}
	f.Fuzz(func(t *testing.T, l1BlockNum uint64, maxChannelDuration uint64, timeout uint64) {
		if maxChannelDuration == 0 {
			t.Skip("Max channel duration cannot be 0")
		}

		// Create the channel builder
		channelConfig := defaultTestChannelConfig
		channelConfig.MaxChannelDuration = maxChannelDuration
		cb, err := newChannelBuilder(channelConfig)
		require.NoError(t, err)

		// Whenever the timeout is greater than the l1BlockNum,
		// the channel builder should have a duration timeout
		cb.timeout = timeout
		cb.updateDurationTimeout(l1BlockNum)
		if timeout > l1BlockNum+maxChannelDuration {
			// Notice: we cannot call this outside of the if statement
			// because it would put the channel builder in an invalid state.
			// That is, where the channel builder has a value set for the timeout
			// with no timeoutReason. This subsequently causes a panic when
			// a nil timeoutReason is used as an error (eg when calling FullErr).
			cb.checkTimeout(l1BlockNum + maxChannelDuration)
			require.ErrorIs(t, cb.FullErr(), ErrMaxDurationReached)
		} else {
			require.NoError(t, cb.FullErr())
		}
	})
}

// FuzzChannelCloseTimeout ensures that the channel builder has a [ErrChannelTimeoutClose]
// as long as the timeout constraint is met and the builder's timeout is greater than
// the calculated timeout
func FuzzChannelCloseTimeout(f *testing.F) {
	// Set multiple seeds in case fuzzing isn't explicitly used
	for i := range [10]int{} {
		f.Add(uint64(i), uint64(i), uint64(i), uint64(i*5))
	}
	f.Fuzz(func(t *testing.T, l1BlockNum uint64, channelTimeout uint64, subSafetyMargin uint64, timeout uint64) {
		// Create the channel builder
		channelConfig := defaultTestChannelConfig
		channelConfig.ChannelTimeout = channelTimeout
		channelConfig.SubSafetyMargin = subSafetyMargin
		cb, err := newChannelBuilder(channelConfig)
		require.NoError(t, err)

		// Check the timeout
		cb.timeout = timeout
		cb.FramePublished(l1BlockNum)
		calculatedTimeout := l1BlockNum + channelTimeout - subSafetyMargin
		if timeout > calculatedTimeout && calculatedTimeout != 0 {
			cb.checkTimeout(calculatedTimeout)
			require.ErrorIs(t, cb.FullErr(), ErrChannelTimeoutClose)
		} else {
			require.NoError(t, cb.FullErr())
		}
	})
}

// FuzzChannelZeroCloseTimeout ensures that the channel builder has a [ErrChannelTimeoutClose]
// as long as the timeout constraint is met and the builder's timeout is set to zero.
func FuzzChannelZeroCloseTimeout(f *testing.F) {
	// Set multiple seeds in case fuzzing isn't explicitly used
	for i := range [10]int{} {
		f.Add(uint64(i), uint64(i), uint64(i))
	}
	f.Fuzz(func(t *testing.T, l1BlockNum uint64, channelTimeout uint64, subSafetyMargin uint64) {
		// Create the channel builder
		channelConfig := defaultTestChannelConfig
		channelConfig.ChannelTimeout = channelTimeout
		channelConfig.SubSafetyMargin = subSafetyMargin
		cb, err := newChannelBuilder(channelConfig)
		require.NoError(t, err)

		// Check the timeout
		cb.timeout = 0
		cb.FramePublished(l1BlockNum)
		calculatedTimeout := l1BlockNum + channelTimeout - subSafetyMargin
		cb.checkTimeout(calculatedTimeout)
		if cb.timeout != 0 {
			require.ErrorIs(t, cb.FullErr(), ErrChannelTimeoutClose)
		}
	})
}

// FuzzSeqWindowClose ensures that the channel builder has a [ErrSeqWindowClose]
// as long as the timeout constraint is met and the builder's timeout is greater than
// the calculated timeout
func FuzzSeqWindowClose(f *testing.F) {
	// Set multiple seeds in case fuzzing isn't explicitly used
	for i := range [10]int{} {
		f.Add(uint64(i), uint64(i), uint64(i), uint64(i*5))
	}
	f.Fuzz(func(t *testing.T, epochNum uint64, seqWindowSize uint64, subSafetyMargin uint64, timeout uint64) {
		// Create the channel builder
		channelConfig := defaultTestChannelConfig
		channelConfig.SeqWindowSize = seqWindowSize
		channelConfig.SubSafetyMargin = subSafetyMargin
		cb, err := newChannelBuilder(channelConfig)
		require.NoError(t, err)

		// Check the timeout
		cb.timeout = timeout
		cb.updateSwTimeout(&derive.BatchData{
			BatchV1: derive.BatchV1{
				EpochNum: rollup.Epoch(epochNum),
			},
		})
		calculatedTimeout := epochNum + seqWindowSize - subSafetyMargin
		if timeout > calculatedTimeout && calculatedTimeout != 0 {
			cb.checkTimeout(calculatedTimeout)
			require.ErrorIs(t, cb.FullErr(), ErrSeqWindowClose)
		} else {
			require.NoError(t, cb.FullErr())
		}
	})
}

// FuzzSeqWindowZeroTimeoutClose ensures that the channel builder has a [ErrSeqWindowClose]
// as long as the timeout constraint is met and the builder's timeout is set to zero.
func FuzzSeqWindowZeroTimeoutClose(f *testing.F) {
	// Set multiple seeds in case fuzzing isn't explicitly used
	for i := range [10]int{} {
		f.Add(uint64(i), uint64(i), uint64(i))
	}
	f.Fuzz(func(t *testing.T, epochNum uint64, seqWindowSize uint64, subSafetyMargin uint64) {
		// Create the channel builder
		channelConfig := defaultTestChannelConfig
		channelConfig.SeqWindowSize = seqWindowSize
		channelConfig.SubSafetyMargin = subSafetyMargin
		cb, err := newChannelBuilder(channelConfig)
		require.NoError(t, err)

		// Check the timeout
		cb.timeout = 0
		cb.updateSwTimeout(&derive.BatchData{
			BatchV1: derive.BatchV1{
				EpochNum: rollup.Epoch(epochNum),
			},
		})
		calculatedTimeout := epochNum + seqWindowSize - subSafetyMargin
		cb.checkTimeout(calculatedTimeout)
		if cb.timeout != 0 {
			require.ErrorIs(t, cb.FullErr(), ErrSeqWindowClose, "Sequence window close should be reached")
		}
	})
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
	frameData := frameData{
		id: frameID{
			chID:        co.ID(),
			frameNumber: fn,
		},
		data: expectedBytes,
	}
	cb.PushFrame(frameData)

	// There should only be 1 frame in the channel builder
	require.Equal(t, 1, cb.NumFrames())

	// We should be able to increment to the next frame
	constructedFrame := cb.NextFrame()
	require.Equal(t, expectedTx, constructedFrame.id)
	require.Equal(t, expectedBytes, constructedFrame.data)
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
		frame := frameData{
			id: frameID{
				chID:        co.ID(),
				frameNumber: fn,
			},
			data: buf.Bytes(),
		}
		cb.PushFrame(frame)
	})
}

// TestOutputFrames tests the OutputFrames function
func TestOutputFrames(t *testing.T) {
	channelConfig := defaultTestChannelConfig
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
	err = addMiniBlock(cb)
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

// TestMaxRLPBytesPerChannel tests the [channelBuilder.OutputFrames]
// function errors when the max RLP bytes per channel is reached.
func TestMaxRLPBytesPerChannel(t *testing.T) {
	t.Parallel()
	channelConfig := defaultTestChannelConfig
	channelConfig.MaxFrameSize = derive.MaxRLPBytesPerChannel * 2
	channelConfig.TargetFrameSize = derive.MaxRLPBytesPerChannel * 2
	channelConfig.ApproxComprRatio = 1

	// Construct the channel builder
	cb, err := newChannelBuilder(channelConfig)
	require.NoError(t, err)

	// Add a block that overflows the [ChannelOut]
	err = addTooManyBlocks(cb)
	require.ErrorIs(t, err, derive.ErrTooManyRLPBytes)
}

// TestOutputFramesMaxFrameIndex tests the [channelBuilder.OutputFrames]
// function errors when the max frame index is reached.
func TestOutputFramesMaxFrameIndex(t *testing.T) {
	channelConfig := defaultTestChannelConfig
	channelConfig.MaxFrameSize = 1
	channelConfig.TargetNumFrames = math.MaxInt
	channelConfig.TargetFrameSize = 1
	channelConfig.ApproxComprRatio = 0

	// Continuously add blocks until the max frame index is reached
	// This should cause the [channelBuilder.OutputFrames] function
	// to error
	cb, err := newChannelBuilder(channelConfig)
	require.NoError(t, err)
	require.False(t, cb.IsFull())
	require.Equal(t, 0, cb.NumFrames())
	for {
		lBlock := types.NewBlock(&types.Header{
			BaseFee:    common.Big0,
			Difficulty: common.Big0,
			Number:     common.Big0,
		}, nil, nil, nil, trie.NewStackTrie(nil))
		l1InfoTx, _ := derive.L1InfoDeposit(0, lBlock, eth.SystemConfig{}, false)
		txs := []*types.Transaction{types.NewTx(l1InfoTx)}
		a := types.NewBlock(&types.Header{
			Number: big.NewInt(0),
		}, txs, nil, nil, trie.NewStackTrie(nil))
		_, err = cb.AddBlock(a)
		if cb.IsFull() {
			fullErr := cb.FullErr()
			require.ErrorIs(t, fullErr, ErrMaxFrameIndex)
			break
		}
		require.NoError(t, err)
		_ = cb.OutputFrames()
		// Flushing so we can construct new frames
		_ = cb.co.Flush()
	}
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
	err = addMiniBlock(cb)
	require.NoError(t, err)

	// Check the fields reset in the AddBlock function
	require.Equal(t, 74, cb.co.InputBytes())
	require.Equal(t, 1, len(cb.blocks))
	require.Equal(t, 0, len(cb.frames))
	require.True(t, cb.IsFull())

	// Since the channel output is full, the next call to AddBlock
	// should return the channel out full error
	err = addMiniBlock(cb)
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
	err = addMiniBlock(cb)
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
	err = addMiniBlock(cb)
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

func TestChannelBuilder_InputBytes(t *testing.T) {
	require := require.New(t)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	cb, _ := defaultChannelBuilderSetup(t)

	require.Zero(cb.InputBytes())

	var l int
	for i := 0; i < 5; i++ {
		block := newMiniL2Block(rng.Intn(32))
		l += blockBatchRlpSize(t, block)

		_, err := cb.AddBlock(block)
		require.NoError(err)
		require.Equal(cb.InputBytes(), l)
	}
}

func TestChannelBuilder_OutputBytes(t *testing.T) {
	require := require.New(t)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	cfg := defaultTestChannelConfig
	cfg.TargetFrameSize = 1000
	cfg.MaxFrameSize = 1000
	cfg.TargetNumFrames = 16
	cfg.ApproxComprRatio = 1.0
	cb, err := newChannelBuilder(cfg)
	require.NoError(err, "newChannelBuilder")

	require.Zero(cb.OutputBytes())

	for {
		block, _ := dtest.RandomL2Block(rng, rng.Intn(32))
		_, err := cb.AddBlock(block)
		if errors.Is(err, ErrInputTargetReached) {
			break
		}
		require.NoError(err)
	}

	require.NoError(cb.OutputFrames())
	require.True(cb.IsFull())
	require.Greater(cb.NumFrames(), 1)

	var flen int
	for cb.HasFrame() {
		f := cb.NextFrame()
		flen += len(f.data)
	}

	require.Equal(cb.OutputBytes(), flen)
}

func defaultChannelBuilderSetup(t *testing.T) (*channelBuilder, ChannelConfig) {
	t.Helper()
	cfg := defaultTestChannelConfig
	cb, err := newChannelBuilder(cfg)
	require.NoError(t, err, "newChannelBuilder")
	return cb, cfg
}

func blockBatchRlpSize(t *testing.T, b *types.Block) int {
	t.Helper()
	batch, _, err := derive.BlockToBatch(b)
	require.NoError(t, err)
	var buf bytes.Buffer
	require.NoError(t, batch.EncodeRLP(&buf), "RLP-encoding batch")
	return buf.Len()
}
