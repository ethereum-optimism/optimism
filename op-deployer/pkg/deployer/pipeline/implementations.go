package pipeline

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
)

func DeployImplementations(env *Env, intent *state.Intent, st *state.State) error {
	lgr := env.Logger.New("stage", "deploy-implementations")

	if !shouldDeployImplementations(intent, st) {
		lgr.Info("implementations deployment not needed")
		return nil
	}

	lgr.Info("deploying implementations")

	var contractsRelease string
	var err error
	if intent.L1ContractsLocator.IsTag() {
		contractsRelease = intent.L1ContractsLocator.Tag
	} else {
		contractsRelease = "dev"
	}

	proofParams, err := jsonutil.MergeJSON(
		state.SuperchainProofParams{
			WithdrawalDelaySeconds:          standard.WithdrawalDelaySeconds,
			MinProposalSizeBytes:            standard.MinProposalSizeBytes,
			ChallengePeriodSeconds:          standard.ChallengePeriodSeconds,
			ProofMaturityDelaySeconds:       standard.ProofMaturityDelaySeconds,
			DisputeGameFinalityDelaySeconds: standard.DisputeGameFinalityDelaySeconds,
			MIPSVersion:                     standard.MIPSVersion,
		},
		intent.GlobalDeployOverrides,
	)
	if err != nil {
		return fmt.Errorf("error merging proof params from overrides: %w", err)
	}

	dio, err := env.Scripts.DeployImplementations.Run(
		opcm.DeployImplementationsInput{
			WithdrawalDelaySeconds:          new(big.Int).SetUint64(proofParams.WithdrawalDelaySeconds),
			MinProposalSizeBytes:            new(big.Int).SetUint64(proofParams.MinProposalSizeBytes),
			ChallengePeriodSeconds:          new(big.Int).SetUint64(proofParams.ChallengePeriodSeconds),
			ProofMaturityDelaySeconds:       new(big.Int).SetUint64(proofParams.ProofMaturityDelaySeconds),
			DisputeGameFinalityDelaySeconds: new(big.Int).SetUint64(proofParams.DisputeGameFinalityDelaySeconds),
			MipsVersion:                     new(big.Int).SetUint64(proofParams.MIPSVersion),
			L1ContractsRelease:              contractsRelease,
			SuperchainConfigProxy:           st.SuperchainDeployment.SuperchainConfigProxy,
			ProtocolVersionsProxy:           st.SuperchainDeployment.ProtocolVersionsProxy,
			SuperchainProxyAdmin:            st.SuperchainDeployment.SuperchainProxyAdminImpl,
			UpgradeController:               st.SuperchainRoles.SuperchainProxyAdminOwner,
			Challenger:                      st.SuperchainRoles.Challenger,
		},
	)
	if err != nil {
		return fmt.Errorf("error deploying implementations: %w", err)
	}

	st.ImplementationsDeployment = &addresses.ImplementationsContracts{
		OpcmImpl:                         dio.Opcm,
		OpcmGameTypeAdderImpl:            dio.OpcmGameTypeAdder,
		OpcmDeployerImpl:                 dio.OpcmDeployer,
		OpcmUpgraderImpl:                 dio.OpcmUpgrader,
		OpcmInteropMigratorImpl:          dio.OpcmInteropMigrator,
		OpcmStandardValidatorImpl:        dio.OpcmStandardValidator,
		DelayedWethImpl:                  dio.DelayedWETHImpl,
		OptimismPortalImpl:               dio.OptimismPortalImpl,
		EthLockboxImpl:                   dio.ETHLockboxImpl,
		PreimageOracleImpl:               dio.PreimageOracleSingleton,
		MipsImpl:                         dio.MipsSingleton,
		SystemConfigImpl:                 dio.SystemConfigImpl,
		L1CrossDomainMessengerImpl:       dio.L1CrossDomainMessengerImpl,
		L1Erc721BridgeImpl:               dio.L1ERC721BridgeImpl,
		L1StandardBridgeImpl:             dio.L1StandardBridgeImpl,
		OptimismMintableErc20FactoryImpl: dio.OptimismMintableERC20FactoryImpl,
		DisputeGameFactoryImpl:           dio.DisputeGameFactoryImpl,
		AnchorStateRegistryImpl:          dio.AnchorStateRegistryImpl,
	}

	return nil
}

func shouldDeployImplementations(intent *state.Intent, st *state.State) bool {
	return st.ImplementationsDeployment == nil
}
