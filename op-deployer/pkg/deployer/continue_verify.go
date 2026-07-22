package deployer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type continuationReadBackend interface {
	opcm.CallContractBackend
	CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error)
}

var _ continuationReadBackend = (*ethclient.Client)(nil)

type scriptHostReadBackend struct {
	*opcm.ScriptHostCallBackend
	host *script.Host
}

var _ continuationReadBackend = (*scriptHostReadBackend)(nil)

func newScriptHostReadBackend(host *script.Host) *scriptHostReadBackend {
	return &scriptHostReadBackend{
		ScriptHostCallBackend: opcm.NewScriptHostCallBackend(host),
		host:                  host,
	}
}

func (b *scriptHostReadBackend) CodeAt(
	ctx context.Context,
	contract common.Address,
	blockNumber *big.Int,
) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if blockNumber != nil {
		return nil, fmt.Errorf("script host code reads do not support a block number")
	}
	return bytes.Clone(b.host.GetCode(contract)), nil
}

type continuationGameMode struct {
	initialGameType        uint32
	initialImplementation  common.Address
	fallbackGameType       *uint32
	fallbackImplementation common.Address
	permissionedGameType   uint32
	permissionless         bool
	hasChallenger          bool
}

type continuationVerifier struct {
	ctx      context.Context
	backend  continuationReadBackend
	observed addresses.OpChainContracts
	expected *state.ChainState
	dci      opcm.DeployOPChainInput
	failures []error
}

func verifyContinuationDeployment(
	ctx context.Context,
	backend continuationReadBackend,
	observed addresses.OpChainContracts,
	expected *state.ChainState,
	dci opcm.DeployOPChainInput,
) error {
	if backend == nil {
		return fmt.Errorf("continuation verification backend is nil")
	}
	if expected == nil {
		return fmt.Errorf("prepared ChainState is nil")
	}

	verifier := &continuationVerifier{
		ctx:      ctx,
		backend:  backend,
		observed: observed,
		expected: expected,
		dci:      dci,
	}
	if !verifier.verifyAddressesAndCode() {
		return errors.Join(verifier.failures...)
	}

	mode, err := verifier.resolveGameMode()
	if err != nil {
		verifier.failures = append(verifier.failures, err)
	} else {
		verifier.verifyStartingAnchor(mode)
		verifier.verifyGameConfiguration(mode)
		verifier.verifyPermissionedGameRoles(mode)
	}
	verifier.verifySystemConfig()
	verifier.verifyProxyAdminOwner()
	verifier.verifySuperchainConfig()
	if mode.permissionless {
		verifier.verifyStandardValidator()
	}
	return errors.Join(verifier.failures...)
}

func (v *continuationVerifier) resolveGameMode() (continuationGameMode, error) {
	if v.expected.InitialGameType == nil {
		return continuationGameMode{}, fmt.Errorf("initial game type mismatch: expected a value from prepared ChainState.InitialGameType, observed nil")
	}
	gameType := *v.expected.InitialGameType
	if gameType != v.dci.DisputeGameType {
		return continuationGameMode{}, fmt.Errorf(
			"initial game type mismatch: expected %d from prepared ChainState.InitialGameType, observed %d in frozen DeployOPChainInput",
			gameType,
			v.dci.DisputeGameType,
		)
	}

	mode := continuationGameMode{initialGameType: gameType}
	switch embedded.GameType(gameType) {
	case embedded.GameTypePermissionedCannon:
		mode.initialImplementation = v.expected.PermissionedDisputeGameImpl
		mode.permissionedGameType = gameType
		mode.hasChallenger = true
	case embedded.GameTypeCannonKona:
		fallback := uint32(embedded.GameTypePermissionedCannon)
		mode.initialImplementation = v.expected.FaultDisputeGameImpl
		mode.fallbackGameType = &fallback
		mode.fallbackImplementation = v.expected.PermissionedDisputeGameImpl
		mode.permissionedGameType = fallback
		mode.permissionless = true
		mode.hasChallenger = true
	case embedded.GameTypeSuperCannonKona:
		fallback := uint32(embedded.GameTypeSuperPermissioned)
		mode.initialImplementation = v.expected.FaultDisputeGameImpl
		mode.fallbackGameType = &fallback
		mode.fallbackImplementation = v.expected.PermissionedDisputeGameImpl
		mode.permissionedGameType = fallback
		mode.permissionless = true
	default:
		return continuationGameMode{}, fmt.Errorf(
			"initial game type mismatch: expected a supported value from prepared ChainState.InitialGameType, observed %d",
			gameType,
		)
	}
	return mode, nil
}

