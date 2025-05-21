package base

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/devnet-sdk/system"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/systest"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// TestChainFork checks that the chain does not fork (all nodes have the same block hash for a fixed block number).
func TestChainFork(t *testing.T) {
	systest.SystemTest(t,
		chainForkTestScenario(),
	)
}

func chainForkTestScenario() systest.SystemTestFunc {
	return func(t systest.T, sys system.System) {
		logger := testlog.Logger(t, log.LevelInfo)
		logger.Info("Started test")

		// Check all L2 chains
		for i, chain := range sys.L2s() {
			chainIndex := i
			currentChain := chain
			t.Run(fmt.Sprintf("Chain_%d", chainIndex), func(t systest.T) {
				t.Parallel()
				chainLogger := logger.New("chain", chainIndex)

				// Initial chain fork check
				laterCheck, err := systest.CheckForChainFork(t.Context(), currentChain, chainLogger)
				if err != nil {
					t.Fatalf("first chain fork check failed: %v", err)
				}

				// Get a geth client
				node := currentChain.Nodes()[0]
				client, err := node.GethClient()
				if err != nil {
					t.Fatalf("failed to get geth client: %v", err)
				}

				// Create a context with timeout
				ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
				defer cancel()

				chainLogger.Debug("Waiting for the next block")
				err = wait.ForNextBlock(ctx, client)
				require.NoError(t, err, "failed to wait for the next block")

				// Check for a chain fork again
				err = laterCheck(false)
				require.NoError(t, err, "second chain fork check failed")
				t.Log("Chain fork check passed")
			})
		}
	}
}
