package compare

import (
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-fetcher/pkg/fetcher/fetch/script"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestCompareProofs(t *testing.T) {
	tests := []struct {
		name               string
		chainList          map[uint64]ChainListEntry
		fetchOutput        map[uint64]script.ChainConfig
		expectedDiffs      map[uint64]script.FaultProofStatus
		expectError        bool
		expectedErrorChain uint64
	}{
		{
			name: "no differences",
			chainList: map[uint64]ChainListEntry{
				10: {
					ChainID: 10,
					FaultProofStatus: script.FaultProofStatus{
						Permissioned:      true,
						Permissionless:    false,
						RespectedGameType: 1,
					},
				},
			},
			fetchOutput: map[uint64]script.ChainConfig{
				10: {
					ChainId: 10,
					FaultProofStatus: script.FaultProofStatus{
						Permissioned:      true,
						Permissionless:    false,
						RespectedGameType: 1,
					},
				},
			},
			expectedDiffs: map[uint64]script.FaultProofStatus{},
			expectError:   false,
		},
		{
			name: "permissioned difference",
			chainList: map[uint64]ChainListEntry{
				10: {
					ChainID: 10,
					FaultProofStatus: script.FaultProofStatus{
						Permissioned:      true,
						Permissionless:    false,
						RespectedGameType: 1,
					},
				},
			},
			fetchOutput: map[uint64]script.ChainConfig{
				10: {
					ChainId: 10,
					FaultProofStatus: script.FaultProofStatus{
						Permissioned:      false, // Different
						Permissionless:    false,
						RespectedGameType: 1,
					},
				},
			},
			expectedDiffs: map[uint64]script.FaultProofStatus{
				10: {
					Permissioned:      false,
					Permissionless:    false,
					RespectedGameType: 1,
				},
			},
			expectError: false,
		},
		{
			name: "permissionless difference",
			chainList: map[uint64]ChainListEntry{
				10: {
					ChainID: 10,
					FaultProofStatus: script.FaultProofStatus{
						Permissioned:      true,
						Permissionless:    false,
						RespectedGameType: 1,
					},
				},
			},
			fetchOutput: map[uint64]script.ChainConfig{
				10: {
					ChainId: 10,
					FaultProofStatus: script.FaultProofStatus{
						Permissioned:      true,
						Permissionless:    true, // Different
						RespectedGameType: 1,
					},
				},
			},
			expectedDiffs: map[uint64]script.FaultProofStatus{
				10: {
					Permissioned:      true,
					Permissionless:    true,
					RespectedGameType: 1,
				},
			},
			expectError: false,
		},
		{
			name: "game type difference",
			chainList: map[uint64]ChainListEntry{
				10: {
					ChainID: 10,
					FaultProofStatus: script.FaultProofStatus{
						Permissioned:      true,
						Permissionless:    false,
						RespectedGameType: 1,
					},
				},
			},
			fetchOutput: map[uint64]script.ChainConfig{
				10: {
					ChainId: 10,
					FaultProofStatus: script.FaultProofStatus{
						Permissioned:      true,
						Permissionless:    false,
						RespectedGameType: 2, // Different
					},
				},
			},
			expectedDiffs: map[uint64]script.FaultProofStatus{
				10: {
					Permissioned:      true,
					Permissionless:    false,
					RespectedGameType: 2,
				},
			},
			expectError: false,
		},
		{
			name: "missing chain in ChainList",
			chainList: map[uint64]ChainListEntry{
				10: {
					ChainID: 10,
					FaultProofStatus: script.FaultProofStatus{
						Permissioned:      true,
						Permissionless:    false,
						RespectedGameType: 1,
					},
				},
			},
			fetchOutput: map[uint64]script.ChainConfig{
				10: {
					ChainId: 10,
					FaultProofStatus: script.FaultProofStatus{
						Permissioned:      true,
						Permissionless:    false,
						RespectedGameType: 1,
					},
				},
				20: { // This chain doesn't exist in chainList
					ChainId: 20,
					FaultProofStatus: script.FaultProofStatus{
						Permissioned:      false,
						Permissionless:    true,
						RespectedGameType: 2,
					},
				},
			},
			expectedDiffs:      nil,
			expectError:        true,
			expectedErrorChain: 20,
		},
		{
			name: "multiple differences",
			chainList: map[uint64]ChainListEntry{
				10: {
					ChainID: 10,
					FaultProofStatus: script.FaultProofStatus{
						Permissioned:      true,
						Permissionless:    false,
						RespectedGameType: 1,
					},
				},
				20: {
					ChainID: 20,
					FaultProofStatus: script.FaultProofStatus{
						Permissioned:      false,
						Permissionless:    true,
						RespectedGameType: 2,
					},
				},
			},
			fetchOutput: map[uint64]script.ChainConfig{
				10: {
					ChainId: 10,
					FaultProofStatus: script.FaultProofStatus{
						Permissioned:      false, // Different
						Permissionless:    false,
						RespectedGameType: 1,
					},
				},
				20: {
					ChainId: 20,
					FaultProofStatus: script.FaultProofStatus{
						Permissioned:      false,
						Permissionless:    true,
						RespectedGameType: 3, // Different
					},
				},
			},
			expectedDiffs: map[uint64]script.FaultProofStatus{
				10: {
					Permissioned:      false,
					Permissionless:    false,
					RespectedGameType: 1,
				},
				20: {
					Permissioned:      false,
					Permissionless:    true,
					RespectedGameType: 3,
				},
			},
			expectError: false,
		},
	}

	testLogger := log.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comparator := &Comparator{
				ChainList:   tt.chainList,
				FetchOutput: tt.fetchOutput,
				lgr:         testLogger,
			}

			diffs, err := comparator.CompareProofs()

			if tt.expectError {
				require.Error(t, err)
				if tt.expectedErrorChain > 0 {
					require.Contains(t, err.Error(),
						fmt.Sprintf("%d", tt.expectedErrorChain),
						"Error should mention the missing chain ID")
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, len(tt.expectedDiffs), len(diffs))

				for chainID, expectedStatus := range tt.expectedDiffs {
					actualStatus, exists := diffs[chainID]
					require.True(t, exists, "Expected diff for chain ID %d", chainID)
					require.Equal(t, expectedStatus.Permissioned, actualStatus.Permissioned)
					require.Equal(t, expectedStatus.Permissionless, actualStatus.Permissionless)
					require.Equal(t, expectedStatus.RespectedGameType, actualStatus.RespectedGameType)
				}
			}
		})
	}
}
