package pipeline

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	op_service "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/stretchr/testify/require"
)

func TestBuildDevFeatureBitmap(t *testing.T) {
	interopBit := devfeatures.OptimismPortalInteropFlag
	l2cmBit := devfeatures.L2CMFlag
	interopAndL2CM := devfeatures.EnableDevFeature(interopBit, l2cmBit)

	tests := []struct {
		name        string
		useInterop  bool
		disableL2CM bool
		setOverride bool
		bitmap      any
		wantError   bool
		wantBitmap  common.Hash
	}{
		{
			name:       "no override, defaults: L2CM on, interop off",
			wantBitmap: l2cmBit,
		},
		{
			name:        "no override, DisableL2CM=true: zero bitmap",
			disableL2CM: true,
			wantBitmap:  common.Hash{},
		},
		{
			name:       "no override, UseInterop=true: interop mismatch error",
			useInterop: true,
			wantError:  true,
		},
		{
			name:        "override with L2CM only: ok",
			setOverride: true,
			bitmap:      l2cmBit,
			wantBitmap:  l2cmBit,
		},
		{
			name:        "override with interop only, DisableL2CM=false: L2CM mismatch error",
			useInterop:  true,
			setOverride: true,
			bitmap:      interopBit,
			wantError:   true,
		},
		{
			name:        "override with interop+L2CM, UseInterop=true: ok",
			useInterop:  true,
			setOverride: true,
			bitmap:      interopAndL2CM,
			wantBitmap:  interopAndL2CM,
		},
		{
			name:        "override with interop only, DisableL2CM=true, UseInterop=true: ok (kill switch)",
			useInterop:  true,
			disableL2CM: true,
			setOverride: true,
			bitmap:      interopBit,
			wantBitmap:  interopBit,
		},
		{
			name:        "override zero, DisableL2CM=true: ok (full kill switch)",
			disableL2CM: true,
			setOverride: true,
			bitmap:      common.Hash{},
			wantBitmap:  common.Hash{},
		},
		{
			name:        "override with L2CM only, DisableL2CM=true: L2CM mismatch error",
			disableL2CM: true,
			setOverride: true,
			bitmap:      l2cmBit,
			wantError:   true,
		},
		{
			name:        "override hex string with L2CM bit: ok",
			setOverride: true,
			bitmap:      l2cmBit.Hex(),
			wantBitmap:  l2cmBit,
		},
		{
			name:        "override hex string zero, DisableL2CM=true: ok",
			disableL2CM: true,
			setOverride: true,
			bitmap:      common.Hash{}.Hex(),
			wantBitmap:  common.Hash{},
		},
		{
			name:        "override with interop+L2CM, UseInterop=false: interop mismatch error",
			useInterop:  false,
			setOverride: true,
			bitmap:      interopAndL2CM,
			wantError:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overrides := map[string]any{}
			if tt.setOverride {
				overrides["devFeatureBitmap"] = tt.bitmap
			}
			intent := &state.Intent{
				L1ContractsLocator:    &artifacts.Locator{},
				UseInterop:            tt.useInterop,
				DisableL2CM:           tt.disableL2CM,
				GlobalDeployOverrides: overrides,
			}
			result, err := buildDevFeatureBitmap(intent)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantBitmap, result)
			}
		})
	}
}