func (v *continuationVerifier) verifyAddressesAndCode() bool {
	if v.observed != v.expected.OpChainContracts {
		v.addMismatch(
			"simulated contract addresses differ from the prepared OpChainContracts address set",
			"predicted ChainState.OpChainContracts",
			fmt.Sprintf("%+v", v.expected.OpChainContracts),
			fmt.Sprintf("%+v", v.observed),
		)
		return false
	}

	valid := true
	for _, contract := range continuationContractAddresses(v.expected.OpChainContracts) {
		if contract.address == (common.Address{}) {
			continue
		}
		code, err := v.backend.CodeAt(v.ctx, contract.address, nil)
		if err != nil {
			valid = false
			v.addReadError(
				contract.name+" code",
				"predicted ChainState.OpChainContracts",
				fmt.Sprintf("non-empty code at %s", contract.address),
				err,
			)
			continue
		}
		if len(code) == 0 {
			valid = false
			v.addMismatch(
				contract.name+" code",
				"predicted ChainState.OpChainContracts",
				fmt.Sprintf("non-empty code at %s", contract.address),
				"empty code",
			)
		}
	}
	return valid
}

func (v *continuationVerifier) verifyStartingAnchor(mode continuationGameMode) {
	expectedRoot := opcm.DefaultStartingAnchorRoot.Root
	expectedSequence := new(big.Int)
	source := "permissioned starting-anchor placeholder"
	if mode.permissionless {
		source = "committed ChainState.StartingAnchorRoot"
		if v.expected.StartingAnchorRoot == nil {
			v.addMismatch("starting anchor proposal", source, "a committed proposal", "nil")
			return
		}
		expectedRoot = v.expected.StartingAnchorRoot.Root
		expectedSequence.SetUint64(uint64(v.expected.StartingAnchorRoot.L2SequenceNumber))
	}

	root, sequence, err := readContinuationAnchor(
		v.ctx,
		v.backend,
		v.expected.AnchorStateRegistryProxy,
	)
	if err != nil {
		v.addReadError(
			"starting anchor proposal",
			source,
			fmt.Sprintf("root %s and sequence %s", expectedRoot, expectedSequence),
			err,
		)
		return
	}
	if root != expectedRoot {
		v.addMismatch("starting anchor root", source, expectedRoot, root)
	}
	if sequence.Cmp(expectedSequence) != 0 {
		v.addMismatch("starting anchor sequence", source, expectedSequence, sequence)
	}
}

