package batcher

import (
	"io"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	derivetest "github.com/ethereum-optimism/optimism/op-node/rollup/derive/test"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/stretchr/testify/require"
)

// TestChannelManagerReturnsErrReorg ensures that the channel manager
// detects a reorg when it has cached L1 blocks.
func TestChannelManagerReturnsErrReorg(t *testing.T) {
	log := testlog.Logger(t, log.LvlCrit)
	m := NewChannelManager(log, ChannelConfig{})

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

	err := m.AddL2Block(a)
	require.NoError(t, err)
	err = m.AddL2Block(b)
	require.NoError(t, err)
	err = m.AddL2Block(c)
	require.NoError(t, err)
	err = m.AddL2Block(x)
	require.ErrorIs(t, err, ErrReorg)
}

// TestChannelManagerReturnsErrReorgWhenDrained ensures that the channel manager
// detects a reorg even if it does not have any blocks inside it.
func TestChannelManagerReturnsErrReorgWhenDrained(t *testing.T) {
	log := testlog.Logger(t, log.LvlCrit)
	m := NewChannelManager(log, ChannelConfig{
		TargetFrameSize:  0,
		MaxFrameSize:     120_000,
		ApproxComprRatio: 1.0,
	})
	l1Block := types.NewBlock(&types.Header{
		BaseFee:    big.NewInt(10),
		Difficulty: common.Big0,
		Number:     big.NewInt(100),
	}, nil, nil, nil, trie.NewStackTrie(nil))
	l1InfoTx, err := derive.L1InfoDeposit(0, l1Block, eth.SystemConfig{}, false)
	require.NoError(t, err)
	txs := []*types.Transaction{types.NewTx(l1InfoTx)}

	a := types.NewBlock(&types.Header{
		Number: big.NewInt(0),
	}, txs, nil, nil, trie.NewStackTrie(nil))
	x := types.NewBlock(&types.Header{
		Number:     big.NewInt(1),
		ParentHash: common.Hash{0xff},
	}, txs, nil, nil, trie.NewStackTrie(nil))

	err = m.AddL2Block(a)
	require.NoError(t, err)

	_, err = m.TxData(eth.BlockID{})
	require.NoError(t, err)
	_, err = m.TxData(eth.BlockID{})
	require.ErrorIs(t, err, io.EOF)

	err = m.AddL2Block(x)
	require.ErrorIs(t, err, ErrReorg)
}

// TestChannelManagerNextTxData checks the nextTxData function.
func TestChannelManagerNextTxData(t *testing.T) {
	log := testlog.Logger(t, log.LvlCrit)
	m := NewChannelManager(log, ChannelConfig{})

	// Nil pending channel should return EOF
	returnedTxData, err := m.nextTxData()
	require.ErrorIs(t, err, io.EOF)
	require.Equal(t, txData{}, returnedTxData)

	// Set the pending channel
	// The nextTxData function should still return EOF
	// since the pending channel has no frames
	m.ensurePendingChannel(eth.BlockID{})
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

// TestChannelManagerTxConfirmed checks the [ChannelManager.TxConfirmed] function.
func TestChannelManagerTxConfirmed(t *testing.T) {
	// Create a channel manager
	log := testlog.Logger(t, log.LvlCrit)
	m := NewChannelManager(log, ChannelConfig{
		// Need to set the channel timeout here so we don't clear pending
		// channels on confirmation. This would result in [TxConfirmed]
		// clearing confirmed transactions, and reseting the pendingChannels map
		ChannelTimeout: 10,
	})

	// Let's add a valid pending transaction to the channel manager
	// So we can demonstrate that TxConfirmed's correctness
	m.ensurePendingChannel(eth.BlockID{})
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
	require.Equal(t, 1, len(m.pendingTransactions))

	// An unknown pending transaction should not be marked as confirmed
	// and should not be removed from the pending transactions map
	actualChannelID := m.pendingChannel.ID()
	unknownChannelID := derive.ChannelID([derive.ChannelIDLength]byte{0x69})
	require.NotEqual(t, actualChannelID, unknownChannelID)
	unknownTxID := frameID{chID: unknownChannelID, frameNumber: 0}
	blockID := eth.BlockID{Number: 0, Hash: common.Hash{0x69}}
	m.TxConfirmed(unknownTxID, blockID)
	require.Empty(t, m.confirmedTransactions)
	require.Equal(t, 1, len(m.pendingTransactions))

	// Now let's mark the pending transaction as confirmed
	// and check that it is removed from the pending transactions map
	// and added to the confirmed transactions map
	m.TxConfirmed(expectedChannelID, blockID)
	require.Empty(t, m.pendingTransactions)
	require.Equal(t, 1, len(m.confirmedTransactions))
	require.Equal(t, blockID, m.confirmedTransactions[expectedChannelID])
}

func TestChannelManager_TxResend(t *testing.T) {
	require := require.New(t)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	log := testlog.Logger(t, log.LvlError)
	m := NewChannelManager(log, ChannelConfig{
		TargetFrameSize:  0,
		MaxFrameSize:     120_000,
		ApproxComprRatio: 1.0,
	})

	a, _ := derivetest.RandomL2Block(rng, 4)

	err := m.AddL2Block(a)
	require.NoError(err)

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
