package pipeline

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	"github.com/ethereum/go-ethereum/common"

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

	proofParams, err := jsonutil.MergeJSON(
		state.SuperchainProofParams{
			WithdrawalDelaySeconds:          standard.WithdrawalDelaySeconds,
			MinProposalSizeBytes:            standard.MinProposalSizeBytes,
			ChallengePeriodSeconds:          standard.ChallengePeriodSeconds,
			ProofMaturityDelaySeconds:       standard.ProofMaturityDelaySeconds,
			DisputeGameFinalityDelaySeconds: standard.DisputeGameFinalityDelaySeconds,
			DisputeMaxGameDepth:             standard.DisputeMaxGameDepth,
			DisputeSplitDepth:               standard.DisputeSplitDepth,
			DisputeClockExtension:           standard.DisputeClockExtension,
			DisputeMaxClockDuration:         standard.DisputeMaxClockDuration,
			MIPSVersion:                     standard.MIPSVersion,
			DevFeatureBitmap:                common.Hash{},
		},
		intent.GlobalDeployOverrides,
	)
	if err != nil {
		return fmt.Errorf("error merging proof params from overrides: %w", err)
	}

	var dio opcm.DeployImplementationsOutput
	input := opcm.DeployImplementationsInput{
		WithdrawalDelaySeconds:          new(big.Int).SetUint64(proofParams.WithdrawalDelaySeconds),
		MinProposalSizeBytes:            new(big.Int).SetUint64(proofParams.MinProposalSizeBytes),
		ChallengePeriodSeconds:          new(big.Int).SetUint64(proofParams.ChallengePeriodSeconds),
		ProofMaturityDelaySeconds:       new(big.Int).SetUint64(proofParams.ProofMaturityDelaySeconds),
		DisputeGameFinalityDelaySeconds: new(big.Int).SetUint64(proofParams.DisputeGameFinalityDelaySeconds),
		MipsVersion:                     new(big.Int).SetUint64(proofParams.MIPSVersion),
		DevFeatureBitmap:                proofParams.DevFeatureBitmap,
		FaultGameV2MaxGameDepth:         new(big.Int).SetUint64(proofParams.DisputeMaxGameDepth),
		FaultGameV2SplitDepth:           new(big.Int).SetUint64(proofParams.DisputeSplitDepth),
		FaultGameV2ClockExtension:       new(big.Int).SetUint64(proofParams.DisputeClockExtension),
		FaultGameV2MaxClockDuration:     new(big.Int).SetUint64(proofParams.DisputeMaxClockDuration),
		SuperchainConfigProxy:           st.SuperchainDeployment.SuperchainConfigProxy,
		ProtocolVersionsProxy:           st.SuperchainDeployment.ProtocolVersionsProxy,
		SuperchainProxyAdmin:            st.SuperchainDeployment.SuperchainProxyAdminImpl,
		L1ProxyAdminOwner:               st.SuperchainRoles.SuperchainProxyAdminOwner,
		Challenger:                      st.SuperchainRoles.Challenger,
	}

	if env.UseForge {
		if env.ForgeClient == nil {
			return fmt.Errorf("Forge client is nil but UseForge is enabled")
		}
		if env.Context == nil {
			env.Context = context.Background()
		}
		lgr.Info("using Forge for DeployImplementations")
		forgeCaller := opcm.NewDeployImplementationsForgeCaller(env.ForgeClient)
		forgeOpts := []string{
			"--rpc-url", env.L1RPCUrl,
			"--broadcast",
			"--private-key", env.PrivateKey,
		}
		dio, _, err = forgeCaller(env.Context, input, forgeOpts...)
		if err != nil {
			return fmt.Errorf("failed to deploy implementations with Forge: %w", err)
		}
	} else {
		dio, err = env.Scripts.DeployImplementations.Run(input)
		if err != nil {
			return fmt.Errorf("error deploying implementations: %w", err)
		}
	}

	st.ImplementationsDeployment = &addresses.ImplementationsContracts{
		OpcmImpl:                         dio.Opcm,
		OpcmGameTypeAdderImpl:            dio.OpcmGameTypeAdder,
		OpcmDeployerImpl:                 dio.OpcmDeployer,
		OpcmUpgraderImpl:                 dio.OpcmUpgrader,
		OpcmInteropMigratorImpl:          dio.OpcmInteropMigrator,
		OpcmStandardValidatorImpl:        dio.OpcmStandardValidator,
		OpcmV2Impl:                       dio.OpcmV2,
		OpcmContainerImpl:                dio.OpcmContainer,
		DelayedWethImpl:                  dio.DelayedWETHImpl,
		OptimismPortalImpl:               dio.OptimismPortalImpl,
		OptimismPortalInteropImpl:        dio.OptimismPortalInteropImpl,
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
		FaultDisputeGameImpl:             dio.FaultDisputeGameImpl,
		PermissionedDisputeGameImpl:      dio.PermissionedDisputeGameImpl,
		StorageSetterImpl:                dio.StorageSetterImpl,
	}

	return nil
}

func shouldDeployImplementations(intent *state.Intent, st *state.State) bool {
	return st.ImplementationsDeployment == nil
}