func (v *continuationVerifier) verifyGameConfiguration(mode continuationGameMode) {
	respected, err := readContinuationUint32(
		v.ctx,
		v.backend,
		v.expected.OptimismPortalProxy,
		continuationRespectedGameTypeMethod,
	)
	if err != nil {
		v.addReadError(
			"respected game type",
			"prepared ChainState.InitialGameType",
			mode.initialGameType,
			err,
		)
	} else if respected != mode.initialGameType {
		v.addMismatch(
			"respected game type",
			"prepared ChainState.InitialGameType",
			mode.initialGameType,
			respected,
		)
	}

	initialImplementation, err := readContinuationAddress(
		v.ctx,
		v.backend,
		v.expected.DisputeGameFactoryProxy,
		continuationGameImplMethod,
		mode.initialGameType,
	)
	if err != nil {
		v.addReadError(
			"initial game implementation",
			"predicted ChainState.OpChainContracts",
			mode.initialImplementation,
			err,
		)
	} else if initialImplementation != mode.initialImplementation {
		v.addMismatch(
			"initial game implementation",
			"predicted ChainState.OpChainContracts",
			mode.initialImplementation,
			initialImplementation,
		)
	}

	if mode.fallbackGameType != nil {
		fallbackImplementation, err := readContinuationAddress(
			v.ctx,
			v.backend,
			v.expected.DisputeGameFactoryProxy,
			continuationGameImplMethod,
			*mode.fallbackGameType,
		)
		if err != nil {
			v.addReadError(
				"fallback game implementation",
				"prepared initial-game fallback rules and predicted ChainState.OpChainContracts",
				mode.fallbackImplementation,
				err,
			)
		} else if fallbackImplementation != mode.fallbackImplementation {
			v.addMismatch(
				"fallback game implementation",
				"prepared initial-game fallback rules and predicted ChainState.OpChainContracts",
				mode.fallbackImplementation,
				fallbackImplementation,
			)
		}
	}

	expectedPrestate := v.dci.DisputeAbsolutePrestate
	prestateSource := "frozen DeployOPChainInput.DisputeAbsolutePrestate"
	if mode.permissionless {
		expectedPrestate = v.expected.Prestate
		prestateSource = "committed ChainState.Prestate"
		if v.dci.DisputeAbsolutePrestate != expectedPrestate {
			v.addMismatch(
				"selected prestate input",
				prestateSource,
				expectedPrestate,
				v.dci.DisputeAbsolutePrestate,
			)
		}
	}
	initialGameArgs, err := readContinuationBytes(
		v.ctx,
		v.backend,
		v.expected.DisputeGameFactoryProxy,
		continuationGameArgsMethod,
		mode.initialGameType,
	)
	if err != nil {
		v.addReadError("selected prestate", prestateSource, expectedPrestate, err)
	} else {
		prestate, err := decodeContinuationGamePrestate(initialGameArgs, mode.permissionless)
		if err != nil {
			v.addReadError("selected prestate", prestateSource, expectedPrestate, err)
		} else if prestate != expectedPrestate {
			v.addMismatch("selected prestate", prestateSource, expectedPrestate, prestate)
		}
	}

	switch embedded.GameType(mode.initialGameType) {
	case embedded.GameTypeCannonKona:
		fallbackGameArgs, err := readContinuationBytes(
			v.ctx,
			v.backend,
			v.expected.DisputeGameFactoryProxy,
			continuationGameArgsMethod,
			*mode.fallbackGameType,
		)
		if err != nil {
			v.addReadError(
				"fallback prestate",
				"fixed CANNON_KONA permissioned fallback",
				opcm.PermissionedCannonFallbackPrestatePlaceholder,
				err,
			)
		} else {
			fallbackPrestate, err := decodeContinuationGamePrestate(fallbackGameArgs, false)
			if err != nil {
				v.addReadError(
					"fallback prestate",
					"fixed CANNON_KONA permissioned fallback",
					opcm.PermissionedCannonFallbackPrestatePlaceholder,
					err,
				)
			} else if fallbackPrestate != opcm.PermissionedCannonFallbackPrestatePlaceholder {
				v.addMismatch(
					"fallback prestate",
					"fixed CANNON_KONA permissioned fallback",
					opcm.PermissionedCannonFallbackPrestatePlaceholder,
					fallbackPrestate,
				)
			}
		}
	case embedded.GameTypeSuperCannonKona:
		if v.dci.CannonAbsolutePrestate != (common.Hash{}) {
			v.addMismatch(
				"fallback prestate input",
				"SUPER_CANNON_KONA no-prestate fallback rule",
				common.Hash{},
				v.dci.CannonAbsolutePrestate,
			)
		}
	}
}

