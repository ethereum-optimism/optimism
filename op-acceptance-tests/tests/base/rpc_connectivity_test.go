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
		for chainIdx, l2Chain := range sys.L2s() {
			chainIdx := chainIdx
			t.Run(fmt.Sprintf("L2_Chain_%d", chainIdx), func(t systest.T) {
				require.NotEmpty(t, l2Chain.Nodes(), "L2 chain has no nodes")

				// Get the first node
				execNode := l2Chain.Nodes()[0]

				// Connect to the node's RPC
				client, err := execNode.GethClient()
				require.NoError(t, err, "failed to connect to L2 execution RPC")

				// Check if we can get chain ID
				chainID, err := client.ChainID(ctx)
				require.NoError(t, err, "failed to get chain ID from L2 execution RPC")
				require.Equal(t, chainIdx, chainID, "L2 chain ID is not correct")

				// Check if we can get the latest block number
				blockNumber, err := client.BlockNumber(ctx)
				require.NoError(t, err, "failed to get block number from L2 execution RPC")
				require.Greater(t, blockNumber, uint64(0), "L2 block number should be greater than 0")

				logger.Info("L2 execution RPC connectivity test passed",
					"chain", chainIdx,
					"node", execNode.Name(),
					"chain_id", chainID,
					"block_number", blockNumber)
			})
		}
	}
}
