package pipeline

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-service/jsonutil"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"

	"github.com/ethereum/go-ethereum/common"
)

type l2GenesisOverrides struct {
	FundDevAccounts                          bool                      `json:"fundDevAccounts"`
	BaseFeeVaultMinimumWithdrawalAmount      *hexutil.Big              `json:"baseFeeVaultMinimumWithdrawalAmount"`
	L1FeeVaultMinimumWithdrawalAmount        *hexutil.Big              `json:"l1FeeVaultMinimumWithdrawalAmount"`
	SequencerFeeVaultMinimumWithdrawalAmount *hexutil.Big              `json:"sequencerFeeVaultMinimumWithdrawalAmount"`
	BaseFeeVaultWithdrawalNetwork            genesis.WithdrawalNetwork `json:"baseFeeVaultWithdrawalNetwork"`
	L1FeeVaultWithdrawalNetwork              genesis.WithdrawalNetwork `json:"l1FeeVaultWithdrawalNetwork"`
	SequencerFeeVaultWithdrawalNetwork       genesis.WithdrawalNetwork `json:"sequencerFeeVaultWithdrawalNetwork"`
	EnableGovernance                         bool                      `json:"enableGovernance"`
	GovernanceTokenOwner                     common.Address            `json:"governanceTokenOwner"`
}

func GenerateL2Genesis(pEnv *Env, intent *state.Intent, bundle ArtifactsBundle, st *state.State, chainID common.Hash) error {
	lgr := pEnv.Logger.New("stage", "generate-l2-genesis")

	thisIntent, err := intent.Chain(chainID)
	if err != nil {
		return fmt.Errorf("failed to get chain intent: %w", err)
	}

	thisChainState, err := st.Chain(chainID)
	if err != nil {
		return fmt.Errorf("failed to get chain state: %w", err)
	}

	if !shouldGenerateL2Genesis(thisChainState) {
		lgr.Info("L2 genesis generation not needed")
		return nil
	}

	lgr.Info("generating L2 genesis", "id", chainID.Hex())

	host, err := env.DefaultScriptHost(
		broadcaster.NoopBroadcaster(),
		pEnv.Logger,
		pEnv.Deployer,
		bundle.L2,
	)
	if err != nil {
		return fmt.Errorf("failed to create L2 script host: %w", err)
	}

	script, err := opcm.NewL2GenesisScript(host)
	if err != nil {
		return fmt.Errorf("failed to create L2Genesis script: %w", err)
	}

	overrides, schedule, err := calculateL2GenesisOverrides(intent, thisIntent)
	if err != nil {
		return fmt.Errorf("failed to calculate L2 genesis overrides: %w", err)
	}

	if err := script.Run(opcm.L2GenesisInput{
		L1ChainID:                                new(big.Int).SetUint64(intent.L1ChainID),
		L2ChainID:                                chainID.Big(),
		L1CrossDomainMessengerProxy:              thisChainState.L1CrossDomainMessengerProxy,
		L1StandardBridgeProxy:                    thisChainState.L1StandardBridgeProxy,
		L1ERC721BridgeProxy:                      thisChainState.L1Erc721BridgeProxy,
		OpChainProxyAdminOwner:                   thisIntent.Roles.L2ProxyAdminOwner,
		BaseFeeVaultWithdrawalNetwork:            wdNetworkToBig(overrides.BaseFeeVaultWithdrawalNetwork),
		L1FeeVaultWithdrawalNetwork:              wdNetworkToBig(overrides.L1FeeVaultWithdrawalNetwork),
		SequencerFeeVaultWithdrawalNetwork:       wdNetworkToBig(overrides.SequencerFeeVaultWithdrawalNetwork),
		SequencerFeeVaultMinimumWithdrawalAmount: overrides.SequencerFeeVaultMinimumWithdrawalAmount.ToInt(),
		BaseFeeVaultMinimumWithdrawalAmount:      overrides.BaseFeeVaultMinimumWithdrawalAmount.ToInt(),
		L1FeeVaultMinimumWithdrawalAmount:        overrides.L1FeeVaultMinimumWithdrawalAmount.ToInt(),
		BaseFeeVaultRecipient:                    thisIntent.BaseFeeVaultRecipient,
		L1FeeVaultRecipient:                      thisIntent.L1FeeVaultRecipient,
		SequencerFeeVaultRecipient:               thisIntent.SequencerFeeVaultRecipient,
		GovernanceTokenOwner:                     overrides.GovernanceTokenOwner,
		Fork:                                     big.NewInt(schedule.SolidityForkNumber(1)),
		DeployCrossL2Inbox:                       len(intent.Chains) > 1,
		EnableGovernance:                         overrides.EnableGovernance,
		FundDevAccounts:                          overrides.FundDevAccounts,
	}); err != nil {
		return fmt.Errorf("failed to call L2Genesis script: %w", err)
	}

	host.Wipe(pEnv.Deployer)

	dump, err := host.StateDump()
	if err != nil {
		return fmt.Errorf("failed to dump state: %w", err)
	}

	thisChainState.Allocs = &state.GzipData[foundry.ForgeAllocs]{
		Data: dump,
	}

	return nil
}