func (v *continuationVerifier) verifyPermissionedGameRoles(mode continuationGameMode) {
	gameArgs, err := readContinuationBytes(
		v.ctx,
		v.backend,
		v.expected.DisputeGameFactoryProxy,
		continuationGameArgsMethod,
		mode.permissionedGameType,
	)
	if err != nil {
		v.addReadError(
			"permissioned game roles",
			"frozen DeployOPChainInput proposer and challenger",
			"readable configured game arguments",
			err,
		)
		return
	}
	proposer, challenger, err := decodeContinuationPermissionedRoles(gameArgs, mode.hasChallenger)
	if err != nil {
		v.addReadError(
			"permissioned game roles",
			"frozen DeployOPChainInput proposer and challenger",
			"valid configured game arguments",
			err,
		)
		return
	}
	if proposer != v.dci.Proposer {
		v.addMismatch(
			"permissioned game proposer",
			"frozen DeployOPChainInput.Proposer",
			v.dci.Proposer,
			proposer,
		)
	}

	if !mode.hasChallenger {
		return
	}
	if challenger != v.dci.Challenger {
		v.addMismatch(
			"permissioned game challenger",
			"frozen DeployOPChainInput.Challenger",
			v.dci.Challenger,
			challenger,
		)
	}
}

func (v *continuationVerifier) verifySystemConfig() {
	systemConfig := v.expected.SystemConfigProxy
	v.verifyAddressGetter(
		"SystemConfig owner",
		"frozen DeployOPChainInput.SystemConfigOwner",
		systemConfig,
		continuationOwnerMethod,
		v.dci.SystemConfigOwner,
	)

	batcherHash, err := readContinuationHash(
		v.ctx,
		v.backend,
		systemConfig,
		continuationBatcherHashMethod,
	)
	expectedBatcherHash := common.BytesToHash(v.dci.Batcher.Bytes())
	if err != nil {
		v.addReadError(
			"SystemConfig batcher",
			"frozen DeployOPChainInput.Batcher",
			expectedBatcherHash,
			err,
		)
	} else if batcherHash != expectedBatcherHash {
		v.addMismatch(
			"SystemConfig batcher",
			"frozen DeployOPChainInput.Batcher",
			expectedBatcherHash,
			batcherHash,
		)
	}

	v.verifyAddressGetter(
		"SystemConfig unsafe block signer",
		"frozen DeployOPChainInput.UnsafeBlockSigner",
		systemConfig,
		continuationUnsafeBlockSignerMethod,
		v.dci.UnsafeBlockSigner,
	)

	gasLimit, err := readContinuationUint64(
		v.ctx,
		v.backend,
		systemConfig,
		continuationGasLimitMethod,
	)
	if err != nil {
		v.addReadError(
			"SystemConfig gas limit",
			"frozen DeployOPChainInput.GasLimit",
			v.dci.GasLimit,
			err,
		)
	} else if gasLimit != v.dci.GasLimit {
		v.addMismatch(
			"SystemConfig gas limit",
			"frozen DeployOPChainInput.GasLimit",
			v.dci.GasLimit,
			gasLimit,
		)
	}

	l2ChainID, err := readContinuationBig(
		v.ctx,
		v.backend,
		systemConfig,
		continuationL2ChainIDMethod,
	)
	if err != nil {
		v.addReadError(
			"SystemConfig L2 chain ID",
			"frozen DeployOPChainInput.L2ChainId",
			v.dci.L2ChainId,
			err,
		)
	} else if v.dci.L2ChainId == nil || l2ChainID.Cmp(v.dci.L2ChainId) != 0 {
		v.addMismatch(
			"SystemConfig L2 chain ID",
			"frozen DeployOPChainInput.L2ChainId",
			v.dci.L2ChainId,
			l2ChainID,
		)
	}
}

