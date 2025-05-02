package pipeline

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/jsonutil"

	op_service "github.com/ethereum-optimism/optimism/op-service"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"

	"github.com/ethereum/go-ethereum/common"
)

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

	schedule := standard.DefaultHardforkScheduleForTag(intent.L1ContractsLocator.Tag)
	if intent.UseInterop {
		if schedule.L2GenesisIsthmusTimeOffset == nil {
			return fmt.Errorf("expecting isthmus fork to be enabled for interop deployments")
		}
		schedule.L2GenesisInteropTimeOffset = op_service.U64UtilPtr(0)
		schedule.UseInterop = true
	}

	if len(intent.GlobalDeployOverrides) > 0 {
		schedule, err = jsonutil.MergeJSON(schedule, intent.GlobalDeployOverrides)
		if err != nil {
			return fmt.Errorf("failed to merge global deploy overrides: %w", err)
		}
	}

	if len(thisIntent.DeployOverrides) > 0 {
		schedule, err = jsonutil.MergeJSON(schedule, thisIntent.DeployOverrides)
		if err != nil {
			return fmt.Errorf("failed to merge L2 deploy overrides: %w", err)
		}
	}

	if err := script.Run(opcm.L2GenesisInput{
		L1ChainID:                                new(big.Int).SetUint64(intent.L1ChainID),
		L2ChainID:                                chainID.Big(),
		L1CrossDomainMessengerProxy:              thisChainState.L1CrossDomainMessengerProxy,
		L1StandardBridgeProxy:                    thisChainState.L1StandardBridgeProxy,
		L1ERC721BridgeProxy:                      thisChainState.L1Erc721BridgeProxy,
		OpChainProxyAdminOwner:                   thisIntent.Roles.L2ProxyAdminOwner,
		BaseFeeVaultWithdrawalNetwork:            common.Big1,
		L1FeeVaultWithdrawalNetwork:              common.Big1,
		SequencerFeeVaultWithdrawalNetwork:       common.Big1,
		SequencerFeeVaultMinimumWithdrawalAmount: standard.VaultMinWithdrawalAmount.ToInt(),
		BaseFeeVaultMinimumWithdrawalAmount:      standard.VaultMinWithdrawalAmount.ToInt(),
		L1FeeVaultMinimumWithdrawalAmount:        standard.VaultMinWithdrawalAmount.ToInt(),
		BaseFeeVaultRecipient:                    thisIntent.BaseFeeVaultRecipient,
		L1FeeVaultRecipient:                      thisIntent.L1FeeVaultRecipient,
		SequencerFeeVaultRecipient:               thisIntent.SequencerFeeVaultRecipient,
		GovernanceTokenOwner:                     govTokenOwner(intent, thisIntent),
		Fork:                                     big.NewInt(schedule.SolidityForkNumber(1)),
		UseInterop:                               intent.UseInterop,
		EnableGovernance:                         isGovEnabled(intent, thisIntent),
		FundDevAccounts:                          intent.FundDevAccounts,
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

func shouldGenerateL2Genesis(thisChainState *state.ChainState) bool {
	return thisChainState.Allocs == nil
}

func govTokenOwner(intent *state.Intent, chainIntent *state.ChainIntent) common.Address {
	if !isGovEnabled(intent, chainIntent) {
		return standard.GovernanceTokenOwner
	}

	globalOverride := intent.GlobalDeployOverrides["governanceTokenOwner"]
	chainOverride := chainIntent.DeployOverrides["governanceTokenOwner"]

	var globalOverrideAddr, chainOverrideStr string
	var ok bool

	if chainOverride != nil {
		chainOverrideStr, ok = chainOverride.(string)
		if !ok || !common.IsHexAddress(chainOverrideStr) {
			return standard.GovernanceTokenOwner
		}
		return common.HexToAddress(chainOverrideStr)
	}

	if globalOverride != nil {
		globalOverrideAddr, ok = globalOverride.(string)
		if !ok || !common.IsHexAddress(globalOverrideAddr) {
			return standard.GovernanceTokenOwner
		}
		return common.HexToAddress(globalOverrideAddr)
	}

	return standard.GovernanceTokenOwner
}

func isGovEnabled(intent *state.Intent, chainIntent *state.ChainIntent) bool {
	globalOverride := intent.GlobalDeployOverrides["enableGovernance"]
	chainOverride := chainIntent.DeployOverrides["enableGovernance"]

	var globalEnabled, chainEnabled, ok bool
	if chainOverride != nil {
		chainEnabled, ok = chainOverride.(bool)
		if !ok {
			chainEnabled = false
		}
		return chainEnabled
	}

	if globalOverride != nil {
		globalEnabled, ok = globalOverride.(bool)
		if !ok {
			globalEnabled = false
		}
		return globalEnabled
	}

	return false
}
