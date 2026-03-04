package types

import (
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/stretchr/testify/require"
)

func TestEnrichedGameData_UsesOutputRoots(t *testing.T) {
	for _, gameType := range outputRootGameTypes {
		gameType := gameType
		t.Run(fmt.Sprintf("GameType-%v", gameType), func(t *testing.T) {
			data := EnrichedGameData{
				GameMetadata: types.GameMetadata{GameType: uint32(gameType)},
			}
			require.True(t, data.UsesOutputRoots())
		})
	}

	nonOutputRootTypes := []uint32{4, 5, 9, 42982, 20013130}
	for _, gameType := range nonOutputRootTypes {
		gameType := gameType
		t.Run(fmt.Sprintf("GameType-%v", gameType), func(t *testing.T) {
			data := EnrichedGameData{
				GameMetadata: types.GameMetadata{GameType: gameType},
			}
			require.False(t, data.UsesOutputRoots())
		})
	}
}

func TestEnrichedGameData_NodeEndpointErrorCountInitialization(t *testing.T) {
	data := EnrichedGameData{}
	require.Equal(t, 0, data.NodeEndpointErrorCount, "NodeEndpointErrorCount should default to 0")
}

func TestEnrichedGameData_HasMixedAvailability(t *testing.T) {
	tests := []struct {
		name                      string
		nodeEndpointTotalCount    int
		nodeEndpointErrorCount    int
		nodeEndpointNotFoundCount int
		expected                  bool
	}{
		{
			name:                      "no endpoints attempted",
			nodeEndpointTotalCount:    0,
			nodeEndpointErrorCount:    0,
			nodeEndpointNotFoundCount: 0,
			expected:                  false,
		},
		{
			name:                      "all endpoints successful",
			nodeEndpointTotalCount:    3,
			nodeEndpointErrorCount:    0,
			nodeEndpointNotFoundCount: 0,
			expected:                  false,
		},
		{
			name:                      "all endpoints had errors",
			nodeEndpointTotalCount:    3,
			nodeEndpointErrorCount:    3,
			nodeEndpointNotFoundCount: 0,
			expected:                  false,
		},
		{
			name:                      "all endpoints returned not found",
			nodeEndpointTotalCount:    3,
			nodeEndpointErrorCount:    0,
			nodeEndpointNotFoundCount: 3,
			expected:                  false,
		},
		{
			name:                      "mixed availability - some not found, some successful",
			nodeEndpointTotalCount:    3,
			nodeEndpointErrorCount:    0,
			nodeEndpointNotFoundCount: 1,
			expected:                  true,
		},
		{
			name:                      "mixed availability with errors - some not found, some successful, some errors",
			nodeEndpointTotalCount:    5,
			nodeEndpointErrorCount:    1,
			nodeEndpointNotFoundCount: 2,
			expected:                  true,
		},
		{
			name:                      "mixed availability - majority not found",
			nodeEndpointTotalCount:    4,
			nodeEndpointErrorCount:    0,
			nodeEndpointNotFoundCount: 3,
			expected:                  true,
		},
		{
			name:                      "no successful endpoints - only errors and not found",
			nodeEndpointTotalCount:    4,
			nodeEndpointErrorCount:    2,
			nodeEndpointNotFoundCount: 2,
			expected:                  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := EnrichedGameData{
				NodeEndpointTotalCount:    test.nodeEndpointTotalCount,
				NodeEndpointErrorCount:    test.nodeEndpointErrorCount,
				NodeEndpointNotFoundCount: test.nodeEndpointNotFoundCount,
			}
			result := data.HasMixedAvailability()
			require.Equal(t, test.expected, result)
		})
	}
}

func TestEnrichedGameData_HasMixedSafety(t *testing.T) {
	tests := []struct {
		name                    string
		nodeEndpointSafeCount   int
		nodeEndpointUnsafeCount int
		expected                bool
	}{
		{
			name:                    "no safety assessments",
			nodeEndpointSafeCount:   0,
			nodeEndpointUnsafeCount: 0,
			expected:                false,
		},
		{
			name:                    "all endpoints report safe",
			nodeEndpointSafeCount:   3,
			nodeEndpointUnsafeCount: 0,
			expected:                false,
		},
		{
			name:                    "all endpoints report unsafe",
			nodeEndpointSafeCount:   0,
			nodeEndpointUnsafeCount: 3,
			expected:                false,
		},
		{
			name:                    "mixed safety - some safe, some unsafe",
			nodeEndpointSafeCount:   2,
			nodeEndpointUnsafeCount: 1,
			expected:                true,
		},
		{
			name:                    "mixed safety - minority safe",
			nodeEndpointSafeCount:   1,
			nodeEndpointUnsafeCount: 4,
			expected:                true,
		},
		{
			name:                    "mixed safety - majority safe",
			nodeEndpointSafeCount:   4,
			nodeEndpointUnsafeCount: 1,
			expected:                true,
		},
		{
			name:                    "mixed safety - equal split",
			nodeEndpointSafeCount:   2,
			nodeEndpointUnsafeCount: 2,
			expected:                true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := EnrichedGameData{
				NodeEndpointSafeCount:   test.nodeEndpointSafeCount,
				NodeEndpointUnsafeCount: test.nodeEndpointUnsafeCount,
			}
			result := data.HasMixedSafety()
			require.Equal(t, test.expected, result)
		})
	}
}
func TestAllSupportedGameTypesAreOutputOrSuperRootType(t *testing.T) {
	for _, gameType := range types.SupportedGameTypes {
		t.Run(gameType.String(), func(t *testing.T) {
			data := EnrichedGameData{
				GameMetadata: types.GameMetadata{
					GameType: uint32(gameType),
				},
			}
			if data.UsesOutputRoots() {
				require.Contains(t, outputRootGameTypes, gameType)
				require.NotContains(t, superRootGameTypes, gameType)
			} else {
				require.Contains(t, superRootGameTypes, gameType)
				require.NotContains(t, outputRootGameTypes, gameType)
			}
		})
	}
}
