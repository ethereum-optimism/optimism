package base

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/devnet-sdk/system"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/systest"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// TestRPCConnectivity checks we can connect to L2 execution layer RPC endpoints
func TestRPCConnectivity(t *testing.T) {
	systest.SystemTest(t,
		rpcConnectivityTestScenario(),
	)
}

func rpcConnectivityTestScenario() systest.SystemTestFunc {
	return func(t systest.T, sys system.System) {
		logger := testlog.Logger(t, log.LevelInfo)
		logger.Info("Started L2 RPC connectivity test")
		ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
		defer cancel()

		// Test each L2 chain's execution RPC
		logger.Debug("Found %d L2 chains", "chains", len(sys.L2s()))
		for chainIdx, l2Chain := range sys.L2s() {
			t.Run(fmt.Sprintf("L2_Chain_%d", chainIdx), func(t systest.T) {
				require.NotEmpty(t, l2Chain.Nodes(), "L2 chain has no nodes")

				// Get the expected chain ID from the L2Chain
				expectedChainID := l2Chain.ID()

				// Test connectivity with each node in the L2 chain
				logger.Debug("Found %d nodes in L2 chain", "nodes", len(l2Chain.Nodes()))
				for nodeIdx, execNode := range l2Chain.Nodes() {
					t.Run(fmt.Sprintf("Node_%d", nodeIdx), func(t systest.T) {
						// Connect to the node's RPC
						client, err := execNode.GethClient()
						require.NoError(t, err, "failed to connect to L2 execution RPC")

						// Check if we can get the chain ID
						chainID, err := client.ChainID(ctx)
						require.NoError(t, err, "failed to get chain ID from L2 execution RPC")
						require.Equal(t, expectedChainID, chainID, "L2 chain ID does not match expected value")

						// Check if we can get the latest block number
						blockNumber, err := client.BlockNumber(ctx)
						require.NoError(t, err, "failed to get block number from L2 execution RPC")
						require.Greater(t, blockNumber, uint64(0), "L2 block number should be greater than 0")

						logger.Info("L2 execution RPC connectivity test passed",
							"chain", chainIdx,
							"node", execNode.Name(),
							"node_idx", nodeIdx,
							"chain_id", chainID,
							"block_number", blockNumber)
					})
				}
			})
		}
	}
}
