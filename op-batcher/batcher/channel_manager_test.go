package batcher

import (
	"io"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-batcher/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	derivetest "github.com/ethereum-optimism/optimism/op-node/rollup/derive/test"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func channelManagerTestConfig(maxFrameSize uint64, batchType uint) ChannelConfig {
	cfg := ChannelConfig{
		MaxFrameSize:    maxFrameSize,
		TargetNumFrames: 1,
		BatchType:       batchType,
	}
	cfg.InitRatioCompressor(1, derive.Zlib)
	return cfg
}

func TestChannelManagerBatchType(t *testing.T) {
	tests := []struct {
		name string
		f    func(t *testing.T, batchType uint)
	}{
		{"ChannelManagerReturnsErrReorg", ChannelManagerReturnsErrReorg},
		{"ChannelManagerReturnsErrReorgWhenDrained", ChannelManagerReturnsErrReorgWhenDrained},
		{"ChannelManager_Clear", ChannelManager_Clear},
		{"ChannelManager_TxResend", ChannelManager_TxResend},
		{"ChannelManagerCloseBeforeFirstUse", ChannelManagerCloseBeforeFirstUse},
		{"ChannelManagerCloseNoPendingChannel", ChannelManagerCloseNoPendingChannel},
		{"ChannelManagerClosePendingChannel", ChannelManagerClosePendingChannel},
		{"ChannelManagerCloseAllTxsFailed", ChannelManagerCloseAllTxsFailed},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name+"_SingularBatch", func(t *testing.T) {
			test.f(t, derive.SingularBatchType)
		})
	}

	for _, test := range tests {
		test := test
		t.Run(test.name+"_SpanBatch", func(t *testing.T) {
			test.f(t, derive.SpanBatchType)
		})
	}
}