func (v *continuationVerifier) verifyProxyAdminOwner() {
	v.verifyAddressGetter(
		"OpChain ProxyAdmin owner",
		"frozen DeployOPChainInput.OpChainProxyAdminOwner",
		v.expected.OpChainProxyAdminImpl,
		continuationOwnerMethod,
		v.dci.OpChainProxyAdminOwner,
	)
}

func (v *continuationVerifier) verifySuperchainConfig() {
	attachments := []struct {
		name    string
		address common.Address
		method  abi.Method
	}{
		{"SystemConfig", v.expected.SystemConfigProxy, continuationSuperchainConfigMethod},
		{"OptimismPortal", v.expected.OptimismPortalProxy, continuationSuperchainConfigMethod},
		{"AnchorStateRegistry", v.expected.AnchorStateRegistryProxy, continuationSuperchainConfigMethod},
		{"L1CrossDomainMessenger", v.expected.L1CrossDomainMessengerProxy, continuationSuperchainConfigMethod},
		{"L1ERC721Bridge", v.expected.L1Erc721BridgeProxy, continuationSuperchainConfigMethod},
		{"L1StandardBridge", v.expected.L1StandardBridgeProxy, continuationSuperchainConfigMethod},
		{"ETHLockbox", v.expected.EthLockboxProxy, continuationSuperchainConfigMethod},
		{"DelayedWETH permissioned", v.expected.DelayedWethPermissionedGameProxy, continuationConfigMethod},
		{"DelayedWETH permissionless", v.expected.DelayedWethPermissionlessGameProxy, continuationConfigMethod},
	}
	for _, attachment := range attachments {
		if attachment.address == (common.Address{}) {
			continue
		}
		v.verifyAddressGetter(
			attachment.name+" SuperchainConfig attachment",
			"frozen DeployOPChainInput.SuperchainConfig",
			attachment.address,
			attachment.method,
			v.dci.SuperchainConfig,
		)
	}

	guardian, err := readContinuationAddress(
		v.ctx,
		v.backend,
		v.dci.SuperchainConfig,
		continuationGuardianMethod,
	)
	if err != nil {
		v.addReadError(
			"SuperchainConfig guardian",
			"guardian read live from frozen DeployOPChainInput.SuperchainConfig",
			"a readable guardian",
			err,
		)
		return
	}
	v.verifyAddressGetter(
		"SystemConfig guardian",
		"guardian read live from frozen DeployOPChainInput.SuperchainConfig",
		v.expected.SystemConfigProxy,
		continuationGuardianMethod,
		guardian,
	)
	v.verifyAddressGetter(
		"OptimismPortal guardian",
		"guardian read live from frozen DeployOPChainInput.SuperchainConfig",
		v.expected.OptimismPortalProxy,
		continuationGuardianMethod,
		guardian,
	)
}

func (v *continuationVerifier) verifyStandardValidator() {
	validator, err := readContinuationAddress(
		v.ctx,
		v.backend,
		v.dci.Opcm,
		continuationStandardValidatorMethod,
	)
	if err != nil {
		v.addReadError(
			"StandardValidator address",
			"pinned DeployOPChainInput.Opcm",
			"a readable OPCM StandardValidator",
			err,
		)
		return
	}
	if err := opcm.ValidateStandardDeployment(
		v.ctx,
		v.backend,
		validator,
		standardValidatorInput(v.dci, v.expected.OpChainContracts),
	); err != nil {
		v.addMismatch(
			"StandardValidator result",
			"frozen DeployOPChainInput",
			"successful call with an empty result",
			err,
		)
	}
}

func (v *continuationVerifier) verifyAddressGetter(
	check string,
	source string,
	contract common.Address,
	method abi.Method,
	expected common.Address,
) {
	observed, err := readContinuationAddress(v.ctx, v.backend, contract, method)
	if err != nil {
		v.addReadError(check, source, expected, err)
	} else if observed != expected {
		v.addMismatch(check, source, expected, observed)
	}
}

