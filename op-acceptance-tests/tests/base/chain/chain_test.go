package chain

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestChainFork checks that the chain does not fork (all nodes have the same block hash for a fixed block number).
func TestChainFork(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)

	t.Logger().Info("Started chain fork test")

	// Check all L2 networks
	for i, network := range sys.L2Networks() {
		networkIndex := i
		currentNetwork := network
		t.Run(fmt.Sprintf("Network_%d", networkIndex), func(t devtest.T) {
			t.Parallel()
			networkLogger := t.Logger().New("network", networkIndex)

			// Initial chain fork check
			laterCheck, err := dsl.CheckForChainFork(t.Ctx(), []*dsl.L2Network{currentNetwork}, networkLogger)
			require := t.Require()
			require.NoError(err, "first chain fork check failed")

			// Get an eth client from the first node
			underlyingNetwork := currentNetwork.Escape()
			if len(underlyingNetwork.L2ELNodes()) == 0 {
				t.Logger().Error("no L2 EL nodes found")
				t.FailNow()
			}
			client := underlyingNetwork.L2ELNodes()[0].L2EthClient()

			// Create a context with timeout
			ctx, cancel := context.WithTimeout(t.Ctx(), 60*time.Second)
			defer cancel()

			networkLogger.Debug("Waiting for the next block")

			// Get current block
			_, err = client.InfoByLabel(ctx, eth.Safe)
			require.NoError(err, "failed to get current block")

			// Wait for the next block
			currentNetwork.WaitForBlock()
			require.NoError(err, "failed to wait for the next block")

			// Check for a chain fork again
			err = laterCheck(false)
			require.NoError(err, "second chain fork check failed")
			t.Log("Chain fork check passed")
		})
	}
}