// ChannelManagerReturnsErrReorg ensures that the channel manager
// detects a reorg when it has cached L1 blocks.
func ChannelManagerReturnsErrReorg(t *testing.T, batchType uint) {
	log := testlog.Logger(t, log.LevelCrit)
	m := NewChannelManager(log, metrics.NoopMetrics, ChannelConfig{BatchType: batchType}, &rollup.Config{})
	m.Clear(eth.BlockID{})

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

// ChannelManagerReturnsErrReorgWhenDrained ensures that the channel manager
// detects a reorg even if it does not have any blocks inside it.
func ChannelManagerReturnsErrReorgWhenDrained(t *testing.T, batchType uint) {
	log := testlog.Logger(t, log.LevelCrit)
	cfg := channelManagerTestConfig(120_000, batchType)
	cfg.CompressorConfig.TargetOutputSize = 1 // full on first block
	m := NewChannelManager(log, metrics.NoopMetrics, cfg, &rollup.Config{})
	m.Clear(eth.BlockID{})

	a := newMiniL2Block(0)
	x := newMiniL2BlockWithNumberParent(0, big.NewInt(1), common.Hash{0xff})

	require.NoError(t, m.AddL2Block(a))

	_, err := m.TxData(eth.BlockID{})
	require.NoError(t, err)
	_, err = m.TxData(eth.BlockID{})
	require.ErrorIs(t, err, io.EOF)

	require.ErrorIs(t, m.AddL2Block(x), ErrReorg)
}

// ChannelManager_Clear tests clearing the channel manager.
func ChannelManager_Clear(t *testing.T, batchType uint) {
	require := require.New(t)

	// Create a channel manager
	log := testlog.Logger(t, log.LevelCrit)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	cfg := channelManagerTestConfig(derive.FrameV0OverHeadSize+1, batchType)
	// Need to set the channel timeout here so we don't clear pending
	// channels on confirmation. This would result in [TxConfirmed]
	// clearing confirmed transactions, and resetting the pendingChannels map
	cfg.ChannelTimeout = 10
	cfg.InitRatioCompressor(1, derive.Zlib)
	m := NewChannelManager(log, metrics.NoopMetrics, cfg, &defaultTestRollupConfig)

	// Channel Manager state should be empty by default
	require.Empty(m.blocks)
	require.Equal(eth.BlockID{}, m.l1OriginLastClosedChannel)
	require.Equal(common.Hash{}, m.tip)
	require.Nil(m.currentChannel)
	require.Empty(m.channelQueue)
	require.Empty(m.txChannels)
	// Set the last block
	m.Clear(eth.BlockID{})

	// Add a block to the channel manager
	a := derivetest.RandomL2BlockWithChainId(rng, 4, defaultTestRollupConfig.L2ChainID)
	newL1Tip := a.Hash()
	l1BlockID := eth.BlockID{
		Hash:   a.Hash(),
		Number: a.NumberU64(),
	}
	require.NoError(m.AddL2Block(a))

	// Make sure there is a channel
	require.NoError(m.ensureChannelWithSpace(l1BlockID))
	require.NotNil(m.currentChannel)
	require.Len(m.currentChannel.confirmedTransactions, 0)

	// Process the blocks
	// We should have a pending channel with 1 frame
	// and no more blocks since processBlocks consumes
	// the list
	require.NoError(m.processBlocks())
	require.NoError(m.currentChannel.channelBuilder.co.Flush())
	require.NoError(m.outputFrames())
	_, err := m.nextTxData(m.currentChannel)
	require.NoError(err)
	require.NotNil(m.l1OriginLastClosedChannel)
	require.Len(m.blocks, 0)
	require.Equal(newL1Tip, m.tip)
	require.Len(m.currentChannel.pendingTransactions, 1)

	// Add a new block so we can test clearing
	// the channel manager with a full state
	b := types.NewBlock(&types.Header{
		Number:     big.NewInt(1),
		ParentHash: a.Hash(),
	}, nil, nil, nil, nil)
	require.NoError(m.AddL2Block(b))
	require.Len(m.blocks, 1)
	require.Equal(b.Hash(), m.tip)

	safeL1Origin := eth.BlockID{
		Number: 123,
	}
	// Clear the channel manager
	m.Clear(safeL1Origin)

	// Check that the entire channel manager state cleared
	require.Empty(m.blocks)
	require.Equal(uint64(123), m.l1OriginLastClosedChannel.Number)
	require.Equal(common.Hash{}, m.tip)
	require.Nil(m.currentChannel)
	require.Empty(m.channelQueue)
	require.Empty(m.txChannels)
}

func ChannelManager_TxResend(t *testing.T, batchType uint) {
	require := require.New(t)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	log := testlog.Logger(t, log.LevelError)
	cfg := channelManagerTestConfig(120_000, batchType)
	cfg.CompressorConfig.TargetOutputSize = 1 // full on first block
	m := NewChannelManager(log, metrics.NoopMetrics, cfg, &defaultTestRollupConfig)
	m.Clear(eth.BlockID{})

	a := derivetest.RandomL2BlockWithChainId(rng, 4, defaultTestRollupConfig.L2ChainID)

	require.NoError(m.AddL2Block(a))

	txdata0, err := m.TxData(eth.BlockID{})
	require.NoError(err)
	txdata0bytes := txdata0.CallData()
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

	data1 := txdata1.CallData()
	require.Equal(data1, data0)
	fs, err := derive.ParseFrames(data1)
	require.NoError(err)
	require.Len(fs, 1)
}

// ChannelManagerCloseBeforeFirstUse ensures that the channel manager
// will not produce any frames if closed immediately.
func ChannelManagerCloseBeforeFirstUse(t *testing.T, batchType uint) {
	require := require.New(t)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	log := testlog.Logger(t, log.LevelCrit)
	m := NewChannelManager(log, metrics.NoopMetrics,
		channelManagerTestConfig(10000, batchType),
		&defaultTestRollupConfig,
	)
	m.Clear(eth.BlockID{})

	a := derivetest.RandomL2BlockWithChainId(rng, 4, defaultTestRollupConfig.L2ChainID)

	require.NoError(m.Close(), "Expected to close channel manager gracefully")

	err := m.AddL2Block(a)
	require.NoError(err, "Failed to add L2 block")

	_, err = m.TxData(eth.BlockID{})
	require.ErrorIs(err, io.EOF, "Expected closed channel manager to contain no tx data")
}

// ChannelManagerCloseNoPendingChannel ensures that the channel manager
// can gracefully close with no pending channels, and will not emit any new
// channel frames.
func ChannelManagerCloseNoPendingChannel(t *testing.T, batchType uint) {
	require := require.New(t)
	log := testlog.Logger(t, log.LevelCrit)
	cfg := channelManagerTestConfig(10000, batchType)
	cfg.CompressorConfig.TargetOutputSize = 1 // full on first block
	cfg.ChannelTimeout = 1000
	m := NewChannelManager(log, metrics.NoopMetrics, cfg, &defaultTestRollupConfig)
	m.Clear(eth.BlockID{})
	a := newMiniL2Block(0)
	b := newMiniL2BlockWithNumberParent(0, big.NewInt(1), a.Hash())

	err := m.AddL2Block(a)
	require.NoError(err, "Failed to add L2 block")

	txdata, err := m.TxData(eth.BlockID{})
	require.NoError(err, "Expected channel manager to return valid tx data")

	m.TxConfirmed(txdata.ID(), eth.BlockID{})

	_, err = m.TxData(eth.BlockID{})
	require.ErrorIs(err, io.EOF, "Expected channel manager to EOF")

	require.NoError(m.Close(), "Expected to close channel manager gracefully")

	err = m.AddL2Block(b)
	require.NoError(err, "Failed to add L2 block")

	_, err = m.TxData(eth.BlockID{})
	require.ErrorIs(err, io.EOF, "Expected closed channel manager to return no new tx data")
}

// ChannelManagerClosePendingChannel ensures that the channel manager
// can gracefully close with a pending channel, and will not produce any
// new channel frames after this point.
func ChannelManagerClosePendingChannel(t *testing.T, batchType uint) {
	require := require.New(t)
	// The number of batch txs depends on compression of the random data, hence the static test RNG seed.
	// Example of different RNG seed that creates less than 2 frames: 1698700588902821588
	rng := rand.New(rand.NewSource(123))
	log := testlog.Logger(t, log.LevelError)
	cfg := channelManagerTestConfig(10_000, batchType)
	cfg.ChannelTimeout = 1000
	m := NewChannelManager(log, metrics.NoopMetrics, cfg, &defaultTestRollupConfig)
	m.Clear(eth.BlockID{})

	numTx := 20 // Adjust number of txs to make 2 frames
	a := derivetest.RandomL2BlockWithChainId(rng, numTx, defaultTestRollupConfig.L2ChainID)

	err := m.AddL2Block(a)
	require.NoError(err, "Failed to add L2 block")

	txdata, err := m.TxData(eth.BlockID{})
	require.NoError(err, "Expected channel manager to produce valid tx data")
	log.Info("generated first tx data", "len", txdata.Len())

	m.TxConfirmed(txdata.ID(), eth.BlockID{})

	require.ErrorIs(m.Close(), ErrPendingAfterClose, "Expected channel manager to error on close because of pending tx data")

	txdata, err = m.TxData(eth.BlockID{})
	require.NoError(err, "Expected channel manager to produce tx data from remaining L2 block data")
	log.Info("generated more tx data", "len", txdata.Len())

	m.TxConfirmed(txdata.ID(), eth.BlockID{})

	_, err = m.TxData(eth.BlockID{})
	require.ErrorIs(err, io.EOF, "Expected channel manager to have no more tx data")

	_, err = m.TxData(eth.BlockID{})
	require.ErrorIs(err, io.EOF, "Expected closed channel manager to produce no more tx data")
}

// ChannelManager_Close_PartiallyPendingChannel ensures that the channel manager
// can gracefully close with a pending channel, where a block is still waiting
// inside the compressor to be flushed.
//
// This test runs only for singular batches on purpose.
// The SpanChannelOut writes full span batches to the compressor for
// every new block that's added, so NonCompressor cannot be used to
// set up a scenario where data is only partially flushed.
// Couldn't get the test to work even with modifying NonCompressor
// to flush half-way through writing to the compressor...
func TestChannelManager_Close_PartiallyPendingChannel(t *testing.T) {
	require := require.New(t)
	// The number of batch txs depends on compression of the random data, hence the static test RNG seed.
	// Example of different RNG seed that creates less than 2 frames: 1698700588902821588
	rng := rand.New(rand.NewSource(123))
	log := testlog.Logger(t, log.LevelError)
	cfg := ChannelConfig{
		MaxFrameSize:    2200,
		ChannelTimeout:  1000,
		TargetNumFrames: 100,
	}
	cfg.InitNoneCompressor()
	m := NewChannelManager(log, metrics.NoopMetrics, cfg, &defaultTestRollupConfig)
	m.Clear(eth.BlockID{})

	numTx := 3 // Adjust number of txs to make 2 frames
	a := derivetest.RandomL2BlockWithChainId(rng, numTx, defaultTestRollupConfig.L2ChainID)
	b := derivetest.RandomL2BlockWithChainId(rng, numTx, defaultTestRollupConfig.L2ChainID)
	bHeader := b.Header()
	bHeader.Number = new(big.Int).Add(a.Number(), big.NewInt(1))
	bHeader.ParentHash = a.Hash()
	b = b.WithSeal(bHeader)

	require.NoError(m.AddL2Block(a), "adding 1st L2 block")
	require.NoError(m.AddL2Block(b), "adding 2nd L2 block")

	// Inside TxData, the two blocks queued above are written to the compressor.
	// The NonCompressor will flush the first, but not the second block, when
	// adding the second block, setting up the test with a partially flushed
	// compressor.
	txdata, err := m.TxData(eth.BlockID{})
	require.NoError(err, "Expected channel manager to produce valid tx data")
	log.Info("generated first tx data", "len", txdata.Len())

	m.TxConfirmed(txdata.ID(), eth.BlockID{})

	// ensure no new ready data before closing
	_, err = m.TxData(eth.BlockID{})
	require.ErrorIs(err, io.EOF, "Expected unclosed channel manager to only return a single frame")

	require.ErrorIs(m.Close(), ErrPendingAfterClose, "Expected channel manager to error on close because of pending tx data")
	require.NotNil(m.currentChannel)
	require.ErrorIs(m.currentChannel.FullErr(), ErrTerminated, "Expected current channel to be terminated by Close")

	txdata, err = m.TxData(eth.BlockID{})
	require.NoError(err, "Expected channel manager to produce tx data from remaining L2 block data")
	log.Info("generated more tx data", "len", txdata.Len())

	m.TxConfirmed(txdata.ID(), eth.BlockID{})

	_, err = m.TxData(eth.BlockID{})
	require.ErrorIs(err, io.EOF, "Expected closed channel manager to produce no more tx data")
}

// ChannelManagerCloseAllTxsFailed ensures that the channel manager
// can gracefully close after producing transaction frames if none of these
// have successfully landed on chain.
func ChannelManagerCloseAllTxsFailed(t *testing.T, batchType uint) {
	require := require.New(t)
	rng := rand.New(rand.NewSource(1357))
	log := testlog.Logger(t, log.LevelCrit)
	cfg := channelManagerTestConfig(100, batchType)
	cfg.TargetNumFrames = 1000
	cfg.InitNoneCompressor()
	m := NewChannelManager(log, metrics.NoopMetrics, cfg, &defaultTestRollupConfig)
	m.Clear(eth.BlockID{})

	a := derivetest.RandomL2BlockWithChainId(rng, 1000, defaultTestRollupConfig.L2ChainID)

	err := m.AddL2Block(a)
	require.NoError(err, "Failed to add L2 block")

	drainTxData := func() (txdatas []txData) {
		for {
			txdata, err := m.TxData(eth.BlockID{})
			if err == io.EOF {
				return
			}
			require.NoError(err, "Expected channel manager to produce valid tx data")
			txdatas = append(txdatas, txdata)
		}
	}

	txdatas := drainTxData()
	require.NotEmpty(txdatas)

	for _, txdata := range txdatas {
		m.TxFailed(txdata.ID())
	}

	// Show that this data will continue to be emitted as long as the transaction
	// fails and the channel manager is not closed
	txdatas1 := drainTxData()
	require.NotEmpty(txdatas)
	require.ElementsMatch(txdatas, txdatas1, "expected same txdatas on re-attempt")

	for _, txdata := range txdatas1 {
		m.TxFailed(txdata.ID())
	}

	require.NoError(m.Close(), "Expected to close channel manager gracefully")

	_, err = m.TxData(eth.BlockID{})
	require.ErrorIs(err, io.EOF, "Expected closed channel manager to produce no more tx data")
}

func TestChannelManager_ChannelCreation(t *testing.T) {
	l := testlog.Logger(t, log.LevelCrit)
	const maxChannelDuration = 15
	cfg := channelManagerTestConfig(1000, derive.SpanBatchType)
	cfg.MaxChannelDuration = maxChannelDuration
	cfg.InitNoneCompressor()

	for _, tt := range []struct {
		name                   string
		safeL1Block            eth.BlockID
		expectedChannelTimeout uint64
	}{
		{
			name: "UseSafeHeadWhenNoLastL1Block",
			safeL1Block: eth.BlockID{
				Number: uint64(123),
			},
			// Safe head + maxChannelDuration
			expectedChannelTimeout: 123 + maxChannelDuration,
		},
		{
			name: "NoLastL1BlockNoSafeL1Block",
			safeL1Block: eth.BlockID{
				Number: 0,
			},
			// No timeout
			expectedChannelTimeout: 0 + maxChannelDuration,
		},
	} {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			m := NewChannelManager(l, metrics.NoopMetrics, cfg, &defaultTestRollupConfig)

			m.l1OriginLastClosedChannel = test.safeL1Block
			require.Nil(t, m.currentChannel)

			require.NoError(t, m.ensureChannelWithSpace(eth.BlockID{}))

			require.NotNil(t, m.currentChannel)
			require.Equal(t, test.expectedChannelTimeout, m.currentChannel.Timeout())
		})
	}
}
