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
	fromRecipients := common.Address{0x01}
	fromCredits := common.Address{0x02}
	fromExpected := common.Address{0x03}
	fromWithdrawals := common.Address{0x04}
	depositor := common.Address{0x05}
	resolvedRecipient := common.Address{0x06}
	unresolvedRecipient := common.Address{0x07}
	data := BondGameData{
		Recipients:      map[common.Address]bool{fromRecipients: true},
		Credits:         map[common.Address]*big.Int{fromCredits: new(big.Int)},
		ExpectedCredits: map[common.Address]*big.Int{fromExpected: new(big.Int)},
		WithdrawalRequests: map[common.Address]*contracts.WithdrawalRequest{
			fromWithdrawals: {},
		},
		Bonds: []BondRecord{
			{Depositor: depositor, Recipient: resolvedRecipient, Resolved: true},
			{Depositor: depositor, Recipient: unresolvedRecipient},
		},
	}

	require.Equal(t, []common.Address{
		fromRecipients, fromCredits, fromExpected, fromWithdrawals, depositor, resolvedRecipient,
	}, data.RecipientAddresses())
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
			name:                       "no successful endpoints - only not found and out of sync",
			nodeEndpointTotalCount:     3,
			nodeEndpointNotFoundCount:  1,
			nodeEndpointOutOfSyncCount: 2,
			expected:                   false,
		},
		{
			name:                       "mixed availability with an out of sync endpoint",
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
	expected := map[types.GameType]bool{
		types.CannonGameType:            true,
		types.PermissionedGameType:      true,
		types.SuperPermissionedGameType: false,
		types.AlphabetGameType:          true,
		types.FastGameType:              true,
		types.CannonKonaGameType:        true,
		types.SuperCannonKonaGameType:   false,
		types.ZKDisputeGameType:         false,
	}
	require.Len(t, expected, len(types.SupportedLifecycleGameTypes))
	for _, gameType := range types.SupportedLifecycleGameTypes {
		t.Run(gameType.String(), func(t *testing.T) {
			data := CommonGameData{
				GameMetadata: types.GameMetadata{
					GameType: uint32(gameType),
				},
			}
			require.Equal(t, expected[gameType], data.UsesOutputRoots())
		})
	}
}
