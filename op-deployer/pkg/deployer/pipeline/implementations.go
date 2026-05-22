package pipeline

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
)

func DeployImplementations(env *Env, intent *state.Intent, st *state.State) error {
	lgr := env.Logger.New("stage", "deploy-implementations")

	if !shouldDeployImplementations(intent, st) {
		lgr.Info("implementations deployment not needed")
		return nil
	}

	lgr.Info("deploying implementations")

	input, err := evaluateLegacyInputMapping(env, "deploy-implementations.input.yaml", opcm.StaticInputSources{
		"intent": intent,
		"state":  st,
	})
	if err != nil {
		return fmt.Errorf("failed to build DeployImplementations input: %w", err)
	}

	var dio opcm.DeployImplementationsOutput

	if env.UseForge {
		lgr.Info("using Forge for DeployImplementations")
		forgeEnv := &opcm.ForgeEnv{
			Client:     env.ForgeClient,
			Context:    env.Context,
			L1RPCUrl:   env.L1RPCUrl,
			PrivateKey: env.PrivateKey,
		}
		dio, err = opcm.DeployImplementationsViaForge(forgeEnv, input)
		if err != nil {
			return err
		}
	} else {
		dio, err = env.Scripts.DeployImplementations.Run(input)
		if err != nil {
			return fmt.Errorf("error deploying implementations: %w", err)
		}
	}

	st.ImplementationsDeployment = &addresses.ImplementationsContracts{
		OpcmStandardValidatorImpl:        dio.Address("opcmStandardValidator"),
		OpcmUtilsImpl:                    dio.Address("opcmUtils"),
		OpcmMigratorImpl:                 dio.Address("opcmMigrator"),
		OpcmV2Impl:                       dio.Address("opcmV2"),
		OpcmContainerImpl:                dio.Address("opcmContainer"),
		DelayedWethImpl:                  dio.Address("delayedWETHImpl"),
		OptimismPortalImpl:               dio.Address("optimismPortalImpl"),
		EthLockboxImpl:                   dio.Address("ethLockboxImpl"),
		PreimageOracleImpl:               dio.Address("preimageOracleSingleton"),
		MipsImpl:                         dio.Address("mipsSingleton"),
		SystemConfigImpl:                 dio.Address("systemConfigImpl"),
		L1CrossDomainMessengerImpl:       dio.Address("l1CrossDomainMessengerImpl"),
		L1Erc721BridgeImpl:               dio.Address("l1ERC721BridgeImpl"),
		L1StandardBridgeImpl:             dio.Address("l1StandardBridgeImpl"),
		OptimismMintableErc20FactoryImpl: dio.Address("optimismMintableERC20FactoryImpl"),
		DisputeGameFactoryImpl:           dio.Address("disputeGameFactoryImpl"),
		AnchorStateRegistryImpl:          dio.Address("anchorStateRegistryImpl"),
		FaultDisputeGameImpl:             dio.Address("faultDisputeGameImpl"),
		PermissionedDisputeGameImpl:      dio.Address("permissionedDisputeGameImpl"),
		ZkDisputeGameImpl:                dio.Address("zkDisputeGameImpl"),
		StorageSetterImpl:                dio.Address("storageSetterImpl"),
		SuperFaultDisputeGameImpl:        dio.Address("superFaultDisputeGameImpl"),
		SuperPermissionedDisputeGameImpl: dio.Address("superPermissionedDisputeGameImpl"),
	}

	return nil
}

func shouldDeployImplementations(intent *state.Intent, st *state.State) bool {
	return st.ImplementationsDeployment == nil
}
