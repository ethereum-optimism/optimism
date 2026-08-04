package types

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestBondGameDataRecipientAddresses(t *testing.T) {
	addresses := [6]common.Address{{0x06}, {0x01}, {0x05}, {0x03}, {0x02}, {0x04}}
	data := BondGameData{
		Recipients:      map[common.Address]bool{addresses[0]: true},
		Credits:         map[common.Address]*big.Int{addresses[1]: big.NewInt(1)},
		ExpectedCredits: map[common.Address]*big.Int{addresses[2]: big.NewInt(2)},
		WithdrawalRequests: map[common.Address]*contracts.WithdrawalRequest{
			addresses[3]: {},
		},
		Bonds: []BondRecord{
			{Depositor: addresses[4]},
			{Depositor: addresses[4], Recipient: addresses[5], Resolved: true},
			{Depositor: addresses[4], Recipient: common.Address{0xff}},
		},
	}

	require.Equal(t, []common.Address{
		addresses[1], addresses[4], addresses[3], addresses[5], addresses[2], addresses[0],
	}, data.RecipientAddresses())
}

func TestNewHonestActorsIgnoresZeroAddress(t *testing.T) {
	actor := common.Address{0x01}
	honest := NewHonestActors([]common.Address{{}, actor})
	require.Equal(t, HonestActors{actor: true}, honest)
}

func TestCommonGameData_UsesOutputRoots(t *testing.T) {
	for _, gameType := range outputRootGameTypes {
		gameType := gameType
		t.Run(fmt.Sprintf("GameType-%v", gameType), func(t *testing.T) {
			data := CommonGameData{
				GameMetadata: types.GameMetadata{GameType: uint32(gameType)},
			}
			require.True(t, data.UsesOutputRoots())
		})
	}

	nonOutputRootTypes := []uint32{4, 5, 9, 42982, 20013130}
	for _, gameType := range nonOutputRootTypes {
		gameType := gameType
		t.Run(fmt.Sprintf("GameType-%v", gameType), func(t *testing.T) {
			data := CommonGameData{
				GameMetadata: types.GameMetadata{GameType: gameType},
			}
			require.False(t, data.UsesOutputRoots())
		})
	}
}

func TestCommonGameData_NodeEndpointErrorCountInitialization(t *testing.T) {
	data := CommonGameData{}
	require.Equal(t, 0, data.NodeEndpointErrorCount, "NodeEndpointErrorCount should default to 0")
}

func TestCommonGameData_HasMixedAvailability(t *testing.T) {
	tests := []struct {
		name                       string
		nodeEndpointTotalCount     int
		nodeEndpointErrorCount     int
		nodeEndpointNotFoundCount  int
		nodeEndpointOutOfSyncCount int
		expected                   bool
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
		{
			name:                       "not found and out of sync is not mixed availability",
			nodeEndpointTotalCount:     2,
			nodeEndpointNotFoundCount:  1,
			nodeEndpointOutOfSyncCount: 1,
			expected:                   false,
		},
		{
			name:                       "not found, out of sync, and successful is mixed availability",
			nodeEndpointTotalCount:     3,
			nodeEndpointNotFoundCount:  1,
			nodeEndpointOutOfSyncCount: 1,
			expected:                   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := CommonGameData{
				NodeEndpointTotalCount:     test.nodeEndpointTotalCount,
				NodeEndpointErrorCount:     test.nodeEndpointErrorCount,
				NodeEndpointNotFoundCount:  test.nodeEndpointNotFoundCount,
				NodeEndpointOutOfSyncCount: test.nodeEndpointOutOfSyncCount,
			}
			result := data.HasMixedAvailability()
			require.Equal(t, test.expected, result)
		})
	}
}

func TestCommonGameData_HasMixedSafety(t *testing.T) {
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
			data := CommonGameData{
				NodeEndpointSafeCount:   test.nodeEndpointSafeCount,
				NodeEndpointUnsafeCount: test.nodeEndpointUnsafeCount,
			}
			result := data.HasMixedSafety()
			require.Equal(t, test.expected, result)
		})
	}
}
func TestAllSupportedLifecycleGameTypesAreOutputOrSuperRootType(t *testing.T) {
	for _, gameType := range types.SupportedLifecycleGameTypes {
		t.Run(gameType.String(), func(t *testing.T) {
			data := CommonGameData{
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
