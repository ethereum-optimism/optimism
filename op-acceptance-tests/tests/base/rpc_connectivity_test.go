package base

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// TestRPCConnectivity checks we can connect to L2 execution layer RPC endpoints
func TestRPCConnectivity(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	logger := testlog.Logger(t, log.LevelInfo).With("Test", "TestRPCConnectivity")
	tracer := t.Tracer()
	ctx := t.Ctx()
	logger.Info("Started L2 RPC connectivity test")

	ctx, span := tracer.Start(ctx, "test chains")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Test all L2 chains in the system
	for _, l2Chain := range sys.L2Networks() {
		_, span = tracer.Start(ctx, "test chain")
		defer span.End()

		networkName := l2Chain.String()
		t.Run(fmt.Sprintf("L2_Chain_%s", networkName), func(tt devtest.T) {
			// Get the expected chain ID from the L2Chain
			expectedChainID := l2Chain.ChainID().ToBig()
			for _, node := range l2Chain.L2ELNodes() {
				testL2ELNode(tt, ctx, logger, networkName, expectedChainID, node)
			}
		})
	}
}

// testL2ELNode tests connectivity for a single L2 execution layer node
func testL2ELNode(t devtest.T, ctx context.Context, logger log.Logger, chainName string, expectedChainID *big.Int, elNode *dsl.L2ELNode) {
	if elNode == nil {
		return
	}

	t.Run(fmt.Sprintf("L2EL_Node_%s", elNode.String()), func(tt devtest.T) {
		// Get Ethereum client for the node
		client := elNode.Escape().EthClient()

		// Check if we can get the chain ID
		chainID, err := client.ChainID(ctx)
		require.NoError(tt, err, "failed to get chain ID from L2 execution RPC")
		require.Equal(tt, expectedChainID, chainID, "L2 chain ID does not match expected value")

		// Check if we can get the latest block
		latestBlockRef, err := client.BlockRefByLabel(ctx, "latest")
		require.NoError(tt, err, "failed to get latest block from L2 execution RPC")
		require.NotZero(tt, latestBlockRef.Hash, "L2 block hash seems invalid")

		logger.Info("L2 execution RPC connectivity test passed",
			"chain", chainName,
			"node", elNode.String(),
			"chain_id", chainID,
		)
	})
}
