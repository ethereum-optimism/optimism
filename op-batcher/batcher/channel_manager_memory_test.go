package batcher

import (
	"math/big"
	"runtime"
	"testing"

	"github.com/ethereum-optimism/optimism/op-batcher/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestChannelManager_Memory(t *testing.T) {
	// Define the test matrix
	compressorConfigs := []struct {
		name      string
		setupFunc func(*ChannelConfig, derive.CompressionAlgo)
	}{
		{
			name: "ShadowCompressor",
			setupFunc: func(cfg *ChannelConfig, algo derive.CompressionAlgo) {
				cfg.InitShadowCompressor(algo)
			},
		},
		{
			name: "RatioCompressor",
			setupFunc: func(cfg *ChannelConfig, algo derive.CompressionAlgo) {
				cfg.InitRatioCompressor(0.6, algo)
			},
		},
		{
			name: "NoneCompressor",
			setupFunc: func(cfg *ChannelConfig, algo derive.CompressionAlgo) {
				cfg.InitNoneCompressor()
			},
		},
	}

	compressionAlgos := []struct {
		name string
		algo derive.CompressionAlgo
	}{
		{"Zlib", derive.Zlib},
		{"Brotli9", derive.Brotli9},
		{"Brotli10", derive.Brotli10},
		{"Brotli11", derive.Brotli11},
	}

	batchTypes := []struct {
		name      string
		batchType uint
	}{
		{"SingularBatch", derive.SingularBatchType},
		{"SpanBatch", derive.SpanBatchType},
	}

	// Generate test cases automatically
	for _, compressor := range compressorConfigs {
		for _, algo := range compressionAlgos {
			for _, batch := range batchTypes {
				testName := compressor.name + "_" + algo.name + "_" + batch.name

				t.Run(testName, func(t *testing.T) {
					runMemoryTest(t, batch.batchType, compressor.name, algo.algo, compressor.setupFunc)
				})
			}
		}
	}
}

func runMemoryTest(t *testing.T, batchType uint, compressorType string, compressionAlgo derive.CompressionAlgo, setupCompressor func(*ChannelConfig, derive.CompressionAlgo)) {
	log := testlog.Logger(t, log.LevelCrit)

	// Create a channel manager with small frame size to force multiple channels
	// Use smaller frame size for span batches since they're more efficient at packing data
	frameSize := uint64(1000)
	if batchType == derive.SpanBatchType {
		frameSize = 500 // Smaller frame size for span batches to create more channels
	}
	cfg := channelManagerTestConfig(frameSize, batchType)
	cfg.ChannelTimeout = 100 // Reasonable timeout
	setupCompressor(&cfg, compressionAlgo)

	// Use the existing default test rollup config to ensure chain IDs match
	m := NewChannelManager(log, metrics.NoopMetrics, cfg, defaultTestRollupConfig)
	m.Clear(eth.BlockID{})

	// Measure initial memory
	var initialMem runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&initialMem)

	// Create the first block (genesis)
	var prevBlock *types.Block

	// Add many blocks to create multiple channels
	const numBlocks = 1000
	for i := 0; i < numBlocks; i++ {
		var block *types.Block

		if i == 0 {
			block = newMiniL2BlockWithChainID(0, defaultTestRollupConfig.L2ChainID)
		} else {
			// Create a block with proper parent hash and transaction content
			block = newMiniL2BlockWithChainIDNumberParentAndL1Information(5, defaultTestRollupConfig.L2ChainID, big.NewInt(int64(i)), prevBlock.Hash(), 100, 0)
		}

		require.NoError(t, m.AddL2Block(block))
		prevBlock = block

		// Periodically process blocks and create channels
		if i%100 == 0 {
			// Ensure we have a channel
			require.NoError(t, m.ensureChannelWithSpace(eth.BlockID{
				Hash:   block.Hash(),
				Number: block.NumberU64(),
			}))

			// Process blocks into channels
			require.NoError(t, m.processBlocks())

			// Try to get transaction data to fill channels
			_, err := m.TxData(eth.BlockID{}, false)
			// It's okay if there's no data ready (io.EOF)
			if err != nil && err.Error() != "EOF" {
				require.NoError(t, err)
			}
		}
	}

	// Final processing to ensure all blocks are processed
	require.NoError(t, m.ensureChannelWithSpace(eth.BlockID{}))
	require.NoError(t, m.processBlocks())

	// Measure final memory
	var finalMem runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&finalMem)

	// Calculate memory used by the channel manager
	memUsed := finalMem.Alloc - initialMem.Alloc

	// Assert that memory usage doesn't exceed 512MB
	const maxMemoryMB = 512 * 1024 * 1024 // 512MB in bytes
	require.Less(t, memUsed, uint64(maxMemoryMB),
		"Channel manager used %d bytes (%.2f MB), exceeding 512 MB limit",
		memUsed, float64(memUsed)/1024/1024)

	// Log memory usage for debugging
	t.Logf("Compressor: %s, Algorithm: %s, Batch Type: %d",
		compressorType, compressionAlgo, batchType)
	t.Logf("Channel manager memory usage: %d bytes (%.2f MB)",
		memUsed, float64(memUsed)/1024/1024)
	t.Logf("Number of channels in queue: %d", len(m.channelQueue))
	t.Logf("Number of blocks processed: %d", len(m.blocks))

	// Verify we actually created multiple channels (unless using none compressor which might behave differently)
	if compressorType != "none" {
		require.Greater(t, len(m.channelQueue), 0, "Expected at least one channel to be created")
	}

	// Verify that blocks form a proper chain by checking parent hashes
	// (This verifies our block creation logic is correct)
	require.Greater(t, len(m.blocks), 1, "Expected multiple blocks to be queued")
	if len(m.blocks) > 1 {
		for i := 1; i < len(m.blocks); i++ {
			require.Equal(t, m.blocks[i-1].Hash(), m.blocks[i].ParentHash(),
				"Block %d should have parent hash matching block %d", i, i-1)
		}
	}
}
