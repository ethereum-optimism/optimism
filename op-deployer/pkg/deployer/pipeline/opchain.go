package pipeline

import (
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	"github.com/ethereum/go-ethereum/common"
)

// OPChainDeploymentResult must be obtained from ExecuteOPChainDeployment.
type OPChainDeploymentResult struct {
	chainID         common.Hash
	contracts       addresses.OpChainContracts
	outputContracts addresses.OpChainContracts
	readback        opcm.ReadImplementationAddressesOutput
	initialized     bool
}

// Contracts returns the addresses emitted by DeployOPChain, matching the set
// recorded by prepare. RecordOPChainDeployment uses a separate set whose dispute
// game implementations are replaced with addresses read back from their proxies.
func (r OPChainDeploymentResult) Contracts() addresses.OpChainContracts {
	return r.outputContracts
}

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

	dci, err := makeDCI(intent, thisIntent, chainID, st)
	if err != nil {
		return fmt.Errorf("error making deploy OP chain input: %w", err)
	}

	result, err := ExecuteOPChainDeployment(env, st, chainID, dci)
	if err != nil {
		return err
	}
	// Record in memory. The stage runner persists state after its broadcast succeeds.
	return RecordOPChainDeployment(st, result)
}

// ExecuteOPChainDeployment runs the deployment without recording state.
// Script-host execution queues transactions. Forge broadcasts directly.
func ExecuteOPChainDeployment(
	env *Env,
	st *state.State,
	chainID common.Hash,
	dci opcm.DeployOPChainInput,
) (OPChainDeploymentResult, error) {
	var result OPChainDeploymentResult
	if dci.L2ChainId == nil {
		return result, fmt.Errorf("deploy OP chain input has nil L2 chain ID; expected %s", chainID.Big())
	}
	if dci.L2ChainId.Cmp(chainID.Big()) != 0 {
		return result, fmt.Errorf(
			"deploy OP chain input L2 chain ID %s does not match requested chain ID %s",
			dci.L2ChainId,
			chainID.Big(),
		)
	}

	lgr := env.Logger.New("stage", "deploy-opchain")
	lgr.Info("deploying OP chain using local allocs", "id", chainID.Hex())

	// We make sure that the deployer and OPCM are the same as the ones used in the dry-run, if any.
	// Skip when the deployer is the placeholder, which means non-live strategies are being used.
	if env.Deployer != standard.PlaceholderAddress {
		if err := st.CheckL1PredictInputs(env.Deployer, dci.Opcm); err != nil {
			return result, err
		}
	}

	var dco opcm.DeployOPChainOutput
	var err error
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
			return result, err
		}
	} else {
		dco, err = env.Scripts.DeployOPChain.Run(dci)
		if err != nil {
			return result, fmt.Errorf("error deploying OP chain: %w", err)
		}
	}

	return readOPChainDeployment(env, chainID, OpChainContractsFromDeployOutput(dco), dci.Opcm)
}

// ReconcileOPChainDeployment reads the implementation addresses needed to
// produce the same recorded state as a freshly receipted deployment.
func ReconcileOPChainDeployment(
	env *Env,
	chainID common.Hash,
	expected addresses.OpChainContracts,
	pinnedOPCM common.Address,
) (OPChainDeploymentResult, error) {
	return readOPChainDeployment(env, chainID, expected, pinnedOPCM)
}

func readOPChainDeployment(
	env *Env,
	chainID common.Hash,
	outputContracts addresses.OpChainContracts,
	pinnedOPCM common.Address,
) (OPChainDeploymentResult, error) {
	readInput := opcm.ReadImplementationAddressesInput{
		AddressManager:                    outputContracts.AddressManagerImpl,
		L1ERC721BridgeProxy:               outputContracts.L1Erc721BridgeProxy,
		SystemConfigProxy:                 outputContracts.SystemConfigProxy,
		OptimismMintableERC20FactoryProxy: outputContracts.OptimismMintableErc20FactoryProxy,
		L1StandardBridgeProxy:             outputContracts.L1StandardBridgeProxy,
		OptimismPortalProxy:               outputContracts.OptimismPortalProxy,
		DisputeGameFactoryProxy:           outputContracts.DisputeGameFactoryProxy,
		Opcm:                              pinnedOPCM,
	}

	var impls opcm.ReadImplementationAddressesOutput
	var err error
	if env.UseForge {
		env.Logger.Info("using Forge for ReadImplementationAddresses", "stage", "deploy-opchain")
		forgeEnv := &opcm.ForgeEnv{
			Client:   env.ForgeClient,
			Context:  env.Context,
			L1RPCUrl: env.L1RPCUrl,
		}
		impls, err = opcm.ReadImplementationAddressesViaForge(forgeEnv, readInput)
		if err != nil {
			return OPChainDeploymentResult{}, err
		}
	} else {
		readImplementations, err := opcm.NewReadImplementationAddressesScript(env.L1ScriptHost)
		if err != nil {
			return OPChainDeploymentResult{}, fmt.Errorf("failed to load ReadImplementationAddresses script: %w", err)
		}

		impls, err = readImplementations.Run(readInput)
		if err != nil {
			return OPChainDeploymentResult{}, fmt.Errorf("failed to run ReadImplementationAddresses script: %w", err)
		}
	}

	return OPChainDeploymentResult{
		chainID:         chainID,
		contracts:       chainContractsForRecordedState(impls, outputContracts),
		outputContracts: outputContracts,
		readback:        impls,
		initialized:     true,
	}, nil
}

