package opcm

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestL2Genesis2Differential(t *testing.T) {
	tests := []struct {
		name    string
		mutator func(*genesis.L2InitializationConfig, *L2Genesis2Input)
	}{
		{
			name:    "no changes",
			mutator: func(initCfg *genesis.L2InitializationConfig, input *L2Genesis2Input) {},
		},
		{
			"interop",
			func(initCfg *genesis.L2InitializationConfig, input *L2Genesis2Input) {
				initCfg.UseInterop = true
				input.UseInterop = true
			},
		},
		{
			"no dev account",
			func(initCfg *genesis.L2InitializationConfig, input *L2Genesis2Input) {
				initCfg.DevDeployConfig.FundDevAccounts = false
				input.FundDevAccounts = false
			},
		},
		{
			"no governance",
			func(initCfg *genesis.L2InitializationConfig, input *L2Genesis2Input) {
				initCfg.GovernanceDeployConfig.EnableGovernance = false
				input.EnableGovernance = false
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			initCfg := genesis.L2InitializationConfig{
				DevDeployConfig: genesis.DevDeployConfig{FundDevAccounts: true},
				OwnershipDeployConfig: genesis.OwnershipDeployConfig{
					ProxyAdminOwner:  common.Address{'O'},
					FinalSystemOwner: common.Address{'F'},
				},
				GovernanceDeployConfig: genesis.GovernanceDeployConfig{
					EnableGovernance:      true,
					GovernanceTokenSymbol: "OP",
					GovernanceTokenName:   "Optimism",
					GovernanceTokenOwner:  common.HexToAddress("0xDeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAdDEad"),
				},
				L2CoreDeployConfig: genesis.L2CoreDeployConfig{
					L1ChainID: 999,
					L2ChainID: 1000,
				},
				L2VaultsDeployConfig: genesis.L2VaultsDeployConfig{
					BaseFeeVaultRecipient:                    common.Address{'B'},
					L1FeeVaultRecipient:                      common.Address{'L'},
					SequencerFeeVaultRecipient:               common.Address{'S'},
					BaseFeeVaultMinimumWithdrawalAmount:      (*hexutil.Big)(big.NewInt(1)),
					L1FeeVaultMinimumWithdrawalAmount:        (*hexutil.Big)(big.NewInt(1)),
					SequencerFeeVaultMinimumWithdrawalAmount: (*hexutil.Big)(big.NewInt(1)),
					BaseFeeVaultWithdrawalNetwork:            genesis.WithdrawalNetwork("local"),
					L1FeeVaultWithdrawalNetwork:              genesis.WithdrawalNetwork("local"),
					SequencerFeeVaultWithdrawalNetwork:       genesis.WithdrawalNetwork("local"),
				},
			}
			initCfg.UpgradeScheduleDeployConfig.ActivateForkAtGenesis(rollup.Isthmus)

			l1Deps := L1Deployments{
				L1CrossDomainMessengerProxy: common.Address{'M'},
				L1StandardBridgeProxy:       common.Address{'B'},
				L1ERC721BridgeProxy:         common.Address{'E'},
			}

			newInput := L2Genesis2Input{
				L1ChainID:                                new(big.Int).SetUint64(initCfg.L1ChainID),
				L2ChainID:                                new(big.Int).SetUint64(initCfg.L2ChainID),
				L1CrossDomainMessengerProxy:              l1Deps.L1CrossDomainMessengerProxy,
				L1StandardBridgeProxy:                    l1Deps.L1StandardBridgeProxy,
				L1ERC721BridgeProxy:                      l1Deps.L1ERC721BridgeProxy,
				L2ProxyAdminOwner:                        initCfg.OwnershipDeployConfig.ProxyAdminOwner,
				SequencerFeeVaultRecipient:               initCfg.L2VaultsDeployConfig.SequencerFeeVaultRecipient,
				SequencerFeeVaultMinimumWithdrawalAmount: (*big.Int)(initCfg.L2VaultsDeployConfig.SequencerFeeVaultMinimumWithdrawalAmount),
				SequencerFeeVaultWithdrawalNetwork:       big.NewInt(int64(initCfg.L2VaultsDeployConfig.SequencerFeeVaultWithdrawalNetwork.ToUint8())),
				BaseFeeVaultRecipient:                    initCfg.L2VaultsDeployConfig.BaseFeeVaultRecipient,
				BaseFeeVaultMinimumWithdrawalAmount:      (*big.Int)(initCfg.L2VaultsDeployConfig.BaseFeeVaultMinimumWithdrawalAmount),
				BaseFeeVaultWithdrawalNetwork:            big.NewInt(int64(initCfg.L2VaultsDeployConfig.BaseFeeVaultWithdrawalNetwork.ToUint8())),
				L1FeeVaultRecipient:                      initCfg.L2VaultsDeployConfig.L1FeeVaultRecipient,
				L1FeeVaultMinimumWithdrawalAmount:        (*big.Int)(initCfg.L2VaultsDeployConfig.L1FeeVaultMinimumWithdrawalAmount),
				L1FeeVaultWithdrawalNetwork:              big.NewInt(int64(initCfg.L2VaultsDeployConfig.L1FeeVaultWithdrawalNetwork.ToUint8())),
				GovernanceTokenOwner:                     initCfg.GovernanceDeployConfig.GovernanceTokenOwner,
				Fork:                                     big.NewInt(6), // corresponds to isthmus
				UseInterop:                               false,
				EnableGovernance:                         true,
				FundDevAccounts:                          true,
			}

			tt.mutator(&initCfg, &newInput)

			hostOld := createTestHost(t)
			require.NoError(t, L2Genesis(hostOld, &L2GenesisInput{
				L1Deployments: l1Deps,
				L2Config:      initCfg,
			}))

			hostNew := createTestHost(t)
			script, err := NewL2Genesis2Script(hostNew)
			require.NoError(t, err)

			require.NoError(t, script.Run(newInput))

			dumpOld, err := hostOld.StateDump()
			require.NoError(t, err)
			dumpNew, err := hostNew.StateDump()
			require.NoError(t, err)

			dumpOldJSON, err := json.MarshalIndent(dumpOld, "", "  ")
			require.NoError(t, err)
			dumpNewJSON, err := json.MarshalIndent(dumpNew, "", "  ")
			require.NoError(t, err)

			require.JSONEq(t, string(dumpOldJSON), string(dumpNewJSON))
		})
	}
}
