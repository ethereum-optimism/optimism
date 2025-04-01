package base

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/devnet-sdk/system"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/systest"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
)

// TestChainFork checks that the chain does not fork.
func TestChainFork(t *testing.T) {
	chainIdx := uint64(0)

	systest.SystemTest(t,
		chainForkTestScenario(chainIdx),
	)
}

func chainForkTestScenario(chainIdx uint64) systest.SystemTestFunc {
	return func(t systest.T, sys system.System) {
		const numCyclesToWait = 5

		logger := testlog.Logger(t, log.LevelInfo)
		logger.Info("Started test")

		chain := sys.L2s()[chainIdx]

		// Check for a chain fork
		logger.Info("Checking for chain fork")
		laterCheck, err := systest.CheckForChainFork(t.Context(), chain, logger)
		require.NoError(t, err, "first chain fork check failed")

		// Wait for a potential chain fork
		logger.Info("Waiting for a potential chain fork")
		time.Sleep(numCyclesToWait * 2 * time.Second)

		// Check for a chain fork again
		err = laterCheck()
		require.NoError(t, err, "second chain fork check failed")
		t.Log("Chain fork check passed")

	}
}
