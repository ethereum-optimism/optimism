package state

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

var (
	l2GenesisBlockBaseFeePerGas = hexutil.Big(*(big.NewInt(1000000000)))

	vaultMinWithdrawalAmount = mustHexBigFromHex("0x8ac7230489e80000")
)

func DefaultDeployConfig() genesis.DeployConfig {
	return genesis.DeployConfig{
		L2InitializationConfig: genesis.L2InitializationConfig{
			L2GenesisBlockDeployConfig: genesis.L2GenesisBlockDeployConfig{
				L2GenesisBlockGasLimit:      30_000_000,
				L2GenesisBlockBaseFeePerGas: &l2GenesisBlockBaseFeePerGas,
			},
			L2VaultsDeployConfig: genesis.L2VaultsDeployConfig{
				BaseFeeVaultWithdrawalNetwork:            "local",
				L1FeeVaultWithdrawalNetwork:              "local",
				SequencerFeeVaultWithdrawalNetwork:       "local",
				SequencerFeeVaultMinimumWithdrawalAmount: vaultMinWithdrawalAmount,
				BaseFeeVaultMinimumWithdrawalAmount:      vaultMinWithdrawalAmount,
				L1FeeVaultMinimumWithdrawalAmount:        vaultMinWithdrawalAmount,
			},
			GovernanceDeployConfig: genesis.GovernanceDeployConfig{
				EnableGovernance:      true,
				GovernanceTokenSymbol: "OP",
				GovernanceTokenName:   "Optimism",
			},
			GasPriceOracleDeployConfig: genesis.GasPriceOracleDeployConfig{
				GasPriceOracleBaseFeeScalar:     1368,
				GasPriceOracleBlobBaseFeeScalar: 810949,
			},
			EIP1559DeployConfig: genesis.EIP1559DeployConfig{
				EIP1559Denominator:       50,
				EIP1559DenominatorCanyon: 250,
				EIP1559Elasticity:        6,
			},
			UpgradeScheduleDeployConfig: genesis.UpgradeScheduleDeployConfig{
				L2GenesisRegolithTimeOffset: u64UtilPtr(0),
				L2GenesisCanyonTimeOffset:   u64UtilPtr(0),
				L2GenesisDeltaTimeOffset:    u64UtilPtr(0),
				L2GenesisEcotoneTimeOffset:  u64UtilPtr(0),
				L2GenesisFjordTimeOffset:    u64UtilPtr(0),
				L2GenesisGraniteTimeOffset:  u64UtilPtr(0),
				UseInterop:                  false,
			},
			L2CoreDeployConfig: genesis.L2CoreDeployConfig{
				L2BlockTime:               2,
				FinalizationPeriodSeconds: 12,
				MaxSequencerDrift:         600,
				SequencerWindowSize:       3600,
				ChannelTimeoutBedrock:     300,
				SystemConfigStartBlock:    0,
			},
		},
	}
}

func CombineDeployConfig(intent *Intent, chainIntent *ChainIntent, state *State, chainState *ChainState) (genesis.DeployConfig, error) {
	cfg := DefaultDeployConfig()

	var err error
	if len(intent.GlobalDeployOverrides) > 0 {
		cfg, err = mergeJSON(cfg, intent.GlobalDeployOverrides)
		if err != nil {
			return genesis.DeployConfig{}, fmt.Errorf("error merging global L2 overrides: %w", err)

		}
	}

	if len(chainIntent.DeployOverrides) > 0 {
		cfg, err = mergeJSON(cfg, chainIntent.DeployOverrides)
		if err != nil {
			return genesis.DeployConfig{}, fmt.Errorf("error merging chain L2 overrides: %w", err)
		}
	}

	cfg.L2ChainID = chainState.ID.Big().Uint64()
	cfg.L1DependenciesConfig = genesis.L1DependenciesConfig{
		L1StandardBridgeProxy:       chainState.L1StandardBridgeProxyAddress,
		L1CrossDomainMessengerProxy: chainState.L1CrossDomainMessengerProxyAddress,
		L1ERC721BridgeProxy:         chainState.L1ERC721BridgeProxyAddress,
		SystemConfigProxy:           chainState.SystemConfigProxyAddress,
		OptimismPortalProxy:         chainState.OptimismPortalProxyAddress,
		ProtocolVersionsProxy:       state.SuperchainDeployment.ProtocolVersionsProxyAddress,
	}
	cfg.OperatorDeployConfig = genesis.OperatorDeployConfig{
		BatchSenderAddress:  chainIntent.Roles.Batcher,
		P2PSequencerAddress: chainIntent.Roles.UnsafeBlockSigner,
	}
	cfg.BatchInboxAddress = calculateBatchInboxAddr(chainState.ID)
	cfg.L1ChainID = intent.L1ChainID

	return cfg, nil
}

// mergeJSON merges the provided overrides into the input struct. Fields
// must be JSON-serializable for this to work. Overrides are applied in
// order of precedence - i.e., the last overrides will override keys from
// all preceding overrides.
func mergeJSON[T any](in T, overrides ...map[string]any) (T, error) {
	var out T
	inJSON, err := json.Marshal(in)
	if err != nil {
		return out, err
	}

	var tmpMap map[string]interface{}
	if err := json.Unmarshal(inJSON, &tmpMap); err != nil {
		return out, err
	}

	for _, override := range overrides {
		for k, v := range override {
			tmpMap[k] = v
		}
	}

	inJSON, err = json.Marshal(tmpMap)
	if err != nil {
		return out, err
	}

	if err := json.Unmarshal(inJSON, &out); err != nil {
		return out, err
	}

	return out, nil
}

func mustHexBigFromHex(hex string) *hexutil.Big {
	num := hexutil.MustDecodeBig(hex)
	hexBig := hexutil.Big(*num)
	return &hexBig
}

func u64UtilPtr(in uint64) *hexutil.Uint64 {
	util := hexutil.Uint64(in)
	return &util
}

func calculateBatchInboxAddr(chainID common.Hash) common.Address {
	var out common.Address
	copy(out[1:], crypto.Keccak256(chainID[:])[:19])
	return out
}
