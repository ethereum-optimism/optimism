package batcher

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-batcher/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
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

	var parentTimestamps []uint64
	m.SetChannelOutFactory(func(cfg ChannelConfig, rollupCfg *rollup.Config, parentTimestamp uint64) (derive.ChannelOut, error) {
		parentTimestamps = append(parentTimestamps, parentTimestamp)
		return NewChannelOut(cfg, rollupCfg, parentTimestamp)
	})

	block := func(number, timestamp uint64) SizedBlock {
		return mustSizedBlockFromGeth(types.NewBlockWithHeader(&types.Header{
			Number: new(big.Int).SetUint64(number), Time: timestamp, BaseFee: big.NewInt(7),
		}))
	}

	m.Clear(eth.BlockID{}, 5000)
	require.NoError(t, m.ensureChannelWithSpace(eth.BlockID{}))
	require.Equal(t, []uint64{5000}, parentTimestamps, "the first channel continues the safe head")

	// a group of two at 5002, then one block at 5004, with the first two already in a channel
	m.blocks = queue.Queue[SizedBlock]{block(101, 5002), block(102, 5002), block(103, 5004)}
	m.blockCursor = 2
	m.currentChannel = nil
	require.NoError(t, m.ensureChannelWithSpace(eth.BlockID{}))
	require.Equal(t, uint64(5002), parentTimestamps[1], "the parent is the last block handed to a channel")

	// once those two blocks are safe they leave the queue, but the newer one is still their sibling
	m.PruneSafeBlocks(2)
	m.currentChannel = nil
	require.NoError(t, m.ensureChannelWithSpace(eth.BlockID{}))
	require.Equal(t, uint64(5002), parentTimestamps[2], "the parent is the pruned safe head")
}
