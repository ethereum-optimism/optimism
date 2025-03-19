package compare

import (
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-fetcher/pkg/fetcher/fetch/script"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestCompareAddresses(t *testing.T) {
	tests := []struct {
		name               string
		combinedAddresses  map[uint64]ChainInfo
		fetchOutput        map[uint64]script.ChainConfig
		expectedDiffs      map[uint64]AddressDiffs
		expectError        bool
		expectedErrorChain uint64
	}{
		{
			name: "no differences",
			combinedAddresses: map[uint64]ChainInfo{
				10: {
					Addresses: script.Addresses{
						SystemConfigProxy:     common.HexToAddress("0x1234"),
						L1StandardBridgeProxy: common.HexToAddress("0x5678"),
					},
					Roles: script.Roles{
						SystemConfigOwner: common.HexToAddress("0xabcd"),
					},
				},
			},
			fetchOutput: map[uint64]script.ChainConfig{
				10: {
					ChainId: 10,
					Addresses: script.Addresses{
						SystemConfigProxy:     common.HexToAddress("0x1234"),
						L1StandardBridgeProxy: common.HexToAddress("0x5678"),
					},
					Roles: script.Roles{
						SystemConfigOwner: common.HexToAddress("0xabcd"),
					},
				},
			},
			expectedDiffs: map[uint64]AddressDiffs{},
			expectError:   false,
		},
		{
			name: "address differences",
			combinedAddresses: map[uint64]ChainInfo{
				10: {
					Addresses: script.Addresses{
						SystemConfigProxy:     common.HexToAddress("0x1234"),
						L1StandardBridgeProxy: common.HexToAddress("0x5678"),
					},
					Roles: script.Roles{
						SystemConfigOwner: common.HexToAddress("0xabcd"),
					},
				},
			},
			fetchOutput: map[uint64]script.ChainConfig{
				10: {
					ChainId: 10,
					Addresses: script.Addresses{
						SystemConfigProxy:     common.HexToAddress("0x9999"), // Different
						L1StandardBridgeProxy: common.HexToAddress("0x5678"),
					},
					Roles: script.Roles{
						SystemConfigOwner: common.HexToAddress("0xabcd"),
					},
				},
			},
			expectedDiffs: map[uint64]AddressDiffs{
				10: {
					Addresses: map[string]string{
						"SystemConfigProxy": common.HexToAddress("0x9999").Hex(),
					},
					Roles: map[string]string{},
				},
			},
			expectError: false,
		},
		{
			name: "role differences",
			combinedAddresses: map[uint64]ChainInfo{
				10: {
					Addresses: script.Addresses{
						SystemConfigProxy:     common.HexToAddress("0x1234"),
						L1StandardBridgeProxy: common.HexToAddress("0x5678"),
					},
					Roles: script.Roles{
						SystemConfigOwner: common.HexToAddress("0xabcd"),
						Guardian:          common.HexToAddress("0xbeef"),
					},
				},
			},
			fetchOutput: map[uint64]script.ChainConfig{
				10: {
					ChainId: 10,
					Addresses: script.Addresses{
						SystemConfigProxy:     common.HexToAddress("0x1234"),
						L1StandardBridgeProxy: common.HexToAddress("0x5678"),
					},
					Roles: script.Roles{
						SystemConfigOwner: common.HexToAddress("0xabcd"),
						Guardian:          common.HexToAddress("0xdead"), // Different
					},
				},
			},
			expectedDiffs: map[uint64]AddressDiffs{
				10: {
					Addresses: map[string]string{},
					Roles: map[string]string{
						"Guardian": common.HexToAddress("0xdead").Hex(),
					},
				},
			},
			expectError: false,
		},
		{
			name: "missing chain in CombinedAddresses",
			combinedAddresses: map[uint64]ChainInfo{
				10: {
					Addresses: script.Addresses{
						SystemConfigProxy: common.HexToAddress("0x1234"),
					},
				},
			},
			fetchOutput: map[uint64]script.ChainConfig{
				10: {
					ChainId: 10,
					Addresses: script.Addresses{
						SystemConfigProxy: common.HexToAddress("0x1234"),
					},
				},
				20: { // This chain doesn't exist in combinedAddresses
					ChainId: 20,
					Addresses: script.Addresses{
						SystemConfigProxy: common.HexToAddress("0x5678"),
					},
				},
			},
			expectedDiffs:      nil,
			expectError:        true,
			expectedErrorChain: 20,
		},
		{
			name: "multiple differences",
			combinedAddresses: map[uint64]ChainInfo{
				10: {
					Addresses: script.Addresses{
						SystemConfigProxy:     common.HexToAddress("0x1234"),
						L1StandardBridgeProxy: common.HexToAddress("0x5678"),
					},
					Roles: script.Roles{
						SystemConfigOwner: common.HexToAddress("0xabcd"),
					},
				},
				20: {
					Addresses: script.Addresses{
						SystemConfigProxy:     common.HexToAddress("0xaaaa"),
						L1StandardBridgeProxy: common.HexToAddress("0xbbbb"),
					},
					Roles: script.Roles{
						SystemConfigOwner: common.HexToAddress("0xcccc"),
					},
				},
			},
			fetchOutput: map[uint64]script.ChainConfig{
				10: {
					ChainId: 10,
					Addresses: script.Addresses{
						SystemConfigProxy:     common.HexToAddress("0x9999"), // Different
						L1StandardBridgeProxy: common.HexToAddress("0x5678"),
					},
					Roles: script.Roles{
						SystemConfigOwner: common.HexToAddress("0xabcd"),
					},
				},
				20: {
					ChainId: 20,
					Addresses: script.Addresses{
						SystemConfigProxy:     common.HexToAddress("0xaaaa"),
						L1StandardBridgeProxy: common.HexToAddress("0xdddd"), // Different
					},
					Roles: script.Roles{
						SystemConfigOwner: common.HexToAddress("0xffff"), // Different
					},
				},
			},
			expectedDiffs: map[uint64]AddressDiffs{
				10: {
					Addresses: map[string]string{
						"SystemConfigProxy": common.HexToAddress("0x9999").Hex(),
					},
					Roles: map[string]string{},
				},
				20: {
					Addresses: map[string]string{
						"L1StandardBridgeProxy": common.HexToAddress("0xdddd").Hex(),
					},
					Roles: map[string]string{
						"SystemConfigOwner": common.HexToAddress("0xffff").Hex(),
					},
				},
			},
			expectError: false,
		},
	}

	testLogger := log.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comparator := &Comparator{
				CombinedAddresses: tt.combinedAddresses,
				FetchOutput:       tt.fetchOutput,
				lgr:               testLogger,
			}

			diffs, err := comparator.CompareAddresses()
			if tt.expectError {
				require.Error(t, err)
				if tt.expectedErrorChain > 0 {
					require.Contains(t, err.Error(),
						fmt.Sprintf("chain ID %d exists in CombinedAddresses but not in FetchOutput",
							tt.expectedErrorChain))
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, len(tt.expectedDiffs), len(diffs))

				for chainID, expectedDiff := range tt.expectedDiffs {
					actualDiff, exists := diffs[chainID]
					require.True(t, exists, "Expected diff for chain ID %d", chainID)

					// Check address differences
					require.Equal(t, len(expectedDiff.Addresses), len(actualDiff.Addresses))
					for addrName, expectedAddrValue := range expectedDiff.Addresses {
						actualAddrValue, exists := actualDiff.Addresses[addrName]
						require.True(t, exists, "Expected address diff for %s", addrName)
						require.Equal(t, expectedAddrValue, actualAddrValue)
					}

					// Check role differences
					require.Equal(t, len(expectedDiff.Roles), len(actualDiff.Roles))
					for roleName, expectedRoleValue := range expectedDiff.Roles {
						actualRoleValue, exists := actualDiff.Roles[roleName]
						require.True(t, exists, "Expected role diff for %s", roleName)
						require.Equal(t, expectedRoleValue, actualRoleValue)
					}
				}
			}
		})
	}
}
