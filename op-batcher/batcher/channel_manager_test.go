package batcher

import (
	"errors"
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
	"github.com/ethereum-optimism/optimism/op-service/queue"
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
	}, nil, nil, nil)
	b := types.NewBlock(&types.Header{
		Number:     big.NewInt(1),
		ParentHash: a.Hash(),
	}, nil, nil, nil)
	c := types.NewBlock(&types.Header{
		Number:     big.NewInt(2),
		ParentHash: b.Hash(),
	}, nil, nil, nil)
	x := types.NewBlock(&types.Header{
		Number:     big.NewInt(2),
		ParentHash: common.Hash{0xff},
	}, nil, nil, nil)

	require.NoError(t, m.AddL2Block(a))
	require.NoError(t, m.AddL2Block(b))
	require.NoError(t, m.AddL2Block(c))
	require.ErrorIs(t, m.AddL2Block(x), ErrReorg)

	require.Equal(t, queue.Queue[*types.Block]{a, b, c}, m.blocks)
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
	m := NewChannelManager(log, metrics.NoopMetrics, cfg, defaultTestRollupConfig)

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

	require.NoError(m.processBlocks())
	require.NoError(m.currentChannel.channelBuilder.co.Flush())
	require.NoError(m.outputFrames())
	_, err := m.nextTxData(m.currentChannel)
	require.NoError(err)
	require.NotNil(m.l1OriginLastClosedChannel)
	require.Equal(m.blockCursor, len(m.blocks))
	require.Equal(newL1Tip, m.tip)
	require.Len(m.currentChannel.pendingTransactions, 1)

	// Add a new block so we can test clearing
	// the channel manager with a full state
	b := types.NewBlock(&types.Header{
		Number:     big.NewInt(1),
		ParentHash: a.Hash(),
	}, nil, nil, nil)
	require.NoError(m.AddL2Block(b))
	require.Equal(m.blockCursor, len(m.blocks)-1)
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
	m := NewChannelManager(log, metrics.NoopMetrics, cfg, defaultTestRollupConfig)
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
			m := NewChannelManager(l, metrics.NoopMetrics, cfg, defaultTestRollupConfig)

			m.l1OriginLastClosedChannel = test.safeL1Block
			require.Nil(t, m.currentChannel)

			require.NoError(t, m.ensureChannelWithSpace(eth.BlockID{}))

			require.NotNil(t, m.currentChannel)
			require.Equal(t, test.expectedChannelTimeout, m.currentChannel.Timeout())
		})
	}
}

// FakeDynamicEthChannelConfig is a ChannelConfigProvider which always returns
// either a blob- or calldata-based config depending on its internal chooseBlob
// switch.
type FakeDynamicEthChannelConfig struct {
	DynamicEthChannelConfig
	chooseBlobs bool
	assessments int
}

func (f *FakeDynamicEthChannelConfig) ChannelConfig() ChannelConfig {
	f.assessments++
	if f.chooseBlobs {
		return f.blobConfig
	}
	return f.calldataConfig
}

func newFakeDynamicEthChannelConfig(lgr log.Logger,
	reqTimeout time.Duration) *FakeDynamicEthChannelConfig {

	calldataCfg := ChannelConfig{
		MaxFrameSize:    120_000 - 1,
		TargetNumFrames: 1,
	}
	blobCfg := ChannelConfig{
		MaxFrameSize:    eth.MaxBlobDataSize - 1,
		TargetNumFrames: 3, // gets closest to amortized fixed tx costs
		UseBlobs:        true,
	}
	calldataCfg.InitNoneCompressor()
	blobCfg.InitNoneCompressor()

	return &FakeDynamicEthChannelConfig{
		chooseBlobs: false,
		DynamicEthChannelConfig: *NewDynamicEthChannelConfig(
			lgr,
			reqTimeout,
			&mockGasPricer{},
			blobCfg,
			calldataCfg),
	}
}

