package pipeline

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
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

	// We make sure that the deployer and OPCM are the same as the ones used in the dry-run, if any.
	// Skip when the deployer is the placeholder, which means non-live strategies are being used.
	if env.Deployer != standard.PlaceholderAddress {
		if err := st.CheckL1PredictInputs(env.Deployer, dci.Opcm); err != nil {
			return err
		}
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

	readInput := opcm.ReadImplementationAddressesInput{
		AddressManager:                    dco.AddressManager,
		L1ERC721BridgeProxy:               dco.L1ERC721BridgeProxy,
		SystemConfigProxy:                 dco.SystemConfigProxy,
		OptimismMintableERC20FactoryProxy: dco.OptimismMintableERC20FactoryProxy,
		L1StandardBridgeProxy:             dco.L1StandardBridgeProxy,
		OptimismPortalProxy:               dco.OptimismPortalProxy,
		DisputeGameFactoryProxy:           dco.DisputeGameFactoryProxy,
		Opcm:                              dci.Opcm,
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

	st.SetChainContracts(chainID, chainContractsForDeploy(impls, dco), true)

	st.ImplementationsDeployment.DelayedWethImpl = impls.DelayedWETH
	st.ImplementationsDeployment.OptimismPortalImpl = impls.OptimismPortal
	st.ImplementationsDeployment.EthLockboxImpl = impls.EthLockbox
	st.ImplementationsDeployment.SystemConfigImpl = impls.SystemConfig
	st.ImplementationsDeployment.AnchorStateRegistryImpl = impls.AnchorStateRegistry
	st.ImplementationsDeployment.L1CrossDomainMessengerImpl = impls.L1CrossDomainMessenger
	st.ImplementationsDeployment.L1Erc721BridgeImpl = impls.L1ERC721Bridge
	st.ImplementationsDeployment.L1StandardBridgeImpl = impls.L1StandardBridge
	st.ImplementationsDeployment.OptimismMintableErc20FactoryImpl = impls.OptimismMintableERC20Factory
	st.ImplementationsDeployment.DisputeGameFactoryImpl = impls.DisputeGameFactory
	st.ImplementationsDeployment.MipsImpl = impls.MipsSingleton
	st.ImplementationsDeployment.PreimageOracleImpl = impls.PreimageOracleSingleton
	st.ImplementationsDeployment.FaultDisputeGameImpl = impls.FaultDisputeGame
	st.ImplementationsDeployment.PermissionedDisputeGameImpl = impls.PermissionedDisputeGame
	st.ImplementationsDeployment.ZkDisputeGameImpl = impls.ZkDisputeGame
	st.ImplementationsDeployment.OpcmStandardValidatorImpl = impls.OpcmStandardValidator
	st.ImplementationsDeployment.SuperFaultDisputeGameImpl = impls.SuperFaultDisputeGame
	st.ImplementationsDeployment.SuperPermissionedDisputeGameImpl = impls.SuperPermissionedDisputeGame

	return nil
}

// ResolveChainProofParams merges the standard dispute-game defaults with the
// intent's global and per-chain deploy overrides.
func ResolveChainProofParams(intent *state.Intent, chain *state.ChainIntent) (state.ChainProofParams, error) {
	return jsonutil.MergeJSON(
		state.ChainProofParams{
			DisputeGameType:         standard.DisputeGameType,
			DisputeAbsolutePrestate: standard.DisputeAbsolutePrestate,
			DisputeMaxGameDepth:     standard.DisputeMaxGameDepth,
			DisputeSplitDepth:       standard.DisputeSplitDepth,
			DisputeClockExtension:   standard.DisputeClockExtension,
			DisputeMaxClockDuration: standard.DisputeMaxClockDuration,
		},
		intent.GlobalDeployOverrides,
		chain.DeployOverrides,
	)
}

// ResolvePreparedGameType returns the initial game type recorded by prepare after
// verifying that the current intent still resolves to the same type.
func ResolvePreparedGameType(intent *state.Intent, chain *state.ChainIntent, chainState *state.ChainState) (uint32, error) {
	if chainState == nil || chainState.InitialGameType == nil {
		return 0, fmt.Errorf("chain %s has no initial game type recorded by prepare; rerun op-deployer prepare", chain.ID.Hex())
	}

	proofParams, err := ResolveChainProofParams(intent, chain)
	if err != nil {
		return 0, fmt.Errorf("failed to resolve initial dispute game type for chain %s: %w", chain.ID.Hex(), err)
	}

	prepared := *chainState.InitialGameType
	current := proofParams.DisputeGameType
	if prepared != current {
		return 0, fmt.Errorf(
			"chain %s initial game type changed after prepare: prepared %s (%d), intent %s (%d); rerun op-deployer prepare",
			chain.ID.Hex(),
			initialGameTypeName(prepared),
			prepared,
			initialGameTypeName(current),
			current,
		)
	}
	return prepared, nil
}

func initialGameTypeName(gameType uint32) string {
	switch embedded.GameType(gameType) {
	case embedded.GameTypePermissionedCannon:
		return "PERMISSIONED_CANNON"
	case embedded.GameTypeSuperPermissioned:
		return "SUPER_PERMISSIONED"
	case embedded.GameTypeCannonKona:
		return "CANNON_KONA"
	case embedded.GameTypeSuperCannonKona:
		return "SUPER_CANNON_KONA"
	default:
		return "UNKNOWN"
	}
}

// InitialDeployRequirements defines initial game deployment requirements.
type InitialDeployRequirements struct {
	Permissionless   bool
	RequiresPrestate bool
}

// ResolveInitialDeployRequirements returns requirements for a supported initial game type.
func ResolveInitialDeployRequirements(gameType uint32) (InitialDeployRequirements, error) {
	switch embedded.GameType(gameType) {
	case embedded.GameTypePermissionedCannon:
		return InitialDeployRequirements{}, nil
	case embedded.GameTypeCannonKona, embedded.GameTypeSuperCannonKona:
		return InitialDeployRequirements{
			Permissionless:   true,
			RequiresPrestate: true,
		}, nil
	case embedded.GameTypeSuperPermissioned:
		return InitialDeployRequirements{}, fmt.Errorf(
			"initial dispute game type SUPER_PERMISSIONED (%d) is a derived fallback and is not an initial-deploy selector",
			gameType,
		)
	default:
		return InitialDeployRequirements{}, fmt.Errorf("unsupported initial dispute game type %d", gameType)
	}
}

func makeDCI(intent *state.Intent, thisIntent *state.ChainIntent, chainID common.Hash, st *state.State) (opcm.DeployOPChainInput, error) {
	proofParams, err := ResolveChainProofParams(intent, thisIntent)
	if err != nil {
		return opcm.DeployOPChainInput{}, fmt.Errorf("error merging proof params from overrides: %w", err)
	}

	requirements, err := ResolveInitialDeployRequirements(proofParams.DisputeGameType)
	if err != nil {
		return opcm.DeployOPChainInput{}, err
	}
	if requirements.Permissionless {
		return opcm.DeployOPChainInput{}, fmt.Errorf("apply only supports permissioned deploys: permissionless chains are deployed through the prepare flow")
	}

	opcmAddr := st.ImplementationsDeployment.OpcmV2Impl
	if opcmAddr == (common.Address{}) {
		return opcm.DeployOPChainInput{}, fmt.Errorf("OPCM implementation is not deployed")
	}

	return BuildDeployOPChainInput(
		proofParams,
		thisIntent.Roles,
		opcmAddr,
		st.SuperchainDeployment.SuperchainConfigProxy,
		chainID,
		st.Create2Salt.String(),
		thisIntent.GasLimit,
		opcm.Proposal{
			Root:             opcm.DefaultStartingAnchorRoot.Root,
			L2SequenceNumber: common.Big0,
		},
		thisIntent,
	), nil
}

func BuildDeployOPChainInput(
	proofParams state.ChainProofParams,
	roles state.ChainRoles,
	opcmAddr common.Address,
	superchainConfig common.Address,
	l2ChainID common.Hash,
	saltMixer string,
	gasLimit uint64,
	startingAnchorRoot opcm.Proposal,
	chain *state.ChainIntent,
) opcm.DeployOPChainInput {
	if gasLimit == 0 {
		gasLimit = standard.GasLimit
	}

	cannonAbsolutePrestate := proofParams.DisputeAbsolutePrestate
	if proofParams.DisputeGameType == uint32(embedded.GameTypeCannonKona) {
		cannonAbsolutePrestate = opcm.PermissionedGamePrestatePlaceholder
	}

	return opcm.DeployOPChainInput{
		OpChainProxyAdminOwner:       roles.L1ProxyAdminOwner,
		SystemConfigOwner:            roles.SystemConfigOwner,
		Batcher:                      roles.Batcher,
		UnsafeBlockSigner:            roles.UnsafeBlockSigner,
		Proposer:                     roles.Proposer,
		Challenger:                   roles.Challenger,
		BasefeeScalar:                standard.BasefeeScalar,
		BlobBaseFeeScalar:            standard.BlobBaseFeeScalar,
		L2ChainId:                    l2ChainID.Big(),
		Opcm:                         opcmAddr,
		SaltMixer:                    saltMixer,
		GasLimit:                     gasLimit,
		DisputeGameType:              proofParams.DisputeGameType,
		DisputeAbsolutePrestate:      proofParams.DisputeAbsolutePrestate,
		StartingAnchorRoot:           startingAnchorRoot,
		CannonAbsolutePrestate:       cannonAbsolutePrestate,
		DisputeMaxGameDepth:          new(big.Int).SetUint64(proofParams.DisputeMaxGameDepth),
		DisputeSplitDepth:            new(big.Int).SetUint64(proofParams.DisputeSplitDepth),
		DisputeClockExtension:        proofParams.DisputeClockExtension,   // 3 hours (input in seconds)
		DisputeMaxClockDuration:      proofParams.DisputeMaxClockDuration, // 3.5 days (input in seconds)
		AllowCustomDisputeParameters: proofParams.DangerouslyAllowCustomDisputeParameters,
		OperatorFeeScalar:            chain.OperatorFeeScalar,
		OperatorFeeConstant:          chain.OperatorFeeConstant,
		SuperchainConfig:             superchainConfig,
		UseCustomGasToken:            chain.IsCustomGasTokenEnabled(),
	}
}

// OpChainContractsFromDeployOutput maps a DeployOPChain output to OpChainContracts
func OpChainContractsFromDeployOutput(dco opcm.DeployOPChainOutput) addresses.OpChainContracts {
	var opChainContracts addresses.OpChainContracts
	opChainContracts.OpChainProxyAdminImpl = dco.OpChainProxyAdmin
	opChainContracts.AddressManagerImpl = dco.AddressManager
	opChainContracts.L1Erc721BridgeProxy = dco.L1ERC721BridgeProxy
	opChainContracts.SystemConfigProxy = dco.SystemConfigProxy
	opChainContracts.OptimismMintableErc20FactoryProxy = dco.OptimismMintableERC20FactoryProxy
	opChainContracts.L1StandardBridgeProxy = dco.L1StandardBridgeProxy
	opChainContracts.L1CrossDomainMessengerProxy = dco.L1CrossDomainMessengerProxy
	opChainContracts.OptimismPortalProxy = dco.OptimismPortalProxy
	opChainContracts.EthLockboxProxy = dco.EthLockboxProxy
	opChainContracts.DisputeGameFactoryProxy = dco.DisputeGameFactoryProxy
	opChainContracts.AnchorStateRegistryProxy = dco.AnchorStateRegistryProxy
	opChainContracts.FaultDisputeGameImpl = dco.FaultDisputeGame
	opChainContracts.PermissionedDisputeGameImpl = dco.PermissionedDisputeGame
	opChainContracts.DelayedWethPermissionedGameProxy = dco.DelayedWETHPermissionedGameProxy
	opChainContracts.DelayedWethPermissionlessGameProxy = dco.DelayedWETHPermissionlessGameProxy
	return opChainContracts
}

// chainContractsForDeploy builds the OpChainContracts for a deployed chain,
// overriding the dispute game impls with the values read back from the proxies.
func chainContractsForDeploy(impls opcm.ReadImplementationAddressesOutput, dco opcm.DeployOPChainOutput) addresses.OpChainContracts {
	opChainContracts := OpChainContractsFromDeployOutput(dco)

	if (impls.PermissionedDisputeGame != common.Address{}) {
		opChainContracts.PermissionedDisputeGameImpl = impls.PermissionedDisputeGame
	}
	if (impls.FaultDisputeGame != common.Address{}) {
		opChainContracts.FaultDisputeGameImpl = impls.FaultDisputeGame
	}

	return opChainContracts
}

// shouldDeployOPChain reports whether the given chainID still needs to be deployed.
func shouldDeployOPChain(st *state.State, chainID common.Hash) bool {
	return !st.IsChainDeployed(chainID)
}
