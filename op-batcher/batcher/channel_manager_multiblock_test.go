package batcher

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-batcher/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	derivetest "github.com/ethereum-optimism/optimism/op-node/rollup/derive/test"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
	"github.com/ethereum-optimism/optimism/op-service/queue"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// TestChannelManager_ChannelParentTimestamp checks that every channel is opened with the timestamp
// of the block preceding the first block it will take. Channel boundaries fall wherever the
// compressor fills up, so one landing inside a group of blocks sharing a timestamp is expected;
// without the parent timestamp the span batch v2 opened there would claim its first block starts a
// new timestamp, and the resulting span would be rejected by verifiers.
func TestChannelManager_ChannelParentTimestamp(t *testing.T) {
	l := testlog.Logger(t, log.LevelCrit)
	cfg := channelManagerTestConfig(100, derive.SpanBatchType)
	m := NewChannelManager(l, metrics.NoopMetrics, cfg, defaultTestRollupConfig)

	var parentTimestamps []*uint64
	m.SetChannelOutFactory(func(cfg ChannelConfig, rollupCfg *rollup.Config, parentTimestamp *uint64) (derive.ChannelOut, error) {
		parentTimestamps = append(parentTimestamps, parentTimestamp)
		return NewChannelOut(cfg, rollupCfg, parentTimestamp)
	})

	block := func(number, timestamp uint64) SizedBlock {
		return mustSizedBlockFromGeth(types.NewBlockWithHeader(&types.Header{
			Number: new(big.Int).SetUint64(number), Time: timestamp, BaseFee: big.NewInt(7),
		}))
	}

	m.Clear(eth.BlockID{})
	m.SetSafeHeadTimestamp(5000)
	require.NoError(t, m.ensureChannelWithSpace(eth.BlockID{}))
	require.Equal(t, []*uint64{ptr.New(uint64(5000))}, parentTimestamps, "the first channel continues the safe head")

	// a group of two at 5002, then one block at 5004, with the first two already in a channel
	m.blocks = queue.Queue[SizedBlock]{block(101, 5002), block(102, 5002), block(103, 5004)}
	m.blockCursor = 2
	m.currentChannel = nil
	require.NoError(t, m.ensureChannelWithSpace(eth.BlockID{}))
	require.Equal(t, ptr.New(uint64(5002)), parentTimestamps[1], "the parent is the last block handed to a channel")

	// once those two blocks are safe they leave the queue, and the sync tick moves the safe head
	// onto the last of them; the block still pending is their sibling
	m.PruneSafeBlocks(2)
	m.SetSafeHeadTimestamp(5002)
	m.currentChannel = nil
	require.NoError(t, m.ensureChannelWithSpace(eth.BlockID{}))
	require.Equal(t, ptr.New(uint64(5002)), parentTimestamps[2], "the parent is the pruned safe head")
}

// TestBatchSubmitter_SyncAndPruneTracksSafeHead checks that the safe head timestamp is taken from
// the sync status on every tick. The batcher can sit with an empty block queue while the safe head
// advances - it waits for L2 genesis and for node sync after clearing state - and then loads
// blocks from the new safe head without any clear or prune in between. A remembered timestamp
// would be stale there, and a span batch v2 opened on it would locate the wrong parent block.
func TestBatchSubmitter_SyncAndPruneTracksSafeHead(t *testing.T) {
	m := NewChannelManager(testlog.Logger(t, log.LevelCrit), metrics.NoopMetrics,
		channelManagerTestConfig(100, derive.SpanBatchType), defaultTestRollupConfig)
	l := &BatchSubmitter{DriverSetup: DriverSetup{Log: testlog.Logger(t, log.LevelCrit)}, channelMgr: m}

	m.Clear(eth.BlockID{})
	l.syncAndPrune(&eth.SyncStatus{
		HeadL1:      eth.BlockRef{Number: 20},
		CurrentL1:   eth.BlockRef{Number: 10},
		LocalSafeL2: eth.L2BlockRef{Number: 100, Hash: common.Hash{0x64}, Time: 5000},
		UnsafeL2:    eth.L2BlockRef{Number: 101, Hash: common.Hash{0x65}},
	})
	require.Equal(t, ptr.New(uint64(5000)), m.parentTimestamp())

	// the safe head advances while the queue is still empty: no clear, no prune, but the next
	// block to load is now 201, whose parent is the safe head at 5400
	blocksToLoad := l.syncAndPrune(&eth.SyncStatus{
		HeadL1:      eth.BlockRef{Number: 30},
		CurrentL1:   eth.BlockRef{Number: 20},
		LocalSafeL2: eth.L2BlockRef{Number: 200, Hash: common.Hash{0xc8}, Time: 5400},
		UnsafeL2:    eth.L2BlockRef{Number: 201, Hash: common.Hash{0xc9}},
	})
	require.Equal(t, &inclusiveBlockRange{201, 201}, blocksToLoad)
	require.Equal(t, ptr.New(uint64(5400)), m.parentTimestamp())
}

// TestChannelManager_MultiBlockRefusesUnknownParentTimestamp checks that the channel manager
// surfaces the span batch v2 refusal instead of emitting a span whose first block claims to start
// a new timestamp when the parent it builds on is in fact its sibling.
func TestChannelManager_MultiBlockRefusesUnknownParentTimestamp(t *testing.T) {
	rollupCfg := *defaultTestRollupConfig
	rollupCfg.MultiBlockTime = ptr.New(uint64(0))

	withSafeHead := func(t *testing.T, safeHeadTimestamp *uint64) error {
		cfg := channelManagerTestConfig(120_000, derive.SpanBatchType)
		cfg.CompressorConfig.TargetOutputSize = 1 // full on the first block, so TxData has frames
		m := NewChannelManager(testlog.Logger(t, log.LevelCrit), metrics.NoopMetrics, cfg, &rollupCfg)
		m.Clear(eth.BlockID{})
		if safeHeadTimestamp != nil {
			m.SetSafeHeadTimestamp(*safeHeadTimestamp)
		}
		block := derivetest.RandomL2BlockWithChainId(rand.New(rand.NewSource(1234)), 1, rollupCfg.L2ChainID)
		require.NoError(t, m.AddL2Block(mustPayloadFromGeth(block)))
		_, err := m.TxData(eth.BlockID{}, false, pubInfo{})
		return err
	}

	require.ErrorIs(t, withSafeHead(t, nil), derive.ErrUnknownParentTimestamp)
	require.NoError(t, withSafeHead(t, ptr.New(uint64(0))))
}
