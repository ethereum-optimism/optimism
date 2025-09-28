package backend

import (
	"context"
	"testing"

	rollupengine "github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEngineBehaviorIntegration tests the integration of different engine behaviors
func TestEngineBehaviorIntegration(t *testing.T) {
	tests := []struct {
		name              string
		engineKind        rollupengine.Kind
		syncMode          string
		networkType       string
		expectedSync      bool
		expectedRegenesis bool
	}{
		{
			name:              "Geth on Sepolia with full sync",
			engineKind:        rollupengine.Geth,
			syncMode:          "full",
			networkType:       "sepolia",
			expectedSync:      false, // Geth doesn't support post-finalization EL sync
			expectedRegenesis: false,
		},
		{
			name:              "Reth on Mainnet with snap sync",
			engineKind:        rollupengine.Reth,
			syncMode:          "snap",
			networkType:       "mainnet",
			expectedSync:      true, // Reth supports post-finalization EL sync
			expectedRegenesis: true,
		},
		{
			name:              "Erigon on Goerli with full sync",
			engineKind:        rollupengine.Erigon,
			syncMode:          "full",
			networkType:       "goerli",
			expectedSync:      true, // Erigon supports post-finalization EL sync
			expectedRegenesis: true,
		},
		{
			name:              "Default Geth behavior",
			engineKind:        rollupengine.Geth,
			syncMode:          "", // Empty sync mode should use network default
			networkType:       "sepolia",
			expectedSync:      false,
			expectedRegenesis: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create engine behavior
			behavior, err := CreateEngineBehavior(tt.engineKind, tt.syncMode, tt.networkType)
			require.NoError(t, err)
			require.NotNil(t, behavior)

			// Test engine kind
			assert.Equal(t, tt.engineKind, behavior.GetEngineKind())

			// Test post-finalization EL sync support
			assert.Equal(t, tt.expectedSync, behavior.SupportsPostFinalizationELSync())

			// Test network characteristics
			network := behavior.GetNetworkCharacteristics()
			assert.Equal(t, tt.expectedRegenesis, network.SupportsRegenesis)
			assert.Equal(t, uint64(2), network.BlockTime) // All networks have 2s block time

			// Test sync mode
			expectedSyncMode := tt.syncMode
			if expectedSyncMode == "" {
				// Should use network default
				switch tt.networkType {
				case "sepolia":
					expectedSyncMode = "snap"
				case "mainnet", "goerli":
					expectedSyncMode = "full"
				default:
					expectedSyncMode = "full"
				}
			}
			assert.Equal(t, expectedSyncMode, behavior.GetSyncMode())

			// Test that behavior methods can be called without errors
			logger := testlog.Logger(t, log.LevelInfo)
			ctx := context.Background()

			// Create a dummy session for testing
			session := &eth.SyncTesterSession{
				SessionID: "test-session",
				CurrentState: eth.FCUState{
					Latest:    100,
					Safe:      95,
					Finalized: 90,
				},
			}

			// Test HandleForkchoiceUpdate (should delegate to default implementation)
			result, err := behavior.HandleForkchoiceUpdate(ctx, session, logger, nil, nil, 0, false, false)
			assert.NoError(t, err)
			assert.Nil(t, result) // Should delegate to default implementation

			// Test HandleNewPayload (should delegate to default implementation)
			payload := &eth.ExecutionPayload{
				BlockHash: common.Hash{0x1},
			}
			status, err := behavior.HandleNewPayload(ctx, session, logger, payload, nil, nil, nil, false, false)
			assert.NoError(t, err)
			assert.Nil(t, status) // Should delegate to default implementation
		})
	}
}

// TestEngineBehaviorComparison tests the differences between engine behaviors
func TestEngineBehaviorComparison(t *testing.T) {
	// Create behaviors for all three engines
	gethBehavior, err := CreateEngineBehavior(rollupengine.Geth, "full", "sepolia")
	require.NoError(t, err)

	rethBehavior, err := CreateEngineBehavior(rollupengine.Reth, "snap", "mainnet")
	require.NoError(t, err)

	erigonBehavior, err := CreateEngineBehavior(rollupengine.Erigon, "full", "goerli")
	require.NoError(t, err)

	// Test post-finalization EL sync support differences
	assert.False(t, gethBehavior.SupportsPostFinalizationELSync(), "Geth should not support post-finalization EL sync")
	assert.True(t, rethBehavior.SupportsPostFinalizationELSync(), "Reth should support post-finalization EL sync")
	assert.True(t, erigonBehavior.SupportsPostFinalizationELSync(), "Erigon should support post-finalization EL sync")

	// Test network characteristics
	gethNetwork := gethBehavior.GetNetworkCharacteristics()
	rethNetwork := rethBehavior.GetNetworkCharacteristics()
	erigonNetwork := erigonBehavior.GetNetworkCharacteristics()

	assert.False(t, gethNetwork.SupportsRegenesis, "Sepolia should not support regenesis")
	assert.True(t, rethNetwork.SupportsRegenesis, "Mainnet should support regenesis")
	assert.True(t, erigonNetwork.SupportsRegenesis, "Goerli should support regenesis")

	// Test sync modes
	assert.Equal(t, "full", gethBehavior.GetSyncMode())
	assert.Equal(t, "snap", rethBehavior.GetSyncMode())
	assert.Equal(t, "full", erigonBehavior.GetSyncMode())
}

// TestEngineBehaviorWithDifferentNetworks tests engine behavior across different networks
func TestEngineBehaviorWithDifferentNetworks(t *testing.T) {
	networks := []struct {
		name                string
		networkType         string
		expectedRegenesis   bool
		expectedDefaultSync string
	}{
		{"Mainnet", "mainnet", true, "full"},
		{"Sepolia", "sepolia", false, "snap"},
		{"Goerli", "goerli", true, "snap"},
		{"Unknown", "unknown", false, "full"},
	}

	for _, network := range networks {
		t.Run(network.name, func(t *testing.T) {
			// Test with Geth
			behavior, err := CreateEngineBehavior(rollupengine.Geth, "", network.networkType)
			require.NoError(t, err)

			networkChars := behavior.GetNetworkCharacteristics()
			assert.Equal(t, network.expectedRegenesis, networkChars.SupportsRegenesis,
				"Network %s regenesis support", network.name)
			assert.Equal(t, network.expectedDefaultSync, behavior.GetSyncMode(),
				"Network %s default sync mode", network.name)
		})
	}
}