// TestChannelManager_TxData seeds the channel manager with blocks and triggers the
// blocks->channels pipeline multiple times. Values are chosen such that a channel
// is created under one set of market conditions, and then submitted under a different
// set of market conditions. The test asserts that the DA type is changed at channel
// submission time.
func TestChannelManager_TxData(t *testing.T) {

	type TestCase struct {
		name                            string
		chooseBlobsWhenChannelCreated   bool
		chooseBlobsWhenChannelSubmitted bool

		// * One when the channelManager was created
		// * One when the channel is about to be submitted
		// * Potentially one more when the replacement channel
		//   is not immediately ready to be submitted, but later
		//   becomes ready after more data is added.
		//   This only happens when going from calldata->blobs because
		//   the channel is not immediately ready to send until more data
		//   is added due to blob channels having greater capacity.
		numExpectedAssessments int
	}

	tt := []TestCase{
		{"blobs->blobs", true, true, 2},
		{"calldata->calldata", false, false, 2},
		{"blobs->calldata", true, false, 2},
		{"calldata->blobs", false, true, 3},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			l := testlog.Logger(t, log.LevelCrit)

			cfg := newFakeDynamicEthChannelConfig(l, 1000)

			cfg.chooseBlobs = tc.chooseBlobsWhenChannelCreated
			m := NewChannelManager(l, metrics.NoopMetrics, cfg, defaultTestRollupConfig)
			require.Equal(t, tc.chooseBlobsWhenChannelCreated, m.defaultCfg.UseBlobs)

			// Seed channel manager with a block
			rng := rand.New(rand.NewSource(99))
			blockA := derivetest.RandomL2BlockWithChainId(rng, 200, defaultTestRollupConfig.L2ChainID)
			m.blocks = []*types.Block{blockA}

			// Call TxData a first time to trigger blocks->channels pipeline
			_, err := m.TxData(eth.BlockID{})
			require.ErrorIs(t, err, io.EOF)

			// The test requires us to have something in the channel queue
			// at this point, but not yet ready to send and not full
			require.NotEmpty(t, m.channelQueue)
			require.False(t, m.channelQueue[0].IsFull())

			// Simulate updated market conditions
			// by possibly flipping the state of the
			// fake channel provider
			l.Info("updating market conditions", "chooseBlobs", tc.chooseBlobsWhenChannelSubmitted)
			cfg.chooseBlobs = tc.chooseBlobsWhenChannelSubmitted

			// Add a block and call TxData until
			// we get some data to submit
			var data txData
			for {
				m.blocks = append(m.blocks, blockA)
				data, err = m.TxData(eth.BlockID{})
				if err == nil && data.Len() > 0 {
					break
				}
				if !errors.Is(err, io.EOF) {
					require.NoError(t, err)
				}
			}

			require.Equal(t, tc.numExpectedAssessments, cfg.assessments)
			require.Equal(t, tc.chooseBlobsWhenChannelSubmitted, data.asBlob)
			require.Equal(t, tc.chooseBlobsWhenChannelSubmitted, m.defaultCfg.UseBlobs)
		})
	}

}

// TestChannelManager_Requeue seeds the channel manager with blocks,
// takes a state snapshot, triggers the blocks->channels pipeline,
// and then calls Requeue. Finally, it asserts the channel manager's
// state is equal to the snapshot.
func TestChannelManager_Requeue(t *testing.T) {
	l := testlog.Logger(t, log.LevelCrit)
	cfg := channelManagerTestConfig(100, derive.SingularBatchType)
	m := NewChannelManager(l, metrics.NoopMetrics, cfg, defaultTestRollupConfig)

	// Seed channel manager with blocks
	rng := rand.New(rand.NewSource(99))
	blockA := derivetest.RandomL2BlockWithChainId(rng, 10, defaultTestRollupConfig.L2ChainID)
	blockB := derivetest.RandomL2BlockWithChainId(rng, 10, defaultTestRollupConfig.L2ChainID)

	// This is the snapshot of channel manager state we want to reinstate
	// when we requeue
	stateSnapshot := queue.Queue[*types.Block]{blockA, blockB}
	m.blocks = stateSnapshot
	require.Empty(t, m.channelQueue)

	// Trigger the blocks -> channelQueue data pipelining
	require.NoError(t, m.ensureChannelWithSpace(eth.BlockID{}))
	require.NotEmpty(t, m.channelQueue)
	require.NoError(t, m.processBlocks())

	// Assert that at least one block was processed into the channel
	require.Equal(t, 1, m.blockCursor)

	// Call the function we are testing
	m.Requeue(m.defaultCfg)

	// Ensure we got back to the state above
	require.Equal(t, m.blocks, stateSnapshot)
	require.Empty(t, m.channelQueue)
}

func TestChannelManager_PruneBlocks(t *testing.T) {
	l := testlog.Logger(t, log.LevelCrit)
	cfg := channelManagerTestConfig(100, derive.SingularBatchType)
	m := NewChannelManager(l, metrics.NoopMetrics, cfg, defaultTestRollupConfig)

	a := types.NewBlock(&types.Header{
		Number: big.NewInt(0),
	}, nil, nil, nil)
	b := types.NewBlock(&types.Header{ // This will shortly become the safe head
		Number:     big.NewInt(1),
		ParentHash: a.Hash(),
	}, nil, nil, nil)
	c := types.NewBlock(&types.Header{
		Number:     big.NewInt(2),
		ParentHash: b.Hash(),
	}, nil, nil, nil)

	require.NoError(t, m.AddL2Block(a))
	m.blockCursor += 1
	require.NoError(t, m.AddL2Block(b))
	m.blockCursor += 1
	require.NoError(t, m.AddL2Block(c))
	m.blockCursor += 1

	// Normal path
	m.pruneSafeBlocks(eth.L2BlockRef{
		Hash:   b.Hash(),
		Number: b.NumberU64(),
	})
	require.Equal(t, queue.Queue[*types.Block]{c}, m.blocks)

	// Safe chain didn't move, nothing to prune
	m.pruneSafeBlocks(eth.L2BlockRef{
		Hash:   b.Hash(),
		Number: b.NumberU64(),
	})
	require.Equal(t, queue.Queue[*types.Block]{c}, m.blocks)

	// Safe chain moved beyond the blocks we had
	// state should be cleared
	m.pruneSafeBlocks(eth.L2BlockRef{
		Hash:   c.Hash(),
		Number: uint64(99),
	})
	require.Equal(t, queue.Queue[*types.Block]{}, m.blocks)

	// No blocks to prune, NOOP
	m.pruneSafeBlocks(eth.L2BlockRef{
		Hash:   c.Hash(),
		Number: c.NumberU64(),
	})
	require.Equal(t, queue.Queue[*types.Block]{}, m.blocks)

	// Put another block in
	d := types.NewBlock(&types.Header{
		Number:     big.NewInt(3),
		ParentHash: c.Hash(),
	}, nil, nil, nil)
	require.NoError(t, m.AddL2Block(d))
	m.blockCursor += 1

	// Safe chain reorg
	// state should be cleared
	m.pruneSafeBlocks(eth.L2BlockRef{
		Hash:   a.Hash(),
		Number: uint64(3),
	})
	require.Equal(t, queue.Queue[*types.Block]{}, m.blocks)

}

