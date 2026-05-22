package pipeline

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum/go-ethereum/common"
)

func DeployOPChain(env *Env, intent *state.Intent, st *state.State, chainID common.Hash) error {
	lgr := env.Logger.New("stage", "deploy-opchain")

	if !shouldDeployOPChain(st, chainID) {
		lgr.Info("opchain deployment not needed")
		return nil
	}

	thisIntent, err := intent.Chain(chainID)
	if err != nil {
		return fmt.Errorf("failed to get chain intent: %w", err)
	}

	var dco opcm.DeployOPChainOutput
	lgr.Info("deploying OP chain using local allocs", "id", chainID.Hex())

	dci, err := makeDCIWithEnv(env, intent, thisIntent, chainID, st)
	if err != nil {
		return fmt.Errorf("error making deploy OP chain input: %w", err)
	}

	if env.UseForge {
		lgr.Info("using Forge for DeployOPChain")
		forgeEnv := &opcm.ForgeEnv{
			Client:     env.ForgeClient,
			Context:    env.Context,
			L1RPCUrl:   env.L1RPCUrl,
			PrivateKey: env.PrivateKey,
		}
		dco, err = opcm.DeployOPChainViaForge(forgeEnv, dci)
		if err != nil {
			return err
		}
	} else {
		dco, err = env.Scripts.DeployOPChain.Run(dci)
		if err != nil {
			return fmt.Errorf("error deploying OP chain: %w", err)
		}
	}

	readInput, err := evaluateLegacyInputMapping(env, "read-implementation-addresses.input.yaml", opcm.StaticInputSources{
		"deployOPChainInput":  dci,
		"deployOPChainOutput": dco,
	})
	if err != nil {
		return fmt.Errorf("failed to build ReadImplementationAddresses input: %w", err)
	}

	var impls opcm.ReadImplementationAddressesOutput
	if env.UseForge {
		lgr.Info("using Forge for ReadImplementationAddresses")
		forgeEnv := &opcm.ForgeEnv{
			Client:   env.ForgeClient,
			Context:  env.Context,
			L1RPCUrl: env.L1RPCUrl,
		}
		impls, err = opcm.ReadImplementationAddressesViaForge(forgeEnv, readInput)
		if err != nil {
			return err
		}
	} else {
		readImplementations, err := opcm.NewReadImplementationAddressesScript(env.L1ScriptHost)
		if err != nil {
			return fmt.Errorf("failed to load ReadImplementationAddresses script: %w", err)
		}

		impls, err = readImplementations.Run(readInput)
		if err != nil {
			return fmt.Errorf("failed to run ReadImplementationAddresses script: %w", err)
		}
	}

	st.Chains = append(st.Chains, makeChainState(chainID, impls, dco))

	st.ImplementationsDeployment.DelayedWethImpl = impls.Address("delayedWETH")
	st.ImplementationsDeployment.OptimismPortalImpl = impls.Address("optimismPortal")
	st.ImplementationsDeployment.EthLockboxImpl = impls.Address("ethLockbox")
	st.ImplementationsDeployment.SystemConfigImpl = impls.Address("systemConfig")
	st.ImplementationsDeployment.AnchorStateRegistryImpl = impls.Address("anchorStateRegistry")
	st.ImplementationsDeployment.L1CrossDomainMessengerImpl = impls.Address("l1CrossDomainMessenger")
	st.ImplementationsDeployment.L1Erc721BridgeImpl = impls.Address("l1ERC721Bridge")
	st.ImplementationsDeployment.L1StandardBridgeImpl = impls.Address("l1StandardBridge")
	st.ImplementationsDeployment.OptimismMintableErc20FactoryImpl = impls.Address("optimismMintableERC20Factory")
	st.ImplementationsDeployment.DisputeGameFactoryImpl = impls.Address("disputeGameFactory")
	st.ImplementationsDeployment.MipsImpl = impls.Address("mipsSingleton")
	st.ImplementationsDeployment.PreimageOracleImpl = impls.Address("preimageOracleSingleton")
	st.ImplementationsDeployment.FaultDisputeGameImpl = impls.Address("faultDisputeGame")
	st.ImplementationsDeployment.PermissionedDisputeGameImpl = impls.Address("permissionedDisputeGame")
	st.ImplementationsDeployment.ZkDisputeGameImpl = impls.Address("zkDisputeGame")
	st.ImplementationsDeployment.OpcmStandardValidatorImpl = impls.Address("opcmStandardValidator")
	st.ImplementationsDeployment.SuperFaultDisputeGameImpl = impls.Address("superFaultDisputeGame")
	st.ImplementationsDeployment.SuperPermissionedDisputeGameImpl = impls.Address("superPermissionedDisputeGame")

	return nil
}

