package deployer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
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
	selectorGameType        uint32
	respectedGameType       uint32
	respectedImplementation common.Address
	fallbackGameType        *uint32
	fallbackImplementation  common.Address
	permissionless          bool
	hasChallenger           bool
}

type continuationGameArgsLayout uint8

const (
	continuationPermissionlessGameArgs continuationGameArgsLayout = iota
	continuationPermissionedGameArgs
	continuationSuperPermissionedGameArgs
)

type continuationGameArgs struct {
	absolutePrestate    common.Hash
	vm                  common.Address
	anchorStateRegistry common.Address
	delayedWETH         common.Address
	l2ChainID           *big.Int
	proposer            common.Address
	challenger          common.Address
}

type continuationOPCMImplementations struct {
	SuperchainConfigImpl             common.Address
	L1ERC721BridgeImpl               common.Address
	OptimismPortalImpl               common.Address
	ETHLockboxImpl                   common.Address `abi:"ethLockboxImpl"`
	SystemConfigImpl                 common.Address
	OptimismMintableERC20FactoryImpl common.Address
	L1CrossDomainMessengerImpl       common.Address
	L1StandardBridgeImpl             common.Address
	DisputeGameFactoryImpl           common.Address
	AnchorStateRegistryImpl          common.Address
	DelayedWETHImpl                  common.Address
	MipsImpl                         common.Address
	FaultDisputeGameImpl             common.Address
	PermissionedDisputeGameImpl      common.Address
	SuperFaultDisputeGameImpl        common.Address
	SuperPermissionedDisputeGameImpl common.Address
	ZkDisputeGameImpl                common.Address
	StorageSetterImpl                common.Address
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

	bitmap, err := readContinuationHash(
		v.ctx,
		v.backend,
		v.dci.Opcm,
		continuationDevFeatureBitmapMethod,
	)
	if err != nil {
		return continuationGameMode{}, fmt.Errorf("failed to read pinned OPCM dev feature bitmap: %w", err)
	}
	superRoot := devfeatures.IsDevFeatureEnabled(bitmap, devfeatures.SuperRootGamesMigrationFlag)

	mode := continuationGameMode{
		selectorGameType:  gameType,
		respectedGameType: gameType,
	}
	switch embedded.GameType(gameType) {
	case embedded.GameTypePermissionedCannon:
		if superRoot {
			mode.respectedGameType = uint32(embedded.GameTypeSuperPermissioned)
		}
		mode.respectedImplementation = v.expected.PermissionedDisputeGameImpl
		mode.hasChallenger = !superRoot
	case embedded.GameTypeCannonKona:
		if superRoot {
			return continuationGameMode{}, fmt.Errorf(
				"initial game type mismatch: frozen selector %d requires an OPCM without SUPER_ROOT_GAMES_MIGRATION",
				gameType,
			)
		}
		fallback := uint32(embedded.GameTypePermissionedCannon)
		mode.respectedImplementation = v.expected.FaultDisputeGameImpl
		mode.fallbackGameType = &fallback
		mode.fallbackImplementation = v.expected.PermissionedDisputeGameImpl
		mode.permissionless = true
		mode.hasChallenger = true
	case embedded.GameTypeSuperCannonKona:
		if !superRoot {
			return continuationGameMode{}, fmt.Errorf(
				"initial game type mismatch: frozen selector %d requires an OPCM with SUPER_ROOT_GAMES_MIGRATION",
				gameType,
			)
		}
		fallback := uint32(embedded.GameTypeSuperPermissioned)
		mode.respectedImplementation = v.expected.FaultDisputeGameImpl
		mode.fallbackGameType = &fallback
		mode.fallbackImplementation = v.expected.PermissionedDisputeGameImpl
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
			"prepared selector and pinned OPCM dev feature bitmap",
			mode.respectedGameType,
			err,
		)
	} else if respected != mode.respectedGameType {
		v.addMismatch(
			"respected game type",
			"prepared selector and pinned OPCM dev feature bitmap",
			mode.respectedGameType,
			respected,
		)
	}

	initialImplementation, err := readContinuationAddress(
		v.ctx,
		v.backend,
		v.expected.DisputeGameFactoryProxy,
		continuationGameImplMethod,
		mode.respectedGameType,
	)
	if err != nil {
		v.addReadError(
			"initial game implementation",
			"predicted ChainState.OpChainContracts",
			mode.respectedImplementation,
			err,
		)
	} else if initialImplementation != mode.respectedImplementation {
		v.addMismatch(
			"initial game implementation",
			"predicted ChainState.OpChainContracts",
			mode.respectedImplementation,
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
	if mode.permissionless {
		expectedPrestate = v.expected.Prestate
		if v.dci.DisputeAbsolutePrestate != expectedPrestate {
			v.addMismatch(
				"selected prestate input",
				"committed ChainState.Prestate",
				expectedPrestate,
				v.dci.DisputeAbsolutePrestate,
			)
		}
	}

	initialLayout := continuationPermissionlessGameArgs
	if !mode.permissionless {
		initialLayout = continuationPermissionedGameArgs
		if !mode.hasChallenger {
			initialLayout = continuationSuperPermissionedGameArgs
		}
	}
	var expectedVM *common.Address
	if initialLayout != continuationSuperPermissionedGameArgs {
		implementations, err := readContinuationOPCMImplementations(v.ctx, v.backend, v.dci.Opcm)
		if err != nil {
			v.addReadError(
				"pinned OPCM implementations",
				"frozen DeployOPChainInput.Opcm",
				"a readable MIPS implementation",
				err,
			)
		} else {
			expectedVM = &implementations.MipsImpl
		}
	}
	v.verifyConfiguredGameArgs(
		"selected",
		mode.respectedGameType,
		initialLayout,
		expectedPrestate,
		expectedVM,
	)

	switch embedded.GameType(mode.selectorGameType) {
	case embedded.GameTypeCannonKona:
		if v.dci.CannonAbsolutePrestate != opcm.PermissionedCannonFallbackPrestatePlaceholder {
			v.addMismatch(
				"fallback prestate input",
				"fixed CANNON_KONA permissioned fallback",
				opcm.PermissionedCannonFallbackPrestatePlaceholder,
				v.dci.CannonAbsolutePrestate,
			)
		}
		v.verifyConfiguredGameArgs(
			"fallback",
			*mode.fallbackGameType,
			continuationPermissionedGameArgs,
			opcm.PermissionedCannonFallbackPrestatePlaceholder,
			expectedVM,
		)
	case embedded.GameTypeSuperCannonKona:
		if v.dci.CannonAbsolutePrestate != (common.Hash{}) {
			v.addMismatch(
				"fallback prestate input",
				"SUPER_CANNON_KONA no-prestate fallback rule",
				common.Hash{},
				v.dci.CannonAbsolutePrestate,
			)
		}
		v.verifyConfiguredGameArgs(
			"fallback",
			*mode.fallbackGameType,
			continuationSuperPermissionedGameArgs,
			common.Hash{},
			nil,
		)
	}
}

func (v *continuationVerifier) verifyConfiguredGameArgs(
	label string,
	gameType uint32,
	layout continuationGameArgsLayout,
	expectedPrestate common.Hash,
	expectedVM *common.Address,
) {
	gameArgs, err := readContinuationBytes(
		v.ctx,
		v.backend,
		v.expected.DisputeGameFactoryProxy,
		continuationGameArgsMethod,
		gameType,
	)
	if err != nil {
		v.addReadError(
			label+" game arguments",
			"frozen DeployOPChainInput and predicted ChainState.OpChainContracts",
			"readable configured game arguments",
			err,
		)
		return
	}
	decoded, err := decodeContinuationGameArgs(gameArgs, layout)
	if err != nil {
		v.addReadError(
			label+" game arguments",
			"frozen DeployOPChainInput and predicted ChainState.OpChainContracts",
			"valid configured game arguments",
			err,
		)
		return
	}

	if decoded.anchorStateRegistry != v.expected.AnchorStateRegistryProxy {
		v.addMismatch(
			label+" game AnchorStateRegistry",
			"predicted ChainState.OpChainContracts.AnchorStateRegistryProxy",
			v.expected.AnchorStateRegistryProxy,
			decoded.anchorStateRegistry,
		)
	}

	if layout == continuationSuperPermissionedGameArgs {
		if decoded.proposer != v.dci.Proposer {
			v.addMismatch(
				label+" game proposer",
				"frozen DeployOPChainInput.Proposer",
				v.dci.Proposer,
				decoded.proposer,
			)
		}
		return
	}

	if decoded.absolutePrestate != expectedPrestate {
		v.addMismatch(
			label+" game prestate",
			"frozen/committed game prestate",
			expectedPrestate,
			decoded.absolutePrestate,
		)
	}

	if expectedVM != nil && decoded.vm != *expectedVM {
		v.addMismatch(label+" game VM", "pinned OPCM implementations", *expectedVM, decoded.vm)
	}

	expectedWETH := v.expected.DelayedWethPermissionedGameProxy
	if layout == continuationPermissionlessGameArgs {
		expectedWETH = v.expected.DelayedWethPermissionlessGameProxy
	}
	if decoded.delayedWETH != expectedWETH {
		v.addMismatch(
			label+" game DelayedWETH",
			"predicted ChainState.OpChainContracts",
			expectedWETH,
			decoded.delayedWETH,
		)
	}

	expectedL2ChainID := v.dci.L2ChainId
	if embedded.GameType(gameType) == embedded.GameTypeSuperCannonKona {
		expectedL2ChainID = new(big.Int)
	}
	if expectedL2ChainID == nil || decoded.l2ChainID.Cmp(expectedL2ChainID) != 0 {
		v.addMismatch(
			label+" game L2 chain ID",
			"frozen DeployOPChainInput.L2ChainId and game-type rules",
			expectedL2ChainID,
			decoded.l2ChainID,
		)
	}

	if layout == continuationPermissionedGameArgs {
		if decoded.proposer != v.dci.Proposer {
			v.addMismatch(
				label+" game proposer",
				"frozen DeployOPChainInput.Proposer",
				v.dci.Proposer,
				decoded.proposer,
			)
		}
		if decoded.challenger != v.dci.Challenger {
			v.addMismatch(
				label+" game challenger",
				"frozen DeployOPChainInput.Challenger",
				v.dci.Challenger,
				decoded.challenger,
			)
		}
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
	v.verifyUint32Getter(
		"SystemConfig base fee scalar",
		"frozen DeployOPChainInput.BasefeeScalar",
		systemConfig,
		continuationBasefeeScalarMethod,
		v.dci.BasefeeScalar,
	)
	v.verifyUint32Getter(
		"SystemConfig blob base fee scalar",
		"frozen DeployOPChainInput.BlobBaseFeeScalar",
		systemConfig,
		continuationBlobBasefeeScalarMethod,
		v.dci.BlobBaseFeeScalar,
	)
	v.verifyUint32Getter(
		"SystemConfig operator fee scalar",
		"frozen DeployOPChainInput.OperatorFeeScalar",
		systemConfig,
		continuationOperatorFeeScalarMethod,
		v.dci.OperatorFeeScalar,
	)
	v.verifyUint64Getter(
		"SystemConfig operator fee constant",
		"frozen DeployOPChainInput.OperatorFeeConstant",
		systemConfig,
		continuationOperatorFeeConstantMethod,
		v.dci.OperatorFeeConstant,
	)
	v.verifyBoolGetter(
		"SystemConfig custom gas token mode",
		"frozen DeployOPChainInput.UseCustomGasToken",
		systemConfig,
		continuationIsCustomGasTokenMethod,
		v.dci.UseCustomGasToken,
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

func (v *continuationVerifier) verifyUint32Getter(
	check string,
	source string,
	contract common.Address,
	method abi.Method,
	expected uint32,
) {
	observed, err := readContinuationUint32(v.ctx, v.backend, contract, method)
	if err != nil {
		v.addReadError(check, source, expected, err)
	} else if observed != expected {
		v.addMismatch(check, source, expected, observed)
	}
}

func (v *continuationVerifier) verifyUint64Getter(
	check string,
	source string,
	contract common.Address,
	method abi.Method,
	expected uint64,
) {
	observed, err := readContinuationUint64(v.ctx, v.backend, contract, method)
	if err != nil {
		v.addReadError(check, source, expected, err)
	} else if observed != expected {
		v.addMismatch(check, source, expected, observed)
	}
}

func (v *continuationVerifier) verifyBoolGetter(
	check string,
	source string,
	contract common.Address,
	method abi.Method,
	expected bool,
) {
	observed, err := readContinuationBool(v.ctx, v.backend, contract, method)
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
		L1PAOMultisig:       dci.OpChainProxyAdminOwner,
		Challenger:          dci.Challenger,
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
	continuationStartingAnchorMethod      = newContinuationViewMethod("getStartingAnchorRoot", nil, "bytes32", "uint256")
	continuationRespectedGameTypeMethod   = newContinuationViewMethod("respectedGameType", nil, "uint32")
	continuationGameImplMethod            = newContinuationViewMethod("gameImpls", []string{"uint32"}, "address")
	continuationGameArgsMethod            = newContinuationViewMethod("gameArgs", []string{"uint32"}, "bytes")
	continuationImplementationsMethod     = newContinuationImplementationsMethod()
	continuationOwnerMethod               = newContinuationViewMethod("owner", nil, "address")
	continuationBasefeeScalarMethod       = newContinuationViewMethod("basefeeScalar", nil, "uint32")
	continuationBlobBasefeeScalarMethod   = newContinuationViewMethod("blobbasefeeScalar", nil, "uint32")
	continuationBatcherHashMethod         = newContinuationViewMethod("batcherHash", nil, "bytes32")
	continuationUnsafeBlockSignerMethod   = newContinuationViewMethod("unsafeBlockSigner", nil, "address")
	continuationGasLimitMethod            = newContinuationViewMethod("gasLimit", nil, "uint64")
	continuationL2ChainIDMethod           = newContinuationViewMethod("l2ChainId", nil, "uint256")
	continuationOperatorFeeScalarMethod   = newContinuationViewMethod("operatorFeeScalar", nil, "uint32")
	continuationOperatorFeeConstantMethod = newContinuationViewMethod("operatorFeeConstant", nil, "uint64")
	continuationIsCustomGasTokenMethod    = newContinuationViewMethod("isCustomGasToken", nil, "bool")
	continuationSuperchainConfigMethod    = newContinuationViewMethod("superchainConfig", nil, "address")
	continuationConfigMethod              = newContinuationViewMethod("config", nil, "address")
	continuationGuardianMethod            = newContinuationViewMethod("guardian", nil, "address")
	continuationStandardValidatorMethod   = newContinuationViewMethod("opcmStandardValidator", nil, "address")
	continuationDevFeatureBitmapMethod    = newContinuationViewMethod("devFeatureBitmap", nil, "bytes32")
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

func newContinuationImplementationsMethod() abi.Method {
	components := []abi.ArgumentMarshaling{
		{Name: "superchainConfigImpl", Type: "address"},
		{Name: "l1ERC721BridgeImpl", Type: "address"},
		{Name: "optimismPortalImpl", Type: "address"},
		{Name: "ethLockboxImpl", Type: "address"},
		{Name: "systemConfigImpl", Type: "address"},
		{Name: "optimismMintableERC20FactoryImpl", Type: "address"},
		{Name: "l1CrossDomainMessengerImpl", Type: "address"},
		{Name: "l1StandardBridgeImpl", Type: "address"},
		{Name: "disputeGameFactoryImpl", Type: "address"},
		{Name: "anchorStateRegistryImpl", Type: "address"},
		{Name: "delayedWETHImpl", Type: "address"},
		{Name: "mipsImpl", Type: "address"},
		{Name: "faultDisputeGameImpl", Type: "address"},
		{Name: "permissionedDisputeGameImpl", Type: "address"},
		{Name: "superFaultDisputeGameImpl", Type: "address"},
		{Name: "superPermissionedDisputeGameImpl", Type: "address"},
		{Name: "zkDisputeGameImpl", Type: "address"},
		{Name: "storageSetterImpl", Type: "address"},
	}
	implementationsType, err := abi.NewType("tuple", "", components)
	if err != nil {
		panic(fmt.Errorf("failed to define OPCM implementations ABI type: %w", err))
	}
	return abi.NewMethod(
		"implementations",
		"implementations",
		abi.Function,
		"view",
		true,
		false,
		nil,
		abi.Arguments{{Type: implementationsType}},
	)
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

func readContinuationOPCMImplementations(
	ctx context.Context,
	backend opcm.CallContractBackend,
	opcmAddress common.Address,
) (continuationOPCMImplementations, error) {
	value, err := readContinuationOne(ctx, backend, opcmAddress, continuationImplementationsMethod)
	if err != nil {
		return continuationOPCMImplementations{}, err
	}
	implementations, ok := abi.ConvertType(value, new(continuationOPCMImplementations)).(*continuationOPCMImplementations)
	if !ok {
		return continuationOPCMImplementations{}, fmt.Errorf(
			"implementations returned unexpected type %T",
			value,
		)
	}
	return *implementations, nil
}

// These offsets mirror packages/contracts-bedrock/src/dispute/lib/LibGameArgs.sol.
func decodeContinuationGameArgs(
	gameArgs []byte,
	layout continuationGameArgsLayout,
) (continuationGameArgs, error) {
	if layout == continuationSuperPermissionedGameArgs {
		if len(gameArgs) != 40 {
			return continuationGameArgs{}, fmt.Errorf(
				"configured super permissioned game arguments have length %d, expected 40",
				len(gameArgs),
			)
		}
		return continuationGameArgs{
			anchorStateRegistry: common.BytesToAddress(gameArgs[:20]),
			proposer:            common.BytesToAddress(gameArgs[20:40]),
		}, nil
	}

	expectedLength := 124
	if layout == continuationPermissionedGameArgs {
		expectedLength = 164
	} else if layout != continuationPermissionlessGameArgs {
		return continuationGameArgs{}, fmt.Errorf("unknown continuation game argument layout %d", layout)
	}
	if len(gameArgs) != expectedLength {
		return continuationGameArgs{}, fmt.Errorf(
			"configured game arguments have length %d, expected %d",
			len(gameArgs),
			expectedLength,
		)
	}

	decoded := continuationGameArgs{
		absolutePrestate:    common.BytesToHash(gameArgs[:32]),
		vm:                  common.BytesToAddress(gameArgs[32:52]),
		anchorStateRegistry: common.BytesToAddress(gameArgs[52:72]),
		delayedWETH:         common.BytesToAddress(gameArgs[72:92]),
		l2ChainID:           new(big.Int).SetBytes(gameArgs[92:124]),
	}
	if layout == continuationPermissionedGameArgs {
		decoded.proposer = common.BytesToAddress(gameArgs[124:144])
		decoded.challenger = common.BytesToAddress(gameArgs[144:164])
	}
	return decoded, nil
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

func readContinuationBool(
	ctx context.Context,
	backend opcm.CallContractBackend,
	contract common.Address,
	method abi.Method,
	args ...any,
) (bool, error) {
	value, err := readContinuationOne(ctx, backend, contract, method, args...)
	if err != nil {
		return false, err
	}
	result, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("%s returned unexpected type %T", method.Name, value)
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
