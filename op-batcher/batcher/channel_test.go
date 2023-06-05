package batcher

import (
	"io"
	"testing"

	"github.com/ethereum-optimism/optimism/op-batcher/metrics"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// TestChannelTimeout tests that the channel manager
// correctly identifies when a pending channel is timed out.
func TestChannelTimeout(t *testing.T) {
	// Create a new channel manager with a ChannelTimeout
	log := testlog.Logger(t, log.LvlCrit)
	m := NewChannelManager(log, metrics.NoopMetrics, ChannelConfig{
		ChannelTimeout: 100,
	})

	// Pending channel is nil so is cannot be timed out
	require.Nil(t, m.currentChannel)

	// Set the pending channel
	require.NoError(t, m.ensureChannelWithSpace(eth.BlockID{}))
	channel := m.currentChannel
	require.NotNil(t, channel)

	// There are no confirmed transactions so
	// the pending channel cannot be timed out
	timeout := channel.isTimedOut()
	require.False(t, timeout)

	// Manually set a confirmed transactions
	// To avoid other methods clearing state
	channel.confirmedTransactions[frameID{frameNumber: 0}] = eth.BlockID{Number: 0}
	channel.confirmedTransactions[frameID{frameNumber: 1}] = eth.BlockID{Number: 99}

	// Since the ChannelTimeout is 100, the
	// pending channel should not be timed out
	timeout = channel.isTimedOut()
	require.False(t, timeout)

	// Add a confirmed transaction with a higher number
	// than the ChannelTimeout
	channel.confirmedTransactions[frameID{
		frameNumber: 2,
	}] = eth.BlockID{
		Number: 101,
	}

	// Now the pending channel should be timed out
	timeout = channel.isTimedOut()
	require.True(t, timeout)
}

// TestChannelNextTxData checks the nextTxData function.
func TestChannelNextTxData(t *testing.T) {
	log := testlog.Logger(t, log.LvlCrit)
	m := NewChannelManager(log, metrics.NoopMetrics, ChannelConfig{})

	// Nil pending channel should return EOF
	returnedTxData, err := m.nextTxData(nil)
	require.ErrorIs(t, err, io.EOF)
	require.Equal(t, txData{}, returnedTxData)

	// Set the pending channel
	// The nextTxData function should still return EOF
	// since the pending channel has no frames
	require.NoError(t, m.ensureChannelWithSpace(eth.BlockID{}))
	channel := m.currentChannel
	require.NotNil(t, channel)
	returnedTxData, err = m.nextTxData(channel)
	require.ErrorIs(t, err, io.EOF)
	require.Equal(t, txData{}, returnedTxData)

	// Manually push a frame into the pending channel
	channelID := channel.ID()
	frame := frameData{
		data: []byte{},
		id: frameID{
			chID:        channelID,
			frameNumber: uint16(0),
		},
	}
	channel.channelBuilder.PushFrame(frame)
	require.Equal(t, 1, channel.NumFrames())

	// Now the nextTxData function should return the frame
	returnedTxData, err = m.nextTxData(channel)
	expectedTxData := txData{frame}
	expectedChannelID := expectedTxData.ID()
	require.NoError(t, err)
	require.Equal(t, expectedTxData, returnedTxData)
	require.Equal(t, 0, channel.NumFrames())
	require.Equal(t, expectedTxData, channel.pendingTransactions[expectedChannelID])
}

// TestChannelTxConfirmed checks the [ChannelManager.TxConfirmed] function.
func TestChannelTxConfirmed(t *testing.T) {
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
	require.NoError(t, m.ensureChannelWithSpace(eth.BlockID{}))
	channelID := m.currentChannel.ID()
	frame := frameData{
		data: []byte{},
		id: frameID{
			chID:        channelID,
			frameNumber: uint16(0),
		},
	}
	m.currentChannel.channelBuilder.PushFrame(frame)
	require.Equal(t, 1, m.currentChannel.NumFrames())
	returnedTxData, err := m.nextTxData(m.currentChannel)
	expectedTxData := txData{frame}
	expectedChannelID := expectedTxData.ID()
	require.NoError(t, err)
	require.Equal(t, expectedTxData, returnedTxData)
	require.Equal(t, 0, m.currentChannel.NumFrames())
	require.Equal(t, expectedTxData, m.currentChannel.pendingTransactions[expectedChannelID])
	require.Len(t, m.currentChannel.pendingTransactions, 1)

	// An unknown pending transaction should not be marked as confirmed
	// and should not be removed from the pending transactions map
	actualChannelID := m.currentChannel.ID()
	unknownChannelID := derive.ChannelID([derive.ChannelIDLength]byte{0x69})
	require.NotEqual(t, actualChannelID, unknownChannelID)
	unknownTxID := frameID{chID: unknownChannelID, frameNumber: 0}
	blockID := eth.BlockID{Number: 0, Hash: common.Hash{0x69}}
	m.TxConfirmed(unknownTxID, blockID)
	require.Empty(t, m.currentChannel.confirmedTransactions)
	require.Len(t, m.currentChannel.pendingTransactions, 1)

	// Now let's mark the pending transaction as confirmed
	// and check that it is removed from the pending transactions map
	// and added to the confirmed transactions map
	m.TxConfirmed(expectedChannelID, blockID)
	require.Empty(t, m.currentChannel.pendingTransactions)
	require.Len(t, m.currentChannel.confirmedTransactions, 1)
	require.Equal(t, blockID, m.currentChannel.confirmedTransactions[expectedChannelID])
}

// TestChannelTxFailed checks the [ChannelManager.TxFailed] function.
func TestChannelTxFailed(t *testing.T) {
	// Create a channel manager
	log := testlog.Logger(t, log.LvlCrit)
	m := NewChannelManager(log, metrics.NoopMetrics, ChannelConfig{})

	// Let's add a valid pending transaction to the channel
	// manager so we can demonstrate correctness
	require.NoError(t, m.ensureChannelWithSpace(eth.BlockID{}))
	channelID := m.currentChannel.ID()
	frame := frameData{
		data: []byte{},
		id: frameID{
			chID:        channelID,
			frameNumber: uint16(0),
		},
	}
	m.currentChannel.channelBuilder.PushFrame(frame)
	require.Equal(t, 1, m.currentChannel.NumFrames())
	returnedTxData, err := m.nextTxData(m.currentChannel)
	expectedTxData := txData{frame}
	expectedChannelID := expectedTxData.ID()
	require.NoError(t, err)
	require.Equal(t, expectedTxData, returnedTxData)
	require.Equal(t, 0, m.currentChannel.NumFrames())
	require.Equal(t, expectedTxData, m.currentChannel.pendingTransactions[expectedChannelID])
	require.Len(t, m.currentChannel.pendingTransactions, 1)

	// Trying to mark an unknown pending transaction as failed
	// shouldn't modify state
	m.TxFailed(frameID{})
	require.Equal(t, 0, m.currentChannel.NumFrames())
	require.Equal(t, expectedTxData, m.currentChannel.pendingTransactions[expectedChannelID])

	// Now we still have a pending transaction
	// Let's mark it as failed
	m.TxFailed(expectedChannelID)
	require.Empty(t, m.currentChannel.pendingTransactions)
	// There should be a frame in the pending channel now
	require.Equal(t, 1, m.currentChannel.NumFrames())
}