func TestCalculateL2GenesisOverrides(t *testing.T) {
	testCases := []struct {
		name              string
		intent            *state.Intent
		chainIntent       *state.ChainIntent
		expectError       bool
		expectedOverrides l2GenesisOverrides
		expectedSchedule  func() *genesis.UpgradeScheduleDeployConfig
	}{
		{
			name: "basic",
			intent: &state.Intent{
				L1ContractsLocator: &artifacts.Locator{},
			},
			chainIntent:       &state.ChainIntent{},
			expectError:       false,
			expectedOverrides: defaultOverrides(),
			expectedSchedule: func() *genesis.UpgradeScheduleDeployConfig {
				return standard.DefaultHardforkSchedule()
			},
		},
		{
			name: "special case for fund dev accounts in intent",
			intent: &state.Intent{
				L1ContractsLocator: &artifacts.Locator{},
				FundDevAccounts:    true,
			},
			chainIntent: &state.ChainIntent{},
			expectError: false,
			expectedOverrides: func() l2GenesisOverrides {
				defaults := defaultOverrides()
				defaults.FundDevAccounts = true
				return defaults
			}(),
			expectedSchedule: func() *genesis.UpgradeScheduleDeployConfig {
				return standard.DefaultHardforkSchedule()
			},
		},
		{
			name: "with overrides",
			intent: &state.Intent{
				L1ContractsLocator: &artifacts.Locator{},
				GlobalDeployOverrides: map[string]any{
					"fundDevAccounts":                          true,
					"baseFeeVaultMinimumWithdrawalAmount":      "0x1234",
					"l1FeeVaultMinimumWithdrawalAmount":        "0x2345",
					"sequencerFeeVaultMinimumWithdrawalAmount": "0x3456",
					"operatorFeeVaultMinimumWithdrawalAmount":  "0x4567",
					"baseFeeVaultWithdrawalNetwork":            "remote",
					"l1FeeVaultWithdrawalNetwork":              "remote",
					"sequencerFeeVaultWithdrawalNetwork":       "remote",
					"operatorFeeVaultWithdrawalNetwork":        "remote",
					"enableGovernance":                         true,
					"governanceTokenOwner":                     "0x1111111111111111111111111111111111111111",
					"l2GenesisInteropTimeOffset":               "0x1234",
				},
			},
			chainIntent: &state.ChainIntent{},
			expectError: false,
			expectedOverrides: l2GenesisOverrides{
				FundDevAccounts:                          true,
				BaseFeeVaultMinimumWithdrawalAmount:      (*hexutil.Big)(hexutil.MustDecodeBig("0x1234")),
				L1FeeVaultMinimumWithdrawalAmount:        (*hexutil.Big)(hexutil.MustDecodeBig("0x2345")),
				SequencerFeeVaultMinimumWithdrawalAmount: (*hexutil.Big)(hexutil.MustDecodeBig("0x3456")),
				OperatorFeeVaultMinimumWithdrawalAmount:  (*hexutil.Big)(hexutil.MustDecodeBig("0x4567")),
				BaseFeeVaultWithdrawalNetwork:            "remote",
				L1FeeVaultWithdrawalNetwork:              "remote",
				SequencerFeeVaultWithdrawalNetwork:       "remote",
				OperatorFeeVaultWithdrawalNetwork:        "remote",
				EnableGovernance:                         true,
				GovernanceTokenOwner:                     common.HexToAddress("0x1111111111111111111111111111111111111111"),
			},
			expectedSchedule: func() *genesis.UpgradeScheduleDeployConfig {
				sched := standard.DefaultHardforkSchedule()
				sched.L2GenesisInteropTimeOffset = op_service.U64UtilPtr(0x1234)
				return sched
			},
		},
		{
			name: "with chain-specific overrides",
			intent: &state.Intent{
				L1ContractsLocator: &artifacts.Locator{},
				GlobalDeployOverrides: map[string]any{
					"fundDevAccounts": false,
				},
			},
			chainIntent: &state.ChainIntent{
				DeployOverrides: map[string]any{
					"fundDevAccounts":                          true,
					"baseFeeVaultMinimumWithdrawalAmount":      "0x1234",
					"l1FeeVaultMinimumWithdrawalAmount":        "0x2345",
					"sequencerFeeVaultMinimumWithdrawalAmount": "0x3456",
					"baseFeeVaultWithdrawalNetwork":            "remote",
					"l1FeeVaultWithdrawalNetwork":              "remote",
					"sequencerFeeVaultWithdrawalNetwork":       "remote",
					"operatorFeeVaultWithdrawalNetwork":        "remote",
					"enableGovernance":                         true,
					"governanceTokenOwner":                     "0x1111111111111111111111111111111111111111",
					"l2GenesisInteropTimeOffset":               "0x1234",
				},
			},
			expectError: false,
			expectedOverrides: l2GenesisOverrides{
				FundDevAccounts:                          true,
				BaseFeeVaultMinimumWithdrawalAmount:      (*hexutil.Big)(hexutil.MustDecodeBig("0x1234")),
				L1FeeVaultMinimumWithdrawalAmount:        (*hexutil.Big)(hexutil.MustDecodeBig("0x2345")),
				SequencerFeeVaultMinimumWithdrawalAmount: (*hexutil.Big)(hexutil.MustDecodeBig("0x3456")),
				OperatorFeeVaultMinimumWithdrawalAmount:  standard.VaultMinWithdrawalAmount,
				BaseFeeVaultWithdrawalNetwork:            "remote",
				L1FeeVaultWithdrawalNetwork:              "remote",
				SequencerFeeVaultWithdrawalNetwork:       "remote",
				OperatorFeeVaultWithdrawalNetwork:        "remote",
				EnableGovernance:                         true,
				GovernanceTokenOwner:                     common.HexToAddress("0x1111111111111111111111111111111111111111"),
			},
			expectedSchedule: func() *genesis.UpgradeScheduleDeployConfig {
				sched := standard.DefaultHardforkSchedule()
				sched.L2GenesisInteropTimeOffset = op_service.U64UtilPtr(0x1234)
				return sched
			},
		},
		{
			name: "interop mode",
			intent: &state.Intent{
				L1ContractsLocator: &artifacts.Locator{},
				GlobalDeployOverrides: map[string]any{
					"l2GenesisInteropTimeOffset": "0x0",
				},
			},
			chainIntent:       &state.ChainIntent{},
			expectError:       false,
			expectedOverrides: defaultOverrides(),
			expectedSchedule: func() *genesis.UpgradeScheduleDeployConfig {
				schedule := standard.DefaultHardforkSchedule()
				schedule.L2GenesisInteropTimeOffset = op_service.U64UtilPtr(0)
				return schedule
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			overrides, schedule, err := calculateL2GenesisOverrides(tc.intent, tc.chainIntent)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedOverrides, overrides)
				require.Equal(t, tc.expectedSchedule(), schedule)
			}
		})
	}
}
