package backend

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateEngineBehavior(t *testing.T) {
	tests := []struct {
		name        string
		engineKind  engine.Kind
		syncMode    string
		networkType string
		expectError bool
	}{
		{
			name:        "Geth with full sync",
			engineKind:  engine.Geth,
			syncMode:    "full",
			networkType: "sepolia",
			expectError: false,
		},
		{
			name:        "Reth with snap sync",
			engineKind:  engine.Reth,
			syncMode:    "snap",
			networkType: "mainnet",
			expectError: false,
		},
		{
			name:        "Erigon with full sync",
			engineKind:  engine.Erigon,
			syncMode:    "full",
			networkType: "goerli",
			expectError: false,
		},
		{
			name:        "Invalid engine kind",
			engineKind:  engine.Kind("invalid"),
			syncMode:    "full",
			networkType: "sepolia",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			behavior, err := CreateEngineBehavior(tt.engineKind, tt.syncMode, tt.networkType)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, behavior)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, behavior)
				assert.Equal(t, tt.engineKind, behavior.GetEngineKind())
				assert.Equal(t, tt.syncMode, behavior.GetSyncMode())

				network := behavior.GetNetworkCharacteristics()
				assert.NotNil(t, network)
			}
		})
	}
}

func TestEngineBehaviorSupportsPostFinalizationELSync(t *testing.T) {
	tests := []struct {
		name       string
		engineKind engine.Kind
		expected   bool
	}{
		{
			name:       "Geth does not support post-finalization EL sync",
			engineKind: engine.Geth,
			expected:   false,
		},
		{
			name:       "Reth supports post-finalization EL sync",
			engineKind: engine.Reth,
			expected:   true,
		},
		{
			name:       "Erigon supports post-finalization EL sync",
			engineKind: engine.Erigon,
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			behavior, err := CreateEngineBehavior(tt.engineKind, "full", "sepolia")
			require.NoError(t, err)

			assert.Equal(t, tt.expected, behavior.SupportsPostFinalizationELSync())
		})
	}
}

func TestGetNetworkCharacteristics(t *testing.T) {
	tests := []struct {
		name              string
		networkType       string
		expectedRegenesis bool
		expectedSyncMode  string
		expectedBlockTime uint64
	}{
		{
			name:              "Mainnet characteristics",
			networkType:       "mainnet",
			expectedRegenesis: true,
			expectedSyncMode:  "full",
			expectedBlockTime: 2,
		},
		{
			name:              "Sepolia characteristics",
			networkType:       "sepolia",
			expectedRegenesis: false,
			expectedSyncMode:  "snap",
			expectedBlockTime: 2,
		},
		{
			name:              "Goerli characteristics",
			networkType:       "goerli",
			expectedRegenesis: true,
			expectedSyncMode:  "snap",
			expectedBlockTime: 2,
		},
		{
			name:              "Unknown network defaults",
			networkType:       "unknown",
			expectedRegenesis: false,
			expectedSyncMode:  "full",
			expectedBlockTime: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			behavior, err := CreateEngineBehavior(engine.Geth, "", tt.networkType)
			require.NoError(t, err)

			network := behavior.GetNetworkCharacteristics()
			assert.Equal(t, tt.expectedRegenesis, network.SupportsRegenesis)
			assert.Equal(t, tt.expectedSyncMode, network.DefaultSyncMode)
			assert.Equal(t, tt.expectedBlockTime, network.BlockTime)
		})
	}
}

func TestDefaultSyncMode(t *testing.T) {
	// Test that when syncMode is empty, it defaults to network-specific default
	behavior, err := CreateEngineBehavior(engine.Geth, "", "sepolia")
	require.NoError(t, err)

	assert.Equal(t, "snap", behavior.GetSyncMode()) // Sepolia defaults to snap sync
}

func TestCustomSyncMode(t *testing.T) {
	// Test that custom syncMode overrides network default
	behavior, err := CreateEngineBehavior(engine.Geth, "full", "sepolia")
	require.NoError(t, err)

	assert.Equal(t, "full", behavior.GetSyncMode()) // Custom sync mode overrides default
}
