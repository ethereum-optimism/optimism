package validate

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/devnet-sdk/proofs/prestate"
	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestGetAbsolutePrestate(t *testing.T) {
	tests := []struct {
		name        string
		globalState *state.State
		chainID     common.Hash
		expected    common.Hash
		description string
	}{
		{
			name: "from prestate manifest",
			globalState: &state.State{
				PrestateManifest: func() *prestate.PrestateManifest {
					m := prestate.PrestateManifest{
						"1": "0x1234567890123456789012345678901234567890123456789012345678901234",
					}
					return &m
				}(),
			},
			chainID:     common.BigToHash(common.Big1),
			expected:    common.HexToHash("0x1234567890123456789012345678901234567890123456789012345678901234"),
			description: "Should return prestate from manifest when available",
		},
		{
			name: "from deploy overrides",
			globalState: &state.State{
				AppliedIntent: &state.Intent{
					Chains: []*state.ChainIntent{
						{
							ID: common.BigToHash(common.Big1),
							DeployOverrides: map[string]interface{}{
								"faultGameAbsolutePrestate": "0xabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdef",
							},
						},
					},
				},
			},
			chainID:     common.BigToHash(common.Big1),
			expected:    common.HexToHash("0xabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdef"),
			description: "Should return prestate from deploy overrides when manifest not available",
		},
		{
			name: "fallback to standard prestate",
			globalState: &state.State{
				AppliedIntent: &state.Intent{
					Chains: []*state.ChainIntent{
						{
							ID:              common.BigToHash(common.Big1),
							DeployOverrides: map[string]interface{}{},
						},
					},
				},
			},
			chainID:     common.BigToHash(common.Big1),
			expected:    standard.DisputeAbsolutePrestate,
			description: "Should fallback to standard prestate when not found elsewhere",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := log.New()
			result := GetAbsolutePrestate(tt.globalState, tt.chainID, logger)
			require.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestBuildValidatorConfigFromState(t *testing.T) {
	tests := []struct {
		name        string
		globalState *state.State
		chainState  *state.ChainState
		chainID     common.Hash
		expectError bool
		description string
	}{
		{
			name: "valid config with all required fields",
			globalState: &state.State{
				AppliedIntent: &state.Intent{
					Chains: []*state.ChainIntent{
						{
							ID: common.BigToHash(common.Big1),
							Roles: state.ChainRoles{
								Proposer: common.Address{0x01},
							},
						},
					},
				},
			},
			chainState: &state.ChainState{
				OpChainContracts: addresses.OpChainContracts{
					OpChainCoreContracts: addresses.OpChainCoreContracts{
						OpChainProxyAdminImpl: common.Address{0x02},
						SystemConfigProxy:     common.Address{0x03},
					},
				},
			},
			chainID:     common.BigToHash(common.Big1),
			expectError: false,
			description: "Should build config successfully with all required fields",
		},
		{
			name: "missing proxy admin",
			globalState: &state.State{
				AppliedIntent: &state.Intent{
					Chains: []*state.ChainIntent{
						{
							ID: common.BigToHash(common.Big1),
						},
					},
				},
			},
			chainState: &state.ChainState{
				OpChainContracts: addresses.OpChainContracts{
					OpChainCoreContracts: addresses.OpChainCoreContracts{
						OpChainProxyAdminImpl: common.Address{},
						SystemConfigProxy:     common.Address{0x03},
					},
				},
			},
			chainID:     common.BigToHash(common.Big1),
			expectError: true,
			description: "Should error when proxy admin is missing",
		},
		{
			name: "missing system config",
			globalState: &state.State{
				AppliedIntent: &state.Intent{
					Chains: []*state.ChainIntent{
						{
							ID: common.BigToHash(common.Big1),
						},
					},
				},
			},
			chainState: &state.ChainState{
				OpChainContracts: addresses.OpChainContracts{
					OpChainCoreContracts: addresses.OpChainCoreContracts{
						OpChainProxyAdminImpl: common.Address{0x02},
						SystemConfigProxy:     common.Address{},
					},
				},
			},
			chainID:     common.BigToHash(common.Big1),
			expectError: true,
			description: "Should error when system config is missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			logger := log.New()
			l1RPCURL := "http://localhost:99999"

			cfg, err := BuildValidatorConfigFromState(ctx, logger, tt.globalState, tt.chainState, tt.chainID, l1RPCURL)

			if tt.expectError {
				require.Error(t, err, tt.description)
				require.Nil(t, cfg)
			} else {
				require.NoError(t, err, tt.description)
				require.NotNil(t, cfg)
				require.Equal(t, tt.chainState.OpChainContracts.OpChainProxyAdminImpl, cfg.ProxyAdmin)
				require.Equal(t, tt.chainState.OpChainContracts.SystemConfigProxy, cfg.SystemConfig)
				require.Equal(t, tt.chainID.Big(), cfg.L2ChainID)
				// Proposer should be set from chain intent
				require.Equal(t, common.Address{0x01}, cfg.Proposer)
			}
		})
	}
}

func TestDetectValidatorVersion(t *testing.T) {
	tests := []struct {
		name          string
		validateFlag  string
		appliedIntent *state.Intent
		expected      string
		description   string
	}{
		{
			name:         "auto with tag locator",
			validateFlag: "auto",
			appliedIntent: func() *state.Intent {
				loc, _ := artifacts.NewLocatorFromURL("tag://op-contracts/v5.0.0")
				return &state.Intent{
					L1ContractsLocator: loc,
				}
			}(),
			expected:    "op-contracts/v5.0.0",
			description: "Should detect version from tag:// locator",
		},
		{
			name:         "auto with non-tag locator",
			validateFlag: "auto",
			appliedIntent: func() *state.Intent {
				loc, _ := artifacts.NewLocatorFromURL("http://example.com/artifacts")
				return &state.Intent{
					L1ContractsLocator: loc,
				}
			}(),
			expected:    standard.CurrentTag,
			description: "Should fallback to current tag for non-tag locator",
		},
		{
			name:         "auto with nil locator",
			validateFlag: "auto",
			appliedIntent: &state.Intent{
				L1ContractsLocator: nil,
			},
			expected:    standard.CurrentTag,
			description: "Should fallback to current tag when locator is nil",
		},
		{
			name:          "explicit version without prefix",
			validateFlag:  "v2.0.0",
			appliedIntent: &state.Intent{},
			expected:      "op-contracts/v2.0.0",
			description:   "Should add op-contracts/ prefix when missing",
		},
		{
			name:          "explicit version with prefix",
			validateFlag:  "op-contracts/v3.0.0",
			appliedIntent: &state.Intent{},
			expected:      "op-contracts/v3.0.0",
			description:   "Should use version as-is when prefix already present",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := log.New()
			result := DetectValidatorVersion(tt.validateFlag, tt.appliedIntent, logger)
			require.Equal(t, tt.expected, result, tt.description)
		})
	}
}