func (v *continuationVerifier) addMismatch(check string, source string, expected any, observed any) {
	v.failures = append(v.failures, fmt.Errorf(
		"%s: expected %v from %s, observed %v",
		check,
		expected,
		source,
		observed,
	))
}

func (v *continuationVerifier) addReadError(check string, source string, expected any, err error) {
	v.addMismatch(check, source, expected, fmt.Sprintf("read error: %v", err))
}

func standardValidatorInput(
	dci opcm.DeployOPChainInput,
	contracts addresses.OpChainContracts,
) opcm.StandardValidatorInput {
	gameType := embedded.GameType(dci.DisputeGameType)
	useDevInput := gameType == embedded.GameTypeCannonKona || gameType == embedded.GameTypeSuperCannonKona
	return opcm.StandardValidatorInput{
		SystemConfig:        contracts.SystemConfigProxy,
		AbsolutePrestate:    dci.DisputeAbsolutePrestate,
		CannonPrestate:      dci.CannonAbsolutePrestate,
		CannonKonaPrestate:  dci.DisputeAbsolutePrestate,
		L2ChainID:           dci.L2ChainId,
		Proposer:            dci.Proposer,
		UseDevFeaturesInput: useDevInput,
	}
}

type namedAddress struct {
	name    string
	address common.Address
}

func continuationContractAddresses(contracts addresses.OpChainContracts) []namedAddress {
	return []namedAddress{
		{"OpChainProxyAdminImpl", contracts.OpChainProxyAdminImpl},
		{"OptimismPortalProxy", contracts.OptimismPortalProxy},
		{"AddressManagerImpl", contracts.AddressManagerImpl},
		{"L1Erc721BridgeProxy", contracts.L1Erc721BridgeProxy},
		{"SystemConfigProxy", contracts.SystemConfigProxy},
		{"OptimismMintableErc20FactoryProxy", contracts.OptimismMintableErc20FactoryProxy},
		{"L1StandardBridgeProxy", contracts.L1StandardBridgeProxy},
		{"L1CrossDomainMessengerProxy", contracts.L1CrossDomainMessengerProxy},
		{"EthLockboxProxy", contracts.EthLockboxProxy},
		{"DisputeGameFactoryProxy", contracts.DisputeGameFactoryProxy},
		{"AnchorStateRegistryProxy", contracts.AnchorStateRegistryProxy},
		{"FaultDisputeGameImpl", contracts.FaultDisputeGameImpl},
		{"FaultDisputeGameCannonKonaImpl", contracts.FaultDisputeGameCannonKonaImpl},
		{"PermissionedDisputeGameImpl", contracts.PermissionedDisputeGameImpl},
		{"DelayedWethPermissionedGameProxy", contracts.DelayedWethPermissionedGameProxy},
		{"DelayedWethPermissionlessGameProxy", contracts.DelayedWethPermissionlessGameProxy},
		{"AltDAChallengeProxy", contracts.AltDAChallengeProxy},
		{"AltDAChallengeImpl", contracts.AltDAChallengeImpl},
		{"L2OutputOracleProxy", contracts.L2OutputOracleProxy},
	}
}