func makeDCI(intent *state.Intent, thisIntent *state.ChainIntent, chainID common.Hash, st *state.State) (opcm.DeployOPChainInput, error) {
	return makeDCIWithEnv(nil, intent, thisIntent, chainID, st)
}

func makeDCIWithEnv(env *Env, intent *state.Intent, thisIntent *state.ChainIntent, chainID common.Hash, st *state.State) (opcm.DeployOPChainInput, error) {
	input, err := evaluateLegacyInputMapping(env, "deploy-op-chain.input.yaml", opcm.StaticInputSources{
		"chain":   thisIntent,
		"chainID": chainID,
		"intent":  intent,
		"state":   st,
	})
	if err != nil {
		return nil, err
	}
	if input.Address("opcm") == (common.Address{}) {
		return nil, fmt.Errorf("OPCM implementation is not deployed")
	}
	return input, nil
}

func makeChainState(chainID common.Hash, impls opcm.ReadImplementationAddressesOutput, dco opcm.DeployOPChainOutput) *state.ChainState {
	opChainContracts := addresses.OpChainContracts{}
	opChainContracts.OpChainProxyAdminImpl = dco.Address("opChainProxyAdmin")
	opChainContracts.AddressManagerImpl = dco.Address("addressManager")
	opChainContracts.L1Erc721BridgeProxy = dco.Address("l1ERC721BridgeProxy")
	opChainContracts.SystemConfigProxy = dco.Address("systemConfigProxy")
	opChainContracts.OptimismMintableErc20FactoryProxy = dco.Address("optimismMintableERC20FactoryProxy")
	opChainContracts.L1StandardBridgeProxy = dco.Address("l1StandardBridgeProxy")
	opChainContracts.L1CrossDomainMessengerProxy = dco.Address("l1CrossDomainMessengerProxy")
	opChainContracts.OptimismPortalProxy = dco.Address("optimismPortalProxy")
	opChainContracts.EthLockboxProxy = dco.Address("ethLockboxProxy")
	opChainContracts.DisputeGameFactoryProxy = dco.Address("disputeGameFactoryProxy")
	opChainContracts.AnchorStateRegistryProxy = dco.Address("anchorStateRegistryProxy")
	opChainContracts.FaultDisputeGameImpl = dco.Address("faultDisputeGame")
	opChainContracts.PermissionedDisputeGameImpl = dco.Address("permissionedDisputeGame")
	opChainContracts.DelayedWethPermissionedGameProxy = dco.Address("delayedWETHPermissionedGameProxy")
	opChainContracts.DelayedWethPermissionlessGameProxy = dco.Address("delayedWETHPermissionlessGameProxy")

	if permissionedDisputeGame := impls.Address("permissionedDisputeGame"); permissionedDisputeGame != (common.Address{}) {
		opChainContracts.PermissionedDisputeGameImpl = permissionedDisputeGame
	}
	if faultDisputeGame := impls.Address("faultDisputeGame"); faultDisputeGame != (common.Address{}) {
		opChainContracts.FaultDisputeGameImpl = faultDisputeGame
	}

	return &state.ChainState{
		ID:               chainID,
		OpChainContracts: opChainContracts,
	}
}

func shouldDeployOPChain(st *state.State, chainID common.Hash) bool {
	for _, chain := range st.Chains {
		if chain.ID == chainID {
			return false
		}
	}

	return true
}
