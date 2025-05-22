package pipeline

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	op_service "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/stretchr/testify/require"
)

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
				return standard.DefaultHardforkScheduleForTag("")
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
			expectedOverrides: l2GenesisOverrides{
				FundDevAccounts:                          true,
				BaseFeeVaultMinimumWithdrawalAmount:      standard.VaultMinWithdrawalAmount,
				L1FeeVaultMinimumWithdrawalAmount:        standard.VaultMinWithdrawalAmount,
				SequencerFeeVaultMinimumWithdrawalAmount: standard.VaultMinWithdrawalAmount,
				BaseFeeVaultWithdrawalNetwork:            "local",
				L1FeeVaultWithdrawalNetwork:              "local",
				SequencerFeeVaultWithdrawalNetwork:       "local",
				EnableGovernance:                         false,
				GovernanceTokenOwner:                     standard.GovernanceTokenOwner,
			},
			expectedSchedule: func() *genesis.UpgradeScheduleDeployConfig {
				return standard.DefaultHardforkScheduleForTag("")
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
					"baseFeeVaultWithdrawalNetwork":            "remote",
					"l1FeeVaultWithdrawalNetwork":              "remote",
					"sequencerFeeVaultWithdrawalNetwork":       "remote",
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
				BaseFeeVaultWithdrawalNetwork:            "remote",
				L1FeeVaultWithdrawalNetwork:              "remote",
				SequencerFeeVaultWithdrawalNetwork:       "remote",
				EnableGovernance:                         true,
				GovernanceTokenOwner:                     common.HexToAddress("0x1111111111111111111111111111111111111111"),
			},
			expectedSchedule: func() *genesis.UpgradeScheduleDeployConfig {
				sched := standard.DefaultHardforkScheduleForTag("")
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
				BaseFeeVaultWithdrawalNetwork:            "remote",
				L1FeeVaultWithdrawalNetwork:              "remote",
				SequencerFeeVaultWithdrawalNetwork:       "remote",
				EnableGovernance:                         true,
				GovernanceTokenOwner:                     common.HexToAddress("0x1111111111111111111111111111111111111111"),
			},
			expectedSchedule: func() *genesis.UpgradeScheduleDeployConfig {
				sched := standard.DefaultHardforkScheduleForTag("")
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
				schedule := standard.DefaultHardforkScheduleForTag("")
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