var (
	continuationStartingAnchorMethod    = newContinuationViewMethod("getStartingAnchorRoot", nil, "bytes32", "uint256")
	continuationRespectedGameTypeMethod = newContinuationViewMethod("respectedGameType", nil, "uint32")
	continuationGameImplMethod          = newContinuationViewMethod("gameImpls", []string{"uint32"}, "address")
	continuationGameArgsMethod          = newContinuationViewMethod("gameArgs", []string{"uint32"}, "bytes")
	continuationOwnerMethod             = newContinuationViewMethod("owner", nil, "address")
	continuationBatcherHashMethod       = newContinuationViewMethod("batcherHash", nil, "bytes32")
	continuationUnsafeBlockSignerMethod = newContinuationViewMethod("unsafeBlockSigner", nil, "address")
	continuationGasLimitMethod          = newContinuationViewMethod("gasLimit", nil, "uint64")
	continuationL2ChainIDMethod         = newContinuationViewMethod("l2ChainId", nil, "uint256")
	continuationSuperchainConfigMethod  = newContinuationViewMethod("superchainConfig", nil, "address")
	continuationConfigMethod            = newContinuationViewMethod("config", nil, "address")
	continuationGuardianMethod          = newContinuationViewMethod("guardian", nil, "address")
	continuationStandardValidatorMethod = newContinuationViewMethod("opcmStandardValidator", nil, "address")
)

func newContinuationViewMethod(name string, inputTypes []string, outputTypes ...string) abi.Method {
	inputs := make(abi.Arguments, len(inputTypes))
	for i, inputType := range inputTypes {
		inputs[i] = abi.Argument{Type: opcm.MustType(inputType)}
	}
	outputs := make(abi.Arguments, len(outputTypes))
	for i, outputType := range outputTypes {
		outputs[i] = abi.Argument{Type: opcm.MustType(outputType)}
	}
	return abi.NewMethod(name, name, abi.Function, "view", true, false, inputs, outputs)
}

func callContinuationMethod(
	ctx context.Context,
	backend opcm.CallContractBackend,
	contract common.Address,
	method abi.Method,
	args ...any,
) ([]any, error) {
	input, err := method.Inputs.Pack(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to encode %s call: %w", method.Name, err)
	}
	calldata := append(bytes.Clone(method.ID), input...)
	result, err := backend.CallContract(ctx, ethereum.CallMsg{To: &contract, Data: calldata}, nil)
	if err != nil {
		return nil, fmt.Errorf("%s call to %s failed: %w", method.Name, contract, err)
	}
	values, err := method.Outputs.Unpack(result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode %s result from %s: %w", method.Name, contract, err)
	}
	return values, nil
}

func readContinuationOne(
	ctx context.Context,
	backend opcm.CallContractBackend,
	contract common.Address,
	method abi.Method,
	args ...any,
) (any, error) {
	values, err := callContinuationMethod(ctx, backend, contract, method, args...)
	if err != nil {
		return nil, err
	}
	if len(values) != 1 {
		return nil, fmt.Errorf("%s returned %d values", method.Name, len(values))
	}
	return values[0], nil
}

func readContinuationAddress(
	ctx context.Context,
	backend opcm.CallContractBackend,
	contract common.Address,
	method abi.Method,
	args ...any,
) (common.Address, error) {
	value, err := readContinuationOne(ctx, backend, contract, method, args...)
	if err != nil {
		return common.Address{}, err
	}
	address, ok := value.(common.Address)
	if !ok {
		return common.Address{}, fmt.Errorf("%s returned unexpected type %T", method.Name, value)
	}
	return address, nil
}

func readContinuationHash(
	ctx context.Context,
	backend opcm.CallContractBackend,
	contract common.Address,
	method abi.Method,
	args ...any,
) (common.Hash, error) {
	value, err := readContinuationOne(ctx, backend, contract, method, args...)
	if err != nil {
		return common.Hash{}, err
	}
	hash, ok := value.([common.HashLength]byte)
	if !ok {
		return common.Hash{}, fmt.Errorf("%s returned unexpected type %T", method.Name, value)
	}
	return common.Hash(hash), nil
}

func readContinuationBytes(
	ctx context.Context,
	backend opcm.CallContractBackend,
	contract common.Address,
	method abi.Method,
	args ...any,
) ([]byte, error) {
	value, err := readContinuationOne(ctx, backend, contract, method, args...)
	if err != nil {
		return nil, err
	}
	result, ok := value.([]byte)
	if !ok {
		return nil, fmt.Errorf("%s returned unexpected type %T", method.Name, value)
	}
	return result, nil
}

