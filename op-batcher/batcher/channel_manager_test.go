package batcher

import (
	"io"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-batcher/compressor"
	"github.com/ethereum-optimism/optimism/op-batcher/metrics"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	derivetest "github.com/ethereum-optimism/optimism/op-node/rollup/derive/test"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// TestPendingChannelTimeout tests that the channel manager
// correctly identifies when a pending channel is timed out.
func TestPendingChannelTimeout(t *testing.T) {
	// Create a new channel manager with a ChannelTimeout
	log := testlog.Logger(t, log.LvlCrit)
	m := NewChannelManager(log, metrics.NoopMetrics, ChannelConfig{
		ChannelTimeout: 100,
	})

	// Pending channel is nil so is cannot be timed out
	timeout := m.pendingChannelIsTimedOut()
	require.False(t, timeout)

	// Set the pending channel
	require.NoError(t, m.ensurePendingChannel(eth.BlockID{}))

	// There are no confirmed transactions so
	// the pending channel cannot be timed out
	timeout = m.pendingChannelIsTimedOut()
	require.False(t, timeout)

	// Manually set a confirmed transactions
	// To avoid other methods clearing state
	m.confirmedTransactions[frameID{frameNumber: 0}] = eth.BlockID{Number: 0}
	m.confirmedTransactions[frameID{frameNumber: 1}] = eth.BlockID{Number: 99}

	// Since the ChannelTimeout is 100, the
	// pending channel should not be timed out
	timeout = m.pendingChannelIsTimedOut()
	require.False(t, timeout)

	// Add a confirmed transaction with a higher number
	// than the ChannelTimeout
	m.confirmedTransactions[frameID{
		frameNumber: 2,
	}] = eth.BlockID{
		Number: 101,
	}

	// Now the pending channel should be timed out
	timeout = m.pendingChannelIsTimedOut()
	require.True(t, timeout)
}

// TestChannelManagerReturnsErrReorg ensures that the channel manager
// detects a reorg when it has cached L1 blocks.
func TestChannelManagerReturnsErrReorg(t *testing.T) {
	log := testlog.Logger(t, log.LvlCrit)
	m := NewChannelManager(log, metrics.NoopMetrics, ChannelConfig{})

	a := types.NewBlock(&types.Header{
		Number: big.NewInt(0),
	}, nil, nil, nil, nil)
	b := types.NewBlock(&types.Header{
		Number:     big.NewInt(1),
		ParentHash: a.Hash(),
	}, nil, nil, nil, nil)
	c := types.NewBlock(&types.Header{
		Number:     big.NewInt(2),
		ParentHash: b.Hash(),
	}, nil, nil, nil, nil)
	x := types.NewBlock(&types.Header{
		Number:     big.NewInt(2),
		ParentHash: common.Hash{0xff},
	}, nil, nil, nil, nil)

	require.NoError(t, m.AddL2Block(a))
	require.NoError(t, m.AddL2Block(b))
	require.NoError(t, m.AddL2Block(c))
	require.ErrorIs(t, m.AddL2Block(x), ErrReorg)

	require.Equal(t, []*types.Block{a, b, c}, m.blocks)
}

// TestChannelManagerReturnsErrReorgWhenDrained ensures that the channel manager
// detects a reorg even if it does not have any blocks inside it.
func TestChannelManagerReturnsErrReorgWhenDrained(t *testing.T) {
	log := testlog.Logger(t, log.LvlCrit)
	m := NewChannelManager(log, metrics.NoopMetrics,
		ChannelConfig{
			MaxFrameSize: 120_000,
			CompressorConfig: compressor.Config{
				TargetFrameSize:  0,
				ApproxComprRatio: 1.0,
			},
		})

	a := newMiniL2Block(0)
	x := newMiniL2BlockWithNumberParent(0, big.NewInt(1), common.Hash{0xff})

	require.NoError(t, m.AddL2Block(a))

	_, err := m.TxData(eth.BlockID{})
	require.NoError(t, err)
	_, err = m.TxData(eth.BlockID{})
	require.ErrorIs(t, err, io.EOF)

	require.ErrorIs(t, m.AddL2Block(x), ErrReorg)
}

// TestChannelManagerNextTxData checks the nextTxData function.
func TestChannelManagerNextTxData(t *testing.T) {
	log := testlog.Logger(t, log.LvlCrit)
	m := NewChannelManager(log, metrics.NoopMetrics, ChannelConfig{})

	// Nil pending channel should return EOF
	returnedTxData, err := m.nextTxData()
	require.ErrorIs(t, err, io.EOF)
	require.Equal(t, txData{}, returnedTxData)

	// Set the pending channel
	// The nextTxData function should still return EOF
	// since the pending channel has no frames
	require.NoError(t, m.ensurePendingChannel(eth.BlockID{}))
	returnedTxData, err = m.nextTxData()
	require.ErrorIs(t, err, io.EOF)
	require.Equal(t, txData{}, returnedTxData)

	// Manually push a frame into the pending channel
	channelID := m.pendingChannel.ID()
	frame := frameData{
		data: []byte{},
		id: frameID{
			chID:        channelID,
			frameNumber: uint16(0),
		},
	}
	m.pendingChannel.PushFrame(frame)
	require.Equal(t, 1, m.pendingChannel.NumFrames())

	// Now the nextTxData function should return the frame
	returnedTxData, err = m.nextTxData()
	expectedTxData := txData{frame}
	expectedChannelID := expectedTxData.ID()
	require.NoError(t, err)
	require.Equal(t, expectedTxData, returnedTxData)
	require.Equal(t, 0, m.pendingChannel.NumFrames())
	require.Equal(t, expectedTxData, m.pendingTransactions[expectedChannelID])
}

// TestChannelManager_Clear tests clearing the channel manager.
func TestChannelManager_Clear(t *testing.T) {
	require := require.New(t)

	// Create a channel manager
	log := testlog.Logger(t, log.LvlCrit)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	m := NewChannelManager(log, metrics.NoopMetrics, ChannelConfig{
		// Need to set the channel timeout here so we don't clear pending
		// channels on confirmation. This would result in [TxConfirmed]
		// clearing confirmed transactions, and reseting the pendingChannels map
		ChannelTimeout: 10,
		// Have to set the max frame size here otherwise the channel builder would not
		// be able to output any frames
		MaxFrameSize: 24,
		CompressorConfig: compressor.Config{
			TargetFrameSize:  24,
			TargetNumFrames:  1,
			ApproxComprRatio: 1.0,
		},
	})

	// Channel Manager state should be empty by default
	require.Empty(m.blocks)
	require.Equal(common.Hash{}, m.tip)
	require.Nil(m.pendingChannel)
	require.Empty(m.pendingTransactions)
	require.Empty(m.confirmedTransactions)

	// Add a block to the channel manager
	a, _ := derivetest.RandomL2Block(rng, 4)
	newL1Tip := a.Hash()
	l1BlockID := eth.BlockID{
		Hash:   a.Hash(),
		Number: a.NumberU64(),
	}
	require.NoError(m.AddL2Block(a))

	// Make sure there is a channel builder
	require.NoError(m.ensurePendingChannel(l1BlockID))
	require.NotNil(m.pendingChannel)
	require.Len(m.confirmedTransactions, 0)

	// Process the blocks
	// We should have a pending channel with 1 frame
	// and no more blocks since processBlocks consumes
	// the list
	require.NoError(m.processBlocks())
	require.NoError(m.pendingChannel.co.Flush())
	require.NoError(m.pendingChannel.OutputFrames())
	_, err := m.nextTxData()
	require.NoError(err)
	require.Len(m.blocks, 0)
	require.Equal(newL1Tip, m.tip)
	require.Len(m.pendingTransactions, 1)

	// Add a new block so we can test clearing
	// the channel manager with a full state
	b := types.NewBlock(&types.Header{
		Number:     big.NewInt(1),
		ParentHash: a.Hash(),
	}, nil, nil, nil, nil)
	require.NoError(m.AddL2Block(b))
	require.Len(m.blocks, 1)
	require.Equal(b.Hash(), m.tip)

	// Clear the channel manager
	m.Clear()

	// Check that the entire channel manager state cleared
	require.Empty(m.blocks)
	require.Equal(common.Hash{}, m.tip)
	require.Nil(m.pendingChannel)
	require.Empty(m.pendingTransactions)
	require.Empty(m.confirmedTransactions)
}

// TestChannelManagerTxConfirmed checks the [ChannelManager.TxConfirmed] function.
func TestChannelManagerTxConfirmed(t *testing.T) {
	// Create a channel manager
	log := testlog.Logger(t, log.LvlCrit)
	m := NewChannelManager(log, metrics.NoopMetrics, ChannelConfig{
		// Need to set the channel timeout here so we don't clear pending
		// channels on confirmation. This would result in [TxConfirmed]
		// clearing confirmed transactions, and reseting the pendingChannels map
		ChannelTimeout: 10,
	})

	// Let's add a valid pending transaction to the channel manager
	// So we can demonstrate that TxConfirmed's correctness
	require.NoError(t, m.ensurePendingChannel(eth.BlockID{}))
	channelID := m.pendingChannel.ID()
	frame := frameData{
		data: []byte{},
		id: frameID{
			chID:        channelID,
			frameNumber: uint16(0),
		},
	}
	m.pendingChannel.PushFrame(frame)
	require.Equal(t, 1, m.pendingChannel.NumFrames())
	returnedTxData, err := m.nextTxData()
	expectedTxData := txData{frame}
	expectedChannelID := expectedTxData.ID()
	require.NoError(t, err)
	require.Equal(t, expectedTxData, returnedTxData)
	require.Equal(t, 0, m.pendingChannel.NumFrames())
	require.Equal(t, expectedTxData, m.pendingTransactions[expectedChannelID])
	require.Len(t, m.pendingTransactions, 1)

	// An unknown pending transaction should not be marked as confirmed
	// and should not be removed from the pending transactions map
	actualChannelID := m.pendingChannel.ID()
	unknownChannelID := derive.ChannelID([derive.ChannelIDLength]byte{0x69})
	require.NotEqual(t, actualChannelID, unknownChannelID)
	unknownTxID := frameID{chID: unknownChannelID, frameNumber: 0}
	blockID := eth.BlockID{Number: 0, Hash: common.Hash{0x69}}
	m.TxConfirmed(unknownTxID, blockID)
	require.Empty(t, m.confirmedTransactions)
	require.Len(t, m.pendingTransactions, 1)

	// Now let's mark the pending transaction as confirmed
	// and check that it is removed from the pending transactions map
	// and added to the confirmed transactions map
	m.TxConfirmed(expectedChannelID, blockID)
	require.Empty(t, m.pendingTransactions)
	require.Len(t, m.confirmedTransactions, 1)
	require.Equal(t, blockID, m.confirmedTransactions[expectedChannelID])
}

// TestChannelManagerTxFailed checks the [ChannelManager.TxFailed] function.
func TestChannelManagerTxFailed(t *testing.T) {
	// Create a channel manager
	log := testlog.Logger(t, log.LvlCrit)
	m := NewChannelManager(log, metrics.NoopMetrics, ChannelConfig{})

	// Let's add a valid pending transaction to the channel
	// manager so we can demonstrate correctness
	require.NoError(t, m.ensurePendingChannel(eth.BlockID{}))
	channelID := m.pendingChannel.ID()
	frame := frameData{
		data: []byte{},
		id: frameID{
			chID:        channelID,
			frameNumber: uint16(0),
		},
	}
	m.pendingChannel.PushFrame(frame)
	require.Equal(t, 1, m.pendingChannel.NumFrames())
	returnedTxData, err := m.nextTxData()
	expectedTxData := txData{frame}
	expectedChannelID := expectedTxData.ID()
	require.NoError(t, err)
	require.Equal(t, expectedTxData, returnedTxData)
	require.Equal(t, 0, m.pendingChannel.NumFrames())
	require.Equal(t, expectedTxData, m.pendingTransactions[expectedChannelID])
	require.Len(t, m.pendingTransactions, 1)

	// Trying to mark an unknown pending transaction as failed
	// shouldn't modify state
	m.TxFailed(frameID{})
	require.Equal(t, 0, m.pendingChannel.NumFrames())
	require.Equal(t, expectedTxData, m.pendingTransactions[expectedChannelID])

	// Now we still have a pending transaction
	// Let's mark it as failed
	m.TxFailed(expectedChannelID)
	require.Empty(t, m.pendingTransactions)
	// There should be a frame in the pending channel now
	require.Equal(t, 1, m.pendingChannel.NumFrames())
}

func TestChannelManager_TxResend(t *testing.T) {
	require := require.New(t)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	log := testlog.Logger(t, log.LvlError)
	m := NewChannelManager(log, metrics.NoopMetrics,
		ChannelConfig{
			MaxFrameSize: 120_000,
			CompressorConfig: compressor.Config{
				TargetFrameSize:  0,
				ApproxComprRatio: 1.0,
			},
		})

	a, _ := derivetest.RandomL2Block(rng, 4)

	require.NoError(m.AddL2Block(a))

	txdata0, err := m.TxData(eth.BlockID{})
	require.NoError(err)
	txdata0bytes := txdata0.Bytes()
	data0 := make([]byte, len(txdata0bytes))
	// make sure we have a clone for later comparison
	copy(data0, txdata0bytes)

	// ensure channel is drained
	_, err = m.TxData(eth.BlockID{})
	require.ErrorIs(err, io.EOF)

	// requeue frame
	m.TxFailed(txdata0.ID())

	txdata1, err := m.TxData(eth.BlockID{})
	require.NoError(err)

	data1 := txdata1.Bytes()
	require.Equal(data1, data0)
	fs, err := derive.ParseFrames(data1)
	require.NoError(err)
	require.Len(fs, 1)
}

// TestChannelManagerCloseBeforeFirstUse ensures that the channel manager
// will not produce any frames if closed immediately.
func TestChannelManagerCloseBeforeFirstUse(t *testing.T) {
	require := require.New(t)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	log := testlog.Logger(t, log.LvlCrit)
	m := NewChannelManager(log, metrics.NoopMetrics,
		ChannelConfig{
			MaxFrameSize:   100,
			ChannelTimeout: 1000,
			CompressorConfig: compressor.Config{
				TargetFrameSize:  0,
				ApproxComprRatio: 1.0,
			},
		})

	a, _ := derivetest.RandomL2Block(rng, 4)

	m.Close()

	err := m.AddL2Block(a)
	require.NoError(err, "Failed to add L2 block")

	_, err = m.TxData(eth.BlockID{})
	require.ErrorIs(err, io.EOF, "Expected closed channel manager to contain no tx data")
}

// TestChannelManagerCloseNoPendingChannel ensures that the channel manager
// can gracefully close with no pending channels, and will not emit any new
// channel frames.
func TestChannelManagerCloseNoPendingChannel(t *testing.T) {
	require := require.New(t)
	log := testlog.Logger(t, log.LvlCrit)
	m := NewChannelManager(log, metrics.NoopMetrics,
		ChannelConfig{
			MaxFrameSize:   100,
			ChannelTimeout: 1000,
			CompressorConfig: compressor.Config{
				TargetFrameSize:  1,
				TargetNumFrames:  1,
				ApproxComprRatio: 1.0,
			},
		})
	a := newMiniL2Block(0)
	b := newMiniL2BlockWithNumberParent(0, big.NewInt(1), a.Hash())

	err := m.AddL2Block(a)
	require.NoError(err, "Failed to add L2 block")

	txdata, err := m.TxData(eth.BlockID{})
	require.NoError(err, "Expected channel manager to return valid tx data")

	m.TxConfirmed(txdata.ID(), eth.BlockID{})

	_, err = m.TxData(eth.BlockID{})
	require.ErrorIs(err, io.EOF, "Expected channel manager to EOF")

	m.Close()

	err = m.AddL2Block(b)
	require.NoError(err, "Failed to add L2 block")

	_, err = m.TxData(eth.BlockID{})
	require.ErrorIs(err, io.EOF, "Expected closed channel manager to return no new tx data")
}

// TestChannelManagerCloseNoPendingChannel ensures that the channel manager
// can gracefully close with a pending channel, and will not produce any
// new channel frames after this point.
func TestChannelManagerClosePendingChannel(t *testing.T) {
	require := require.New(t)
	log := testlog.Logger(t, log.LvlCrit)
	m := NewChannelManager(log, metrics.NoopMetrics,
		ChannelConfig{
			MaxFrameSize:   1000,
			ChannelTimeout: 1000,
			CompressorConfig: compressor.Config{
				TargetNumFrames:  100,
				TargetFrameSize:  1000,
				ApproxComprRatio: 1.0,
			},
		})

	a := newMiniL2Block(50_000)
	b := newMiniL2BlockWithNumberParent(10, big.NewInt(1), a.Hash())

	err := m.AddL2Block(a)
	require.NoError(err, "Failed to add L2 block")

	txdata, err := m.TxData(eth.BlockID{})
	require.NoError(err, "Expected channel manager to produce valid tx data")

	m.TxConfirmed(txdata.ID(), eth.BlockID{})

	m.Close()

	txdata, err = m.TxData(eth.BlockID{})
	require.NoError(err, "Expected channel manager to produce tx data from remaining L2 block data")

	m.TxConfirmed(txdata.ID(), eth.BlockID{})

	_, err = m.TxData(eth.BlockID{})
	require.ErrorIs(err, io.EOF, "Expected channel manager to have no more tx data")

	err = m.AddL2Block(b)
	require.NoError(err, "Failed to add L2 block")

	_, err = m.TxData(eth.BlockID{})
	require.ErrorIs(err, io.EOF, "Expected closed channel manager to produce no more tx data")
}

// TestChannelManagerCloseAllTxsFailed ensures that the channel manager
// can gracefully close after producing transaction frames if none of these
// have successfully landed on chain.
func TestChannelManagerCloseAllTxsFailed(t *testing.T) {
	require := require.New(t)
	log := testlog.Logger(t, log.LvlCrit)
	m := NewChannelManager(log, metrics.NoopMetrics,
		ChannelConfig{
			MaxFrameSize:   1000,
			ChannelTimeout: 1000,
			CompressorConfig: compressor.Config{
				TargetNumFrames:  100,
				TargetFrameSize:  1000,
				ApproxComprRatio: 1.0,
			},
		})

	a := newMiniL2Block(50_000)

	err := m.AddL2Block(a)
	require.NoError(err, "Failed to add L2 block")

	txdata, err := m.TxData(eth.BlockID{})
	require.NoError(err, "Expected channel manager to produce valid tx data")

	m.TxFailed(txdata.ID())

	// Show that this data will continue to be emitted as long as the transaction
	// fails and the channel manager is not closed
	txdata, err = m.TxData(eth.BlockID{})
	require.NoError(err, "Expected channel manager to re-attempt the failed transaction")

	m.TxFailed(txdata.ID())

	m.Close()

	_, err = m.TxData(eth.BlockID{})
	require.ErrorIs(err, io.EOF, "Expected closed channel manager to produce no more tx data")
}