func TestChannelManager_PruneChannels(t *testing.T) {
	l := testlog.Logger(t, log.LevelCrit)
	cfg := channelManagerTestConfig(100, derive.SingularBatchType)
	cfg.InitNoneCompressor()
	m := NewChannelManager(l, metrics.NoopMetrics, cfg, defaultTestRollupConfig)

	A, err := newChannel(l, metrics.NoopMetrics, cfg, m.rollupCfg, 0)
	require.NoError(t, err)
	B, err := newChannel(l, metrics.NoopMetrics, cfg, m.rollupCfg, 0)
	require.NoError(t, err)
	C, err := newChannel(l, metrics.NoopMetrics, cfg, m.rollupCfg, 0)
	require.NoError(t, err)

	m.channelQueue = []*channel{A, B, C}

	numTx := 1
	rng := rand.New(rand.NewSource(123))
	a := derivetest.RandomL2BlockWithChainId(rng, numTx, defaultTestRollupConfig.L2ChainID)
	a = a.WithSeal(&types.Header{Number: big.NewInt(0)})
	b := derivetest.RandomL2BlockWithChainId(rng, numTx, defaultTestRollupConfig.L2ChainID)
	b = b.WithSeal(&types.Header{Number: big.NewInt(1)})
	c := derivetest.RandomL2BlockWithChainId(rng, numTx, defaultTestRollupConfig.L2ChainID)
	c = c.WithSeal(&types.Header{Number: big.NewInt(2)})
	d := derivetest.RandomL2BlockWithChainId(rng, numTx, defaultTestRollupConfig.L2ChainID)
	d = d.WithSeal(&types.Header{Number: big.NewInt(3)})
	e := derivetest.RandomL2BlockWithChainId(rng, numTx, defaultTestRollupConfig.L2ChainID)
	e = e.WithSeal(&types.Header{Number: big.NewInt(4)})

	_, err = A.AddBlock(a)
	require.NoError(t, err)
	_, err = A.AddBlock(b)
	require.NoError(t, err)

	_, err = B.AddBlock(c)
	require.NoError(t, err)
	_, err = B.AddBlock(d)
	require.NoError(t, err)

	_, err = C.AddBlock(e)
	require.NoError(t, err)

	m.pruneChannels(eth.L2BlockRef{
		Number: uint64(3),
	})

	require.Equal(t, []*channel{C}, m.channelQueue)

	m.pruneChannels(eth.L2BlockRef{
		Number: uint64(4),
	})

	require.Equal(t, []*channel{}, m.channelQueue)

	m.pruneChannels(eth.L2BlockRef{
		Number: uint64(4),
	})

	require.Equal(t, []*channel{}, m.channelQueue)

}
func TestChannelManager_ChannelOutFactory(t *testing.T) {
	type ChannelOutWrapper struct {
		derive.ChannelOut
	}

	l := testlog.Logger(t, log.LevelCrit)
	cfg := channelManagerTestConfig(100, derive.SingularBatchType)
	m := NewChannelManager(l, metrics.NoopMetrics, cfg, defaultTestRollupConfig)
	m.SetChannelOutFactory(func(cfg ChannelConfig, rollupCfg *rollup.Config) (derive.ChannelOut, error) {
		co, err := NewChannelOut(cfg, rollupCfg)
		if err != nil {
			return nil, err
		}
		// return a wrapper type, to validate that the factory was correctly used by checking the type below
		return &ChannelOutWrapper{
			ChannelOut: co,
		}, nil
	})
	require.NoError(t, m.ensureChannelWithSpace(eth.BlockID{}))

	require.IsType(t, &ChannelOutWrapper{}, m.currentChannel.channelBuilder.co)
}