// These offsets mirror packages/contracts-bedrock/src/dispute/lib/LibGameArgs.sol.
func decodeContinuationGamePrestate(gameArgs []byte, permissionless bool) (common.Hash, error) {
	expectedLength := 164
	if permissionless {
		expectedLength = 124
	}
	if len(gameArgs) != expectedLength {
		return common.Hash{}, fmt.Errorf("configured game arguments have length %d, expected %d", len(gameArgs), expectedLength)
	}
	return common.BytesToHash(gameArgs[:common.HashLength]), nil
}

func decodeContinuationPermissionedRoles(
	gameArgs []byte,
	hasChallenger bool,
) (common.Address, common.Address, error) {
	if hasChallenger {
		if len(gameArgs) != 164 {
			return common.Address{}, common.Address{}, fmt.Errorf(
				"configured permissioned game arguments have length %d, expected 164",
				len(gameArgs),
			)
		}
		return common.BytesToAddress(gameArgs[124:144]), common.BytesToAddress(gameArgs[144:164]), nil
	}
	if len(gameArgs) != 40 {
		return common.Address{}, common.Address{}, fmt.Errorf(
			"configured super permissioned game arguments have length %d, expected 40",
			len(gameArgs),
		)
	}
	return common.BytesToAddress(gameArgs[20:40]), common.Address{}, nil
}

func readContinuationUint32(
	ctx context.Context,
	backend opcm.CallContractBackend,
	contract common.Address,
	method abi.Method,
	args ...any,
) (uint32, error) {
	value, err := readContinuationOne(ctx, backend, contract, method, args...)
	if err != nil {
		return 0, err
	}
	result, ok := value.(uint32)
	if !ok {
		return 0, fmt.Errorf("%s returned unexpected type %T", method.Name, value)
	}
	return result, nil
}

func readContinuationUint64(
	ctx context.Context,
	backend opcm.CallContractBackend,
	contract common.Address,
	method abi.Method,
	args ...any,
) (uint64, error) {
	value, err := readContinuationOne(ctx, backend, contract, method, args...)
	if err != nil {
		return 0, err
	}
	result, ok := value.(uint64)
	if !ok {
		return 0, fmt.Errorf("%s returned unexpected type %T", method.Name, value)
	}
	return result, nil
}

func readContinuationBig(
	ctx context.Context,
	backend opcm.CallContractBackend,
	contract common.Address,
	method abi.Method,
	args ...any,
) (*big.Int, error) {
	value, err := readContinuationOne(ctx, backend, contract, method, args...)
	if err != nil {
		return nil, err
	}
	result, ok := value.(*big.Int)
	if !ok {
		return nil, fmt.Errorf("%s returned unexpected type %T", method.Name, value)
	}
	return result, nil
}

func readContinuationAnchor(
	ctx context.Context,
	backend opcm.CallContractBackend,
	contract common.Address,
) (common.Hash, *big.Int, error) {
	values, err := callContinuationMethod(ctx, backend, contract, continuationStartingAnchorMethod)
	if err != nil {
		return common.Hash{}, nil, err
	}
	if len(values) != 2 {
		return common.Hash{}, nil, fmt.Errorf("getStartingAnchorRoot returned %d values", len(values))
	}
	root, ok := values[0].([common.HashLength]byte)
	if !ok {
		return common.Hash{}, nil, fmt.Errorf("getStartingAnchorRoot root returned unexpected type %T", values[0])
	}
	sequence, ok := values[1].(*big.Int)
	if !ok {
		return common.Hash{}, nil, fmt.Errorf("getStartingAnchorRoot sequence returned unexpected type %T", values[1])
	}
	return common.Hash(root), sequence, nil
}
