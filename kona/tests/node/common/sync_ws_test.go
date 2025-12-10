package node

import (
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	node_utils "github.com/op-rs/kona/node/utils"
	"github.com/stretchr/testify/require"
)

// Check that unsafe heads eventually consolidate and become safe
// For this test to be deterministic it should...
// 1. Only have two L2 nodes
// 2. Only have one DA layer node
func TestSyncUnsafeBecomesSafe(gt *testing.T) {
	const SECS_WAIT_FOR_UNSAFE_HEAD = 10
	// We are waiting longer for the safe head to sync because it is usually a few seconds behind the unsafe head.
	const SECS_WAIT_FOR_SAFE_HEAD = 60

	t := devtest.ParallelT(gt)

	out := node_utils.NewMixedOpKona(t)

	nodes := out.L2CLKonaNodes()

	// Ensure that all the nodes advance the unsafe and safe head
	advancedFns := make([]dsl.CheckFunc, 0, len(nodes))
	for _, node := range nodes {
		advancedFns = append(advancedFns, node.AdvancedFn(types.LocalSafe, 20, 80))
		advancedFns = append(advancedFns, node.AdvancedFn(types.LocalUnsafe, 20, 80))
	}
	dsl.CheckAll(t, advancedFns...)

	var wg sync.WaitGroup
	for _, node := range nodes {
		wg.Add(1)
		go func(node *dsl.L2CLNode) {
			defer wg.Done()

			unsafeBlocks := node_utils.GetKonaWs(t, node, "unsafe_head", time.After(SECS_WAIT_FOR_UNSAFE_HEAD*time.Second))

			safeBlocks := node_utils.GetKonaWs(t, node, "safe_head", time.After(SECS_WAIT_FOR_SAFE_HEAD*time.Second))

			require.GreaterOrEqual(t, len(unsafeBlocks), 1, "we didn't receive enough unsafe gossip blocks!")
			require.GreaterOrEqual(t, len(safeBlocks), 1, "we didn't receive enough safe gossip blocks!")

			safeBlockMap := make(map[uint64]eth.L2BlockRef)
			// Create a map of safe blocks with block number as the key
			for _, safeBlock := range safeBlocks {
				safeBlockMap[safeBlock.Number] = safeBlock
			}

			cond := false

			// Iterate over unsafe blocks and find matching safe blocks
			for _, unsafeBlock := range unsafeBlocks {
				if safeBlock, exists := safeBlockMap[unsafeBlock.Number]; exists {
					require.Equal(t, unsafeBlock, safeBlock, "unsafe block %d doesn't match safe block %d", unsafeBlock.Number, safeBlock.Number)
					cond = true
				}
			}

			require.True(t, cond, "No matching safe block found for unsafe block")

			t.Log("✓ unsafe and safe head blocks match between all nodes")
		}(&node)
	}
	wg.Wait()
}

// System tests that ensure that the kona-nodes are syncing the unsafe chain.
// Note: this test should only be ran on networks that don't reorg, only have one sequencer and that only have one DA layer node.
func TestSyncUnsafe(gt *testing.T) {
	t := devtest.ParallelT(gt)

	out := node_utils.NewMixedOpKona(t)

	nodes := out.L2CLKonaNodes()

	// Ensure that all the nodes advance the unsafe head
	advancedFns := make([]dsl.CheckFunc, 0, len(nodes))
	for _, node := range nodes {
		advancedFns = append(advancedFns, node.AdvancedFn(types.LocalUnsafe, 20, 80))
	}
	dsl.CheckAll(t, advancedFns...)

	var wg sync.WaitGroup
	for _, node := range nodes {
		wg.Add(1)
		go func(node *dsl.L2CLNode) {
			defer wg.Done()

			output := node_utils.GetKonaWs(t, node, "unsafe_head", time.After(2*time.Minute))

			// For each block, we check that the block is actually in the chain of the other nodes.
			// That should always be the case unless there is a reorg or a long sync.
			// We shouldn't have safe heads reorgs in this very simple testnet because there is only one DA layer node.
			for _, block := range output {
				for _, node := range nodes {
					otherCLNode := node.Escape().ID().Key()
					otherCLSyncStatus := node.ChainSyncStatus(out.L2Chain.ChainID(), types.LocalUnsafe)

					if otherCLSyncStatus.Number < block.Number {
						t.Log("✗ peer too far behind!", otherCLNode, block.Number, otherCLSyncStatus.Number)
						continue
					}

					expectedOutputResponse, err := node.Escape().RollupAPI().OutputAtBlock(t.Ctx(), block.Number)
					require.NoError(t, err, "impossible to get block from node %s", otherCLNode)

					// Make sure the blocks match!
					require.Equal(t, expectedOutputResponse.BlockRef, block, "block mismatch between %s and %s", otherCLNode, node.Escape().ID().Key())
				}
			}

			t.Log("✓ unsafe head blocks match between all nodes")
		}(&node)
	}
	wg.Wait()
}

