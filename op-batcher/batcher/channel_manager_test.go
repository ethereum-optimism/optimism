package batcher_test

import (
	"io"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-batcher/batcher"
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
	m := batcher.NewChannelManager(log, batcher.ChannelConfig{})

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
	require.ErrorIs(t, err, batcher.ErrReorg)
}

// TestChannelManagerReturnsErrReorgWhenDrained ensures that the channel manager
// detects a reorg even if it does not have any blocks inside it.
func TestChannelManagerReturnsErrReorgWhenDrained(t *testing.T) {
	log := testlog.Logger(t, log.LvlCrit)
	m := batcher.NewChannelManager(log, batcher.ChannelConfig{
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
	require.ErrorIs(t, err, batcher.ErrReorg)
}

func TestChannelManager_TxResend(t *testing.T) {
	require := require.New(t)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	log := testlog.Logger(t, log.LvlError)
	m := batcher.NewChannelManager(log, batcher.ChannelConfig{
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
