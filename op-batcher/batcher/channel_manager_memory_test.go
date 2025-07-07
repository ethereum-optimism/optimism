package batcher

import (
	"math/big"
	"math/rand"
	"runtime"
	"testing"

	"github.com/ethereum-optimism/optimism/op-batcher/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	derivetest "github.com/ethereum-optimism/optimism/op-node/rollup/derive/test"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestChannelManager_Memory(t *testing.T) {
	log := testlog.Logger(t, log.LevelCrit)
	// Use a fixed seed to make the test deterministic
	rng := rand.New(rand.NewSource(42))

	// Create a channel manager with small frame size to force multiple channels
	cfg := channelManagerTestConfig(1000, derive.SingularBatchType) // Small frame size
	cfg.ChannelTimeout = 100                                        // Reasonable timeout
	cfg.InitShadowCompressor(derive.Brotli10)

	// Use default test rollup config
	rollupCfg := &rollup.Config{
		L2ChainID: big.NewInt(42),
	}

	m := NewChannelManager(log, metrics.NoopMetrics, cfg, rollupCfg)
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
			// Create genesis block
			block = derivetest.RandomL2BlockWithChainId(rng, 50, rollupCfg.L2ChainID)
		} else {
			// Create a block with proper parent hash to form a chain
			blockHeader := derivetest.RandomL2BlockWithChainId(rng, 50, rollupCfg.L2ChainID).Header()
			blockHeader.Number = big.NewInt(int64(i))
			blockHeader.ParentHash = prevBlock.Hash()

			block = types.NewBlock(blockHeader, nil, nil, nil, types.DefaultBlockConfig)
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
	const maxMemoryGB = 512 * 1024 * 1024 // 512MB in bytes
	require.Less(t, memUsed, uint64(maxMemoryGB),
		"Channel manager used %d bytes (%.2f MB), exceeding 512 MB limit",
		memUsed, float64(memUsed)/1024/1024)

	// Log memory usage for debugging
	t.Logf("Channel manager memory usage: %d bytes (%.2f MB)",
		memUsed, float64(memUsed)/1024/1024)
	t.Logf("Number of channels in queue: %d", len(m.channelQueue))
	t.Logf("Number of blocks processed: %d", len(m.blocks))

	// Verify we actually created multiple channels
	require.Greater(t, len(m.channelQueue), 0, "Expected at least one channel to be created")

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