// System tests that ensure that the kona-nodes are syncing the safe chain.
// Note: this test should only be ran on networks that don't reorg and that only have one DA layer node.
func TestSyncSafe(gt *testing.T) {
	t := devtest.ParallelT(gt)

	out := node_utils.NewMixedOpKona(t)

	nodes := out.L2CLKonaNodes()

	// Ensure that all the nodes advance the safe head
	advancedFns := make([]dsl.CheckFunc, 0, len(nodes))
	for _, node := range nodes {
		advancedFns = append(advancedFns, node.AdvancedFn(types.LocalSafe, 20, 80))
	}
	dsl.CheckAll(t, advancedFns...)

	var wg sync.WaitGroup
	for _, node := range nodes {
		wg.Add(1)
		go func(node *dsl.L2CLNode) {
			defer wg.Done()
			clName := node.Escape().ID().Key()

			output := node_utils.GetKonaWs(t, node, "safe_head", time.After(2*time.Minute))

			// For each block, we check that the block is actually in the chain of the other nodes.
			// That should always be the case unless there is a reorg or a long sync.
			// We shouldn't have safe heads reorgs in this very simple testnet because there is only one DA layer node.
			for _, block := range output {
				for _, node := range nodes {
					otherCLNode := node.Escape().ID().Key()
					otherCLSyncStatus := node.ChainSyncStatus(out.L2Chain.ChainID(), types.LocalSafe)

					if otherCLSyncStatus.Number < block.Number {
						t.Log("✗ peer too far behind!", otherCLNode, block.Number, otherCLSyncStatus.Number)
						continue
					}

					expectedOutputResponse, err := node.Escape().RollupAPI().OutputAtBlock(t.Ctx(), block.Number)
					require.NoError(t, err, "impossible to get block from node %s", otherCLNode)

					// Make sure the blocks match!
					require.Equal(t, expectedOutputResponse.BlockRef, block, "block mismatch between %s and %s", otherCLNode, clName)
				}
			}

			t.Log("✓ safe head blocks match between all nodes")
		}(&node)
	}
	wg.Wait()
}

// System tests that ensure that the kona-nodes are syncing the finalized chain.
// Note: this test can be ran on any sort of network, including the ones that should reorg.
func TestSyncFinalized(gt *testing.T) {
	t := devtest.ParallelT(gt)

	out := node_utils.NewMixedOpKona(t)

	nodes := out.L2CLKonaNodes()

	var wg sync.WaitGroup
	for _, node := range nodes {
		wg.Add(1)
		go func(node *dsl.L2CLNode) {
			defer wg.Done()
			clName := node.Escape().ID().Key()

			output := node_utils.GetKonaWs(t, node, "finalized_head", time.After(4*time.Minute))

			// We should check that we received at least 1 finalized block within 4 minutes!
			require.GreaterOrEqual(t, len(output), 1, "we didn't receive enough finalized gossip blocks!")
			t.Log("Number of finalized blocks received within 4 minutes:", len(output))

			// For each block, we check that the block is actually in the chain of the other nodes.
			for _, block := range output {
				for _, node := range nodes {
					otherCLNode := node.Escape().ID().Key()
					otherCLSyncStatus := node.ChainSyncStatus(out.L2Chain.ChainID(), types.Finalized)

					if otherCLSyncStatus.Number < block.Number {
						t.Log("✗ peer too far behind!", otherCLNode, block.Number, otherCLSyncStatus.Number)
						continue
					}

					expectedOutputResponse, err := node.Escape().RollupAPI().OutputAtBlock(t.Ctx(), block.Number)
					require.NoError(t, err, "impossible to get block from node %s", otherCLNode)

					// Make sure the blocks match!
					require.Equal(t, expectedOutputResponse.BlockRef, block, "block mismatch between %s and %s", otherCLNode, clName)
				}
			}

			t.Log("✓ finalized head blocks match between all nodes")
		}(&node)
	}
	wg.Wait()
}
