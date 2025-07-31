package pipeline

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
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

	dci, err := makeDCI(intent, thisIntent, chainID, st)
	if err != nil {
		return fmt.Errorf("error making deploy OP chain input: %w", err)
	}

	dco, err = opcm.DeployOPChain(env.L1ScriptHost, dci)
	if err != nil {
		return fmt.Errorf("error deploying OP chain: %w", err)
	}

	st.Chains = append(st.Chains, makeChainState(chainID, dco))

	var release string
	if intent.L1ContractsLocator.IsEmbedded() {
		release = standard.CurrentTag
	} else {
		release = "dev"
	}

	readInput := opcm.ReadImplementationAddressesInput{
		DeployOPChainOutput: dco,
		Opcm:                dci.Opcm,
		Release:             release,
	}
	impls, err := opcm.ReadImplementationAddresses(env.L1ScriptHost, readInput)
	if err != nil {
		return fmt.Errorf("failed to read implementation addresses: %w", err)
	}

	st.ImplementationsDeployment.DelayedWethImpl = impls.DelayedWETH
	st.ImplementationsDeployment.OptimismPortalImpl = impls.OptimismPortal
	st.ImplementationsDeployment.EthLockboxImpl = impls.ETHLockbox
	st.ImplementationsDeployment.SystemConfigImpl = impls.SystemConfig
	st.ImplementationsDeployment.L1CrossDomainMessengerImpl = impls.L1CrossDomainMessenger
	st.ImplementationsDeployment.L1Erc721BridgeImpl = impls.L1ERC721Bridge
	st.ImplementationsDeployment.L1StandardBridgeImpl = impls.L1StandardBridge
	st.ImplementationsDeployment.OptimismMintableErc20FactoryImpl = impls.OptimismMintableERC20Factory
	st.ImplementationsDeployment.DisputeGameFactoryImpl = impls.DisputeGameFactory
	st.ImplementationsDeployment.MipsImpl = impls.MipsSingleton
	st.ImplementationsDeployment.PreimageOracleImpl = impls.PreimageOracleSingleton

	return nil
}

func makeDCI(intent *state.Intent, thisIntent *state.ChainIntent, chainID common.Hash, st *state.State) (opcm.DeployOPChainInput, error) {
	proofParams, err := jsonutil.MergeJSON(
		state.ChainProofParams{
			DisputeGameType:         standard.DisputeGameType,
			DisputeAbsolutePrestate: standard.DisputeAbsolutePrestate,
			DisputeMaxGameDepth:     standard.DisputeMaxGameDepth,
			DisputeSplitDepth:       standard.DisputeSplitDepth,
			DisputeClockExtension:   standard.DisputeClockExtension,
			DisputeMaxClockDuration: standard.DisputeMaxClockDuration,
		},
		intent.GlobalDeployOverrides,
		thisIntent.DeployOverrides,
	)
	if err != nil {
		return opcm.DeployOPChainInput{}, fmt.Errorf("error merging proof params from overrides: %w", err)
	}

	return opcm.DeployOPChainInput{
		OpChainProxyAdminOwner:       thisIntent.Roles.L1ProxyAdminOwner,
		SystemConfigOwner:            thisIntent.Roles.SystemConfigOwner,
		Batcher:                      thisIntent.Roles.Batcher,
		UnsafeBlockSigner:            thisIntent.Roles.UnsafeBlockSigner,
		Proposer:                     thisIntent.Roles.Proposer,
		Challenger:                   thisIntent.Roles.Challenger,
		BasefeeScalar:                standard.BasefeeScalar,
		BlobBaseFeeScalar:            standard.BlobBaseFeeScalar,
		L2ChainId:                    chainID.Big(),
		Opcm:                         st.ImplementationsDeployment.OpcmImpl,
		SaltMixer:                    st.Create2Salt.String(), // passing through salt generated at state initialization
		GasLimit:                     standard.GasLimit,
		DisputeGameType:              proofParams.DisputeGameType,
		DisputeAbsolutePrestate:      proofParams.DisputeAbsolutePrestate,
		DisputeMaxGameDepth:          proofParams.DisputeMaxGameDepth,
		DisputeSplitDepth:            proofParams.DisputeSplitDepth,
		DisputeClockExtension:        proofParams.DisputeClockExtension,   // 3 hours (input in seconds)
		DisputeMaxClockDuration:      proofParams.DisputeMaxClockDuration, // 3.5 days (input in seconds)
		AllowCustomDisputeParameters: proofParams.DangerouslyAllowCustomDisputeParameters,
		OperatorFeeScalar:            thisIntent.OperatorFeeScalar,
		OperatorFeeConstant:          thisIntent.OperatorFeeConstant,
	}, nil
}

func makeChainState(chainID common.Hash, dco opcm.DeployOPChainOutput) *state.ChainState {
	opChainContracts := addresses.OpChainContracts{}
	opChainContracts.OpChainProxyAdminImpl = dco.OpChainProxyAdmin
	opChainContracts.AddressManagerImpl = dco.AddressManager
	opChainContracts.L1Erc721BridgeProxy = dco.L1ERC721BridgeProxy
	opChainContracts.SystemConfigProxy = dco.SystemConfigProxy
	opChainContracts.OptimismMintableErc20FactoryProxy = dco.OptimismMintableERC20FactoryProxy
	opChainContracts.L1StandardBridgeProxy = dco.L1StandardBridgeProxy
	opChainContracts.L1CrossDomainMessengerProxy = dco.L1CrossDomainMessengerProxy
	opChainContracts.OptimismPortalProxy = dco.OptimismPortalProxy
	opChainContracts.EthLockboxProxy = dco.ETHLockboxProxy
	opChainContracts.DisputeGameFactoryProxy = dco.DisputeGameFactoryProxy
	opChainContracts.AnchorStateRegistryProxy = dco.AnchorStateRegistryProxy
	opChainContracts.FaultDisputeGameImpl = dco.FaultDisputeGame
	opChainContracts.PermissionedDisputeGameImpl = dco.PermissionedDisputeGame
	opChainContracts.DelayedWethPermissionedGameProxy = dco.DelayedWETHPermissionedGameProxy
	opChainContracts.DelayedWethPermissionlessGameProxy = dco.DelayedWETHPermissionlessGameProxy

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
