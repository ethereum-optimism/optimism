package pipeline

import (
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum/go-ethereum/common"
)

func Test_makeDCI_OpcmAddress(t *testing.T) {
	opcmV1Addr := common.HexToAddress("0x1111111111111111111111111111111111111111")
	opcmV2Addr := common.HexToAddress("0x2222222222222222222222222222222222222222")
	opcmV2Flag := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000010000")
	chainID := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000300")
	salt := common.HexToHash("0x1234567890123456789012345678901234567890123456789012345678901234")
	superchainConfig := common.HexToAddress("0x3333333333333333333333333333333333333333")

	baseIntent := &state.Intent{
		GlobalDeployOverrides: make(map[string]any),
	}

	baseChainIntent := &state.ChainIntent{
		ID: chainID,
		Roles: state.ChainRoles{
			L1ProxyAdminOwner: common.HexToAddress("0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"),
			SystemConfigOwner: common.HexToAddress("0xBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"),
			Batcher:           common.HexToAddress("0xCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC"),
			UnsafeBlockSigner: common.HexToAddress("0xDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD"),
			Proposer:          common.HexToAddress("0xEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEE"),
			Challenger:        common.HexToAddress("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
		},
		GasLimit: 60_000_000,
	}

	baseState := &state.State{
		Create2Salt: salt,
		SuperchainDeployment: &addresses.SuperchainContracts{
			SuperchainConfigProxy: superchainConfig,
		},
		ImplementationsDeployment: &addresses.ImplementationsContracts{
			OpcmImpl: opcmV1Addr,
		},
	}

	tests := []struct {
		name           string
		intent         *state.Intent
		thisIntent     *state.ChainIntent
		chainID        common.Hash
		st             *state.State
		expectedOpcm   common.Address
		shouldThrowErr bool
		expectedErrMsg string
	}{
		{
			name:           "default_uses_opcm_v1",
			intent:         baseIntent,
			thisIntent:     baseChainIntent,
			chainID:        chainID,
			st:             baseState,
			expectedOpcm:   opcmV1Addr,
			shouldThrowErr: false,
			expectedErrMsg: "",
		},
		{
			name: "opcm_v2_flag_enabled_with_v2_impl_uses_v2",
			intent: &state.Intent{
				GlobalDeployOverrides: map[string]any{
					"devFeatureBitmap": opcmV2Flag,
				},
			},
			thisIntent: baseChainIntent,
			chainID:    chainID,
			st: &state.State{
				Create2Salt: salt,
				SuperchainDeployment: &addresses.SuperchainContracts{
					SuperchainConfigProxy: superchainConfig,
				},
				ImplementationsDeployment: &addresses.ImplementationsContracts{
					OpcmImpl:   opcmV1Addr,
					OpcmV2Impl: opcmV2Addr,
				},
			},
			expectedOpcm:   opcmV2Addr,
			shouldThrowErr: false,
			expectedErrMsg: "",
		},
		{
			name: "opcm_v2_flag_enabled_but_v2_impl_zero_reverts",
			intent: &state.Intent{
				GlobalDeployOverrides: map[string]any{
					"devFeatureBitmap": opcmV2Flag,
				},
			},
			thisIntent: baseChainIntent,
			chainID:    chainID,
			st: &state.State{
				Create2Salt: salt,
				SuperchainDeployment: &addresses.SuperchainContracts{
					SuperchainConfigProxy: superchainConfig,
				},
				ImplementationsDeployment: &addresses.ImplementationsContracts{
					OpcmImpl:   opcmV1Addr,
					OpcmV2Impl: common.Address{}, // zero address
				},
			},
			expectedOpcm:   common.Address{},
			shouldThrowErr: true,
			expectedErrMsg: "OPCM implementation is not deployed",
		},
		{
			name:       "opcm_v2_flag_disabled_but_opcm_impl_zero_reverts",
			intent:     baseIntent,
			thisIntent: baseChainIntent,
			chainID:    chainID,
			st: &state.State{
				Create2Salt: salt,
				SuperchainDeployment: &addresses.SuperchainContracts{
					SuperchainConfigProxy: superchainConfig,
				},
				ImplementationsDeployment: &addresses.ImplementationsContracts{
					OpcmImpl:   common.Address{}, // zero address
					OpcmV2Impl: opcmV2Addr,
				},
			},
			expectedOpcm:   common.Address{},
			shouldThrowErr: true,
			expectedErrMsg: "OPCM implementation is not deployed",
		},
		{
			name: "opcm_v2_flag_not_enabled_uses_v1_even_if_v2_impl_set",
			intent: &state.Intent{
				GlobalDeployOverrides: map[string]any{
					"devFeatureBitmap": common.Hash{}, // flag not set
				},
			},
			thisIntent: baseChainIntent,
			chainID:    chainID,
			st: &state.State{
				Create2Salt: salt,
				SuperchainDeployment: &addresses.SuperchainContracts{
					SuperchainConfigProxy: superchainConfig,
				},
				ImplementationsDeployment: &addresses.ImplementationsContracts{
					OpcmImpl:   opcmV1Addr,
					OpcmV2Impl: opcmV2Addr,
				},
			},
			expectedOpcm:   opcmV1Addr,
			shouldThrowErr: false,
			expectedErrMsg: "",
		},
		{
			name: "no_dev_feature_bitmap_uses_v1",
			intent: &state.Intent{
				GlobalDeployOverrides: make(map[string]any), // no devFeatureBitmap key
			},
			thisIntent: baseChainIntent,
			chainID:    chainID,
			st: &state.State{
				Create2Salt: salt,
				SuperchainDeployment: &addresses.SuperchainContracts{
					SuperchainConfigProxy: superchainConfig,
				},
				ImplementationsDeployment: &addresses.ImplementationsContracts{
					OpcmImpl:   opcmV1Addr,
					OpcmV2Impl: opcmV2Addr,
				},
			},
			expectedOpcm:   opcmV1Addr,
			shouldThrowErr: false,
			expectedErrMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := makeDCI(tt.intent, tt.thisIntent, tt.chainID, tt.st)
			if gotErr != nil {
				if !tt.shouldThrowErr {
					t.Errorf("makeDCI() failed: %v", gotErr)
				}
				if tt.expectedErrMsg != "" && !strings.Contains(gotErr.Error(), tt.expectedErrMsg) {
					t.Errorf("makeDCI() error = %v, want error containing %q", gotErr, tt.expectedErrMsg)
				}
				return
			}
			if tt.shouldThrowErr {
				t.Fatal("makeDCI() succeeded unexpectedly")
			}
			if got.Opcm != tt.expectedOpcm {
				t.Errorf("makeDCI() Opcm = %v, want %v", got.Opcm, tt.expectedOpcm)
			}
		})
	}
}
