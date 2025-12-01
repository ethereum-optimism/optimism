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
				GameMetadata: types.GameMetadata{GameType: gameType},
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

func TestEnrichedGameData_RollupEndpointErrorCountInitialization(t *testing.T) {
	data := EnrichedGameData{}
	require.Equal(t, 0, data.RollupEndpointErrorCount, "RollupEndpointErrorCount should default to 0")
}

func TestEnrichedGameData_HasMixedAvailability(t *testing.T) {
	tests := []struct {
		name                        string
		rollupEndpointTotalCount    int
		rollupEndpointErrorCount    int
		rollupEndpointNotFoundCount int
		expected                    bool
	}{
		{
			name:                        "no endpoints attempted",
			rollupEndpointTotalCount:    0,
			rollupEndpointErrorCount:    0,
			rollupEndpointNotFoundCount: 0,
			expected:                    false,
		},
		{
			name:                        "all endpoints successful",
			rollupEndpointTotalCount:    3,
			rollupEndpointErrorCount:    0,
			rollupEndpointNotFoundCount: 0,
			expected:                    false,
		},
		{
			name:                        "all endpoints had errors",
			rollupEndpointTotalCount:    3,
			rollupEndpointErrorCount:    3,
			rollupEndpointNotFoundCount: 0,
			expected:                    false,
		},
		{
			name:                        "all endpoints returned not found",
			rollupEndpointTotalCount:    3,
			rollupEndpointErrorCount:    0,
			rollupEndpointNotFoundCount: 3,
			expected:                    false,
		},
		{
			name:                        "mixed availability - some not found, some successful",
			rollupEndpointTotalCount:    3,
			rollupEndpointErrorCount:    0,
			rollupEndpointNotFoundCount: 1,
			expected:                    true,
		},
		{
			name:                        "mixed availability with errors - some not found, some successful, some errors",
			rollupEndpointTotalCount:    5,
			rollupEndpointErrorCount:    1,
			rollupEndpointNotFoundCount: 2,
			expected:                    true,
		},
		{
			name:                        "mixed availability - majority not found",
			rollupEndpointTotalCount:    4,
			rollupEndpointErrorCount:    0,
			rollupEndpointNotFoundCount: 3,
			expected:                    true,
		},
		{
			name:                        "no successful endpoints - only errors and not found",
			rollupEndpointTotalCount:    4,
			rollupEndpointErrorCount:    2,
			rollupEndpointNotFoundCount: 2,
			expected:                    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := EnrichedGameData{
				RollupEndpointTotalCount:    test.rollupEndpointTotalCount,
				RollupEndpointErrorCount:    test.rollupEndpointErrorCount,
				RollupEndpointNotFoundCount: test.rollupEndpointNotFoundCount,
			}
			result := data.HasMixedAvailability()
			require.Equal(t, test.expected, result)
		})
	}
}

func TestEnrichedGameData_HasMixedSafety(t *testing.T) {
	tests := []struct {
		name                      string
		rollupEndpointSafeCount   int
		rollupEndpointUnsafeCount int
		expected                  bool
	}{
		{
			name:                      "no safety assessments",
			rollupEndpointSafeCount:   0,
			rollupEndpointUnsafeCount: 0,
			expected:                  false,
		},
		{
			name:                      "all endpoints report safe",
			rollupEndpointSafeCount:   3,
			rollupEndpointUnsafeCount: 0,
			expected:                  false,
		},
		{
			name:                      "all endpoints report unsafe",
			rollupEndpointSafeCount:   0,
			rollupEndpointUnsafeCount: 3,
			expected:                  false,
		},
		{
			name:                      "mixed safety - some safe, some unsafe",
			rollupEndpointSafeCount:   2,
			rollupEndpointUnsafeCount: 1,
			expected:                  true,
		},
		{
			name:                      "mixed safety - minority safe",
			rollupEndpointSafeCount:   1,
			rollupEndpointUnsafeCount: 4,
			expected:                  true,
		},
		{
			name:                      "mixed safety - majority safe",
			rollupEndpointSafeCount:   4,
			rollupEndpointUnsafeCount: 1,
			expected:                  true,
		},
		{
			name:                      "mixed safety - equal split",
			rollupEndpointSafeCount:   2,
			rollupEndpointUnsafeCount: 2,
			expected:                  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := EnrichedGameData{
				RollupEndpointSafeCount:   test.rollupEndpointSafeCount,
				RollupEndpointUnsafeCount: test.rollupEndpointUnsafeCount,
			}
			result := data.HasMixedSafety()
			require.Equal(t, test.expected, result)
		})
	}
}