// RecordOPChainDeployment updates in-memory state and is safe to repeat.
func RecordOPChainDeployment(st *state.State, result OPChainDeploymentResult) error {
	if !result.initialized {
		return fmt.Errorf("cannot record an uninitialized OP chain deployment result")
	}

	st.SetChainContracts(result.chainID, result.contracts, true)

	// Continuation deployments may reference an existing OPCM without carrying
	// its implementation addresses in state. Full apply deployments do carry
	// them, so refresh the recorded addresses only when that record exists.
	if st.ImplementationsDeployment != nil {
		impls := result.readback
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
	}
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
// verifying that it matches the currently resolved game type.
func ResolvePreparedGameType(chain *state.ChainIntent, chainState *state.ChainState, current uint32) (uint32, error) {
	if chainState == nil || chainState.InitialGameType == nil {
		return 0, fmt.Errorf("chain %s has no initial game type recorded by prepare; rerun op-deployer prepare", chain.ID.Hex())
	}

	prepared := *chainState.InitialGameType
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

// IsSuperGameType reports whether the given dispute game type is SUPER_CANNON_KONA.
// SUPER_PERMISSIONED is deliberately excluded as it's a derived fallback and
// can never appear here as a chain's resolved DisputeGameType.
func IsSuperGameType(gameType uint32) bool {
	return embedded.GameType(gameType) == embedded.GameTypeSuperCannonKona
}

// DeploymentUsesSuperRoots reports whether the starting anchors are SuperV1 roots over the
// dependency set rather than per-chain output roots. Deployed chains are ignored, their
// games being fixed on L1.
func DeploymentUsesSuperRoots(intent *state.Intent, st *state.State) (bool, error) {
	for _, chain := range intent.Chains {
		if st.IsChainDeployed(chain.ID) {
			continue
		}
		proofParams, err := ResolveChainProofParams(intent, chain)
		if err != nil {
			return false, fmt.Errorf("failed to resolve proof params for chain %s: %w", chain.ID.Hex(), err)
		}
		if IsSuperGameType(proofParams.DisputeGameType) {
			return true, nil
		}
	}
	return false, nil
}

// ValidateInitialGameTypeSet rejects a mix of CANNON_KONA and
// SUPER_CANNON_KONA initial games.
func ValidateInitialGameTypeSet(gameTypes []uint32) error {
	hasCannonKona := false
	hasSuperCannonKona := false
	for _, gameType := range gameTypes {
		switch embedded.GameType(gameType) {
		case embedded.GameTypeCannonKona:
			hasCannonKona = true
		case embedded.GameTypeSuperCannonKona:
			hasSuperCannonKona = true
		}
	}

	if hasCannonKona && hasSuperCannonKona {
		return fmt.Errorf("an intent cannot mix CANNON_KONA and SUPER_CANNON_KONA initial games")
	}
	return nil
}

// ResolveInitialDeployRequirements returns requirements for a supported initial game type.
func ResolveInitialDeployRequirements(gameType uint32) (InitialDeployRequirements, error) {
	switch embedded.GameType(gameType) {
	case embedded.GameTypePermissionedCannon, embedded.GameTypeSuperPermissioned:
		return InitialDeployRequirements{}, nil
	case embedded.GameTypeCannonKona, embedded.GameTypeSuperCannonKona:
		return InitialDeployRequirements{
			Permissionless:   true,
			RequiresPrestate: true,
		}, nil
	default:
		return InitialDeployRequirements{}, fmt.Errorf("unsupported initial dispute game type %d", gameType)
	}
}

// BuildContinuationDCI uses the values saved by prepare.
// It reads Prestate and StartingAnchorRoot from ChainState because later steps set them.
func BuildContinuationDCI(chainID common.Hash, st *state.State) (opcm.DeployOPChainInput, error) {
	if st == nil || st.PreparedDeployment == nil || st.PreparedDeployment.Intent == nil {
		return opcm.DeployOPChainInput{}, fmt.Errorf("state was not produced by op-deployer prepare. Run op-deployer prepare")
	}
	prepared := st.PreparedDeployment
	if st.Create2Salt == (common.Hash{}) {
		return opcm.DeployOPChainInput{}, fmt.Errorf("prepared state has no CREATE2 salt. Rerun op-deployer prepare")
	}
	if prepared.Deployer == (common.Address{}) {
		return opcm.DeployOPChainInput{}, fmt.Errorf("prepared state has no predicted sender address. Rerun op-deployer prepare")
	}
	if prepared.OPCM == (common.Address{}) {
		return opcm.DeployOPChainInput{}, fmt.Errorf("prepared state has no predicted OPCM address. Rerun op-deployer prepare")
	}
	if prepared.Intent.SuperchainConfigProxy == nil || *prepared.Intent.SuperchainConfigProxy == (common.Address{}) {
		return opcm.DeployOPChainInput{}, fmt.Errorf("intent.superchainConfigProxy must be set")
	}

	if _, err := prepared.Chain(chainID); err != nil {
		return opcm.DeployOPChainInput{}, err
	}
	chainState, err := st.Chain(chainID)
	if err != nil {
		return opcm.DeployOPChainInput{}, fmt.Errorf(
			"failed to get state prepared for chain %s: %w. Rerun op-deployer prepare",
			chainID.Hex(),
			err,
		)
	}
	thisIntent, err := prepared.Intent.Chain(chainID)
	if err != nil {
		return opcm.DeployOPChainInput{}, fmt.Errorf("failed to get prepared chain intent: %w", err)
	}

	proofParams, err := ResolveChainProofParams(prepared.Intent, thisIntent)
	if err != nil {
		return opcm.DeployOPChainInput{}, fmt.Errorf("error merging prepared proof params: %w", err)
	}
	preparedGameType, err := ResolvePreparedGameType(thisIntent, chainState, proofParams.DisputeGameType)
	if err != nil {
		return opcm.DeployOPChainInput{}, err
	}
	requirements, err := ResolveInitialDeployRequirements(preparedGameType)
	if err != nil {
		return opcm.DeployOPChainInput{}, fmt.Errorf(
			"chain %s has an invalid prepared game type: %w. Rerun op-deployer prepare",
			chainID.Hex(),
			err,
		)
	}

	if requirements.RequiresPrestate {
		if chainState.Prestate == (common.Hash{}) {
			return opcm.DeployOPChainInput{}, fmt.Errorf(
				"chain %s has no prestate committed. Run op-deployer prestate",
				chainID.Hex(),
			)
		}
		if chainState.Prestate == opcm.PermissionedCannonFallbackPrestatePlaceholder {
			return opcm.DeployOPChainInput{}, fmt.Errorf(
				"chain %s has the reserved permissioned prestate placeholder committed. Rerun op-deployer prestate",
				chainID.Hex(),
			)
		}
	}

	startingAnchorRoot := opcm.DefaultStartingAnchorProposal()
	if requirements.Permissionless {
		if chainState.StartingAnchorRoot == nil || chainState.StartingAnchorRoot.Root == (common.Hash{}) {
			return opcm.DeployOPChainInput{}, fmt.Errorf(
				"chain %s has no valid starting anchor proposal committed. Rerun the proposal-producing stage",
				chainID.Hex(),
			)
		}
		if chainState.StartingAnchorRoot.Root == opcm.DefaultStartingAnchorRoot.Root {
			return opcm.DeployOPChainInput{}, fmt.Errorf(
				"chain %s has the permissioned starting anchor placeholder committed. Rerun the proposal-producing stage",
				chainID.Hex(),
			)
		}
		// The initial anchor must leave room for a strictly greater uint64 game sequence.
		// The field is uint64-bounded, so equality is the only invalid value representable here.
		if chainState.StartingAnchorRoot.L2SequenceNumber == math.MaxUint64 {
			return opcm.DeployOPChainInput{}, fmt.Errorf(
				"chain %s has a starting anchor sequence number that is too large. Rerun the proposal-producing stage",
				chainID.Hex(),
			)
		}

		startingAnchorRoot = opcm.Proposal{
			Root: chainState.StartingAnchorRoot.Root,
			L2SequenceNumber: new(big.Int).SetUint64(
				uint64(chainState.StartingAnchorRoot.L2SequenceNumber),
			),
		}
	}

	if requirements.RequiresPrestate {
		proofParams.DisputeAbsolutePrestate = chainState.Prestate
	}

	return BuildDeployOPChainInput(
		proofParams,
		thisIntent.Roles,
		prepared.OPCM,
		*prepared.Intent.SuperchainConfigProxy,
		chainID,
		st.Create2Salt.String(),
		thisIntent.GasLimit,
		startingAnchorRoot,
		thisIntent,
	), nil
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
		opcm.DefaultStartingAnchorProposal(),
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

	var cannonAbsolutePrestate common.Hash
	switch embedded.GameType(proofParams.DisputeGameType) {
	case embedded.GameTypeCannonKona:
		cannonAbsolutePrestate = opcm.PermissionedCannonFallbackPrestatePlaceholder
	case embedded.GameTypeSuperCannonKona, embedded.GameTypeSuperPermissioned:
		cannonAbsolutePrestate = common.Hash{}
	case embedded.GameTypePermissionedCannon:
		cannonAbsolutePrestate = proofParams.DisputeAbsolutePrestate
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

func chainContractsForRecordedState(
	impls opcm.ReadImplementationAddressesOutput,
	opChainContracts addresses.OpChainContracts,
) addresses.OpChainContracts {
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