func calculateL2GenesisOverrides(intent *state.Intent, thisIntent *state.ChainIntent) (l2GenesisOverrides, *genesis.UpgradeScheduleDeployConfig, error) {
	schedule := standard.DefaultHardforkScheduleForTag(intent.L1ContractsLocator.Tag)

	overrides := defaultOverrides()
	// Special case for FundDevAccounts since it's both an intent value and an override.
	overrides.FundDevAccounts = intent.FundDevAccounts

	var err error
	if len(intent.GlobalDeployOverrides) > 0 {
		schedule, err = jsonutil.MergeJSON(schedule, intent.GlobalDeployOverrides)
		if err != nil {
			return l2GenesisOverrides{}, nil, fmt.Errorf("failed to merge global deploy overrides: %w", err)
		}
		overrides, err = jsonutil.MergeJSON(overrides, intent.GlobalDeployOverrides)
		if err != nil {
			return l2GenesisOverrides{}, nil, fmt.Errorf("failed to merge global deploy overrides: %w", err)
		}
	}

	if len(thisIntent.DeployOverrides) > 0 {
		schedule, err = jsonutil.MergeJSON(schedule, thisIntent.DeployOverrides)
		if err != nil {
			return l2GenesisOverrides{}, nil, fmt.Errorf("failed to merge L2 deploy overrides: %w", err)
		}
		overrides, err = jsonutil.MergeJSON(overrides, thisIntent.DeployOverrides)
		if err != nil {
			return l2GenesisOverrides{}, nil, fmt.Errorf("failed to merge global deploy overrides: %w", err)
		}
	}

	return overrides, schedule, nil
}

func shouldGenerateL2Genesis(thisChainState *state.ChainState) bool {
	return thisChainState.Allocs == nil
}

func wdNetworkToBig(wd genesis.WithdrawalNetwork) *big.Int {
	n := wd.ToUint8()
	return big.NewInt(int64(n))
}

func defaultOverrides() l2GenesisOverrides {
	return l2GenesisOverrides{
		FundDevAccounts:                          false,
		BaseFeeVaultMinimumWithdrawalAmount:      standard.VaultMinWithdrawalAmount,
		L1FeeVaultMinimumWithdrawalAmount:        standard.VaultMinWithdrawalAmount,
		SequencerFeeVaultMinimumWithdrawalAmount: standard.VaultMinWithdrawalAmount,
		BaseFeeVaultWithdrawalNetwork:            "local",
		L1FeeVaultWithdrawalNetwork:              "local",
		SequencerFeeVaultWithdrawalNetwork:       "local",
		EnableGovernance:                         false,
		GovernanceTokenOwner:                     standard.GovernanceTokenOwner,
	}
}
