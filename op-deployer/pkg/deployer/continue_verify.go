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
	opeth "github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/lmittmann/w3"
)

type continuationReadBackend interface {
	opcm.CallContractBackend
	CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error)
}

type continuationHeaderBackend interface {
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
}

func verifyContinuationHead(
	ctx context.Context,
	backend continuationHeaderBackend,
	number *big.Int,
	expectedHash common.Hash,
) error {
	header, err := backend.HeaderByNumber(ctx, number)
	if err != nil {
		return fmt.Errorf("failed to recheck L1 block %s: %w", number, err)
	}
	if header == nil || header.Hash() != expectedHash {
		var observed common.Hash
		if header != nil {
			observed = header.Hash()
		}
		return fmt.Errorf(
			"L1 block %s changed during continuation reads: expected %s, observed %s",
			number,
			expectedHash,
			observed,
		)
	}
	return nil
}

type pinnedContinuationReadBackend struct {
	continuationReadBackend
	blockNumber *big.Int
}

func (b *pinnedContinuationReadBackend) CallContract(
	ctx context.Context,
	call ethereum.CallMsg,
	_ *big.Int,
) ([]byte, error) {
	return b.continuationReadBackend.CallContract(ctx, call, b.blockNumber)
}

func (b *pinnedContinuationReadBackend) CodeAt(
	ctx context.Context,
	contract common.Address,
	_ *big.Int,
) ([]byte, error) {
	return b.continuationReadBackend.CodeAt(ctx, contract, b.blockNumber)
}

func pinContinuationReads(
	ctx context.Context,
	backend continuationReadBackend,
) (continuationReadBackend, func() error, error) {
	headRead, ok := backend.(continuationHeaderBackend)
	if !ok {
		return backend, func() error { return nil }, nil
	}
	head, err := headRead.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	if head == nil || head.Number == nil {
		return nil, nil, fmt.Errorf("current L1 head is missing its block number")
	}
	number := new(big.Int).Set(head.Number)
	hash := head.Hash()
	return &pinnedContinuationReadBackend{
			continuationReadBackend: backend,
			blockNumber:             number,
		}, func() error {
			return verifyContinuationHead(ctx, headRead, number, hash)
		}, nil
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
	respectedGameType       uint32
	respectedImplementation common.Address
	fallbackGameType        *uint32
	fallbackImplementation  common.Address
	permissionless          bool
	hasChallenger           bool
}

type continuationGameArgsLayout uint8

const (
	continuationPermissionedGameArgs continuationGameArgsLayout = iota
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
	SP1PlonkAdapterImpl              common.Address `abi:"sp1PlonkAdapterImpl"`
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
	pinnedBackend, verifyHead, err := pinContinuationReads(ctx, backend)
	if err != nil {
		return fmt.Errorf("failed to pin continuation verification reads: %w", err)
	}

	verifier := &continuationVerifier{
		ctx:      ctx,
		backend:  pinnedBackend,
		observed: observed,
		expected: expected,
		dci:      dci,
	}
	result := func() error { return errors.Join(errors.Join(verifier.failures...), verifyHead()) }
	if !verifier.verifyAddressesAndCode() {
		return result()
	}
	verifier.verifyStartingAnchorRoot()

	mode, err := verifier.resolveGameMode()
	if err != nil {
		verifier.failures = append(verifier.failures, err)
	} else {
		implementations := verifier.verifyGameConfiguration(mode)
		if !mode.permissionless {
			verifier.verifyProxyAdminOwner()
		}
		verifier.verifyPersistentWiring(implementations)
	}
	verifier.verifySystemConfig()
	verifier.verifySuperchainConfig()
	// Only verify using the validator on permissionless mode.
	// The validator expects the permissionless game to exist,
	// so it would always fail on a permissioned chain.
	if mode.permissionless {
		verifier.verifyStandardValidator()
	}
	return result()
}

func (v *continuationVerifier) resolveGameMode() (continuationGameMode, error) {
	gameType := v.dci.DisputeGameType

	bitmap, err := readContinuation[common.Hash](
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
		respectedGameType: gameType,
	}
	switch embedded.GameType(gameType) {
	// DeployOPChain.s.sol requires GameTypes.isSuperGame(gameType) == isSuperRoot, so each
	// selector is valid for exactly one OPCM family.
	case embedded.GameTypePermissionedCannon:
		if superRoot {
			return continuationGameMode{}, fmt.Errorf(
				"initial game type mismatch: frozen selector %d requires an OPCM without SUPER_ROOT_GAMES_MIGRATION",
				gameType,
			)
		}
		mode.respectedImplementation = v.expected.PermissionedDisputeGameImpl
		mode.hasChallenger = true
	case embedded.GameTypeSuperPermissioned:
		if !superRoot {
			return continuationGameMode{}, fmt.Errorf(
				"initial game type mismatch: frozen selector %d requires an OPCM with SUPER_ROOT_GAMES_MIGRATION",
				gameType,
			)
		}
		mode.respectedImplementation = v.expected.PermissionedDisputeGameImpl
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
			"initial game type mismatch: expected a supported value from frozen DeployOPChainInput, observed %d",
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

func (v *continuationVerifier) verifyStartingAnchorRoot() {
	observed, err := readContinuation[opcm.Proposal](
		v.ctx,
		v.backend,
		v.expected.AnchorStateRegistryProxy,
		continuationStartingAnchorRootMethod,
	)
	if err != nil {
		v.addReadError(
			"starting anchor proposal",
			"frozen DeployOPChainInput.StartingAnchorRoot",
			v.dci.StartingAnchorRoot,
			err,
		)
		return
	}
	if observed.Root != v.dci.StartingAnchorRoot.Root {
		v.addMismatch(
			"starting anchor root",
			"frozen DeployOPChainInput.StartingAnchorRoot.Root",
			v.dci.StartingAnchorRoot.Root,
			observed.Root,
		)
	}
	expectedSequence := v.dci.StartingAnchorRoot.L2SequenceNumber
	if expectedSequence == nil || observed.L2SequenceNumber == nil || observed.L2SequenceNumber.Cmp(expectedSequence) != 0 {
		v.addMismatch(
			"starting anchor sequence number",
			"frozen DeployOPChainInput.StartingAnchorRoot.L2SequenceNumber",
			expectedSequence,
			observed.L2SequenceNumber,
		)
	}
}

func (v *continuationVerifier) verifyGameConfiguration(
	mode continuationGameMode,
) *continuationOPCMImplementations {
	respected, err := readContinuation[uint32](
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

	initialImplementation, err := readContinuation[common.Address](
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
		fallbackImplementation, err := readContinuation[common.Address](
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

	if mode.permissionless {
		return nil
	}

	initialLayout := continuationPermissionedGameArgs
	if !mode.hasChallenger {
		initialLayout = continuationSuperPermissionedGameArgs
	}
	var expectedVM *common.Address
	var implementations *continuationOPCMImplementations
	observed, err := readContinuationOPCMImplementations(v.ctx, v.backend, v.dci.Opcm)
	if err != nil {
		v.addReadError(
			"pinned OPCM implementations",
			"frozen DeployOPChainInput.Opcm",
			"a readable implementation set",
			err,
		)
	} else {
		implementations = &observed
		if initialLayout == continuationPermissionedGameArgs {
			expectedVM = &observed.MipsImpl
		}
	}
	v.verifyConfiguredGameArgs(
		"selected",
		mode.respectedGameType,
		initialLayout,
		v.dci.DisputeAbsolutePrestate,
		expectedVM,
	)
	return implementations
}

func (v *continuationVerifier) verifyConfiguredGameArgs(
	label string,
	gameType uint32,
	layout continuationGameArgsLayout,
	expectedPrestate common.Hash,
	expectedVM *common.Address,
) {
	gameArgs, err := readContinuation[[]byte](
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
	if decoded.delayedWETH != expectedWETH {
		v.addMismatch(
			label+" game DelayedWETH",
			"predicted ChainState.OpChainContracts",
			expectedWETH,
			decoded.delayedWETH,
		)
	}

	expectedL2ChainID := v.dci.L2ChainId
	if expectedL2ChainID == nil || decoded.l2ChainID.Cmp(expectedL2ChainID) != 0 {
		v.addMismatch(
			label+" game L2 chain ID",
			"frozen DeployOPChainInput.L2ChainId and game-type rules",
			expectedL2ChainID,
			decoded.l2ChainID,
		)
	}

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

func (v *continuationVerifier) verifySystemConfig() {
	systemConfig := v.expected.SystemConfigProxy
	verifyContinuationExpectations(v, []continuationExpectation[common.Address]{
		{"SystemConfig owner", "frozen DeployOPChainInput.SystemConfigOwner", systemConfig, continuationOwnerMethod, nil, v.dci.SystemConfigOwner},
		{"SystemConfig unsafe block signer", "frozen DeployOPChainInput.UnsafeBlockSigner", systemConfig, continuationUnsafeBlockSignerMethod, nil, v.dci.UnsafeBlockSigner},
	})
	verifyContinuationExpectations(v, []continuationExpectation[uint32]{
		{"SystemConfig base fee scalar", "frozen DeployOPChainInput.BasefeeScalar", systemConfig, continuationBasefeeScalarMethod, nil, v.dci.BasefeeScalar},
		{"SystemConfig blob base fee scalar", "frozen DeployOPChainInput.BlobBaseFeeScalar", systemConfig, continuationBlobBasefeeScalarMethod, nil, v.dci.BlobBaseFeeScalar},
		{"SystemConfig operator fee scalar", "frozen DeployOPChainInput.OperatorFeeScalar", systemConfig, continuationOperatorFeeScalarMethod, nil, v.dci.OperatorFeeScalar},
	})
	verifyContinuationExpectations(v, []continuationExpectation[uint64]{
		{"SystemConfig operator fee constant", "frozen DeployOPChainInput.OperatorFeeConstant", systemConfig, continuationOperatorFeeConstantMethod, nil, v.dci.OperatorFeeConstant},
		{"SystemConfig gas limit", "frozen DeployOPChainInput.GasLimit", systemConfig, continuationGasLimitMethod, nil, v.dci.GasLimit},
	})
	verifyContinuationExpectations(v, []continuationExpectation[bool]{
		{"SystemConfig custom gas token mode", "frozen DeployOPChainInput.UseCustomGasToken", systemConfig, continuationIsCustomGasTokenMethod, nil, v.dci.UseCustomGasToken},
	})
	verifyContinuationExpectations(v, []continuationExpectation[common.Hash]{
		{"SystemConfig batcher", "frozen DeployOPChainInput.Batcher", systemConfig, continuationBatcherHashMethod, nil, common.BytesToHash(v.dci.Batcher.Bytes())},
	})

	l2ChainID, err := readContinuation[*big.Int](
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
	verifyContinuationExpectations(v, []continuationExpectation[common.Address]{
		{"OpChain ProxyAdmin owner", "frozen DeployOPChainInput.OpChainProxyAdminOwner", v.expected.OpChainProxyAdminImpl, continuationOwnerMethod, nil, v.dci.OpChainProxyAdminOwner},
	})
}

// Recheck persistent assertions from the deployment's ChainAssertions.sol.
//
// DeployOPChain.s.sol already runs those Solidity assertions through checkOutput, but
// only inside the forked preflight simulation. The checks below also run against live L1.
//
// Deployment-only state cannot be verified during continuation. StandardValidator
// covers permissionless implementation wiring; permissioned wiring is checked directly.
func (v *continuationVerifier) verifyPersistentWiring(implementations *continuationOPCMImplementations) {
	verifyContinuationExpectations(v, v.persistentAddressExpectations(implementations))

	scalar := opeth.EncodeScalar(opeth.EcotoneScalars{
		BlobBaseFeeScalar: v.dci.BlobBaseFeeScalar,
		BaseFeeScalar:     v.dci.BasefeeScalar,
	})
	observedScalar, err := readContinuation[*big.Int](
		v.ctx,
		v.backend,
		v.expected.SystemConfigProxy,
		continuationScalarMethod,
	)
	expectedScalar := new(big.Int).SetBytes(scalar[:])
	if err != nil {
		v.addReadError("CHECK-SCFG-70 SystemConfig scalar", "frozen DeployOPChainInput fee scalars", expectedScalar, err)
	} else if observedScalar.Cmp(expectedScalar) != 0 {
		v.addMismatch("CHECK-SCFG-70 SystemConfig scalar", "frozen DeployOPChainInput fee scalars", expectedScalar, observedScalar)
	}

	paused, err := readContinuation[bool](
		v.ctx,
		v.backend,
		v.expected.SystemConfigProxy,
		continuationPausedMethod,
	)
	if err != nil {
		v.addReadError("CHECK-OP2-60 SystemConfig paused state", "SystemConfig", "a readable value", err)
	} else {
		verifyContinuationExpectations(v, []continuationExpectation[bool]{
			{"CHECK-OP2-60 OptimismPortal paused state", "SystemConfig.paused()", v.expected.OptimismPortalProxy, continuationPausedMethod, nil, paused},
		})
	}

	lockbox, err := readContinuation[common.Address](
		v.ctx,
		v.backend,
		v.expected.OptimismPortalProxy,
		continuationEthLockboxMethod,
	)
	if err != nil {
		v.addReadError("CHECK-OP2-80 OptimismPortal ETHLockbox", "predicted OpChainContracts", v.expected.EthLockboxProxy, err)
	} else if lockbox != (common.Address{}) && lockbox != v.expected.EthLockboxProxy {
		v.addMismatch("CHECK-OP2-80 OptimismPortal ETHLockbox", "predicted OpChainContracts or disabled feature", v.expected.EthLockboxProxy, lockbox)
	}
}

func (v *continuationVerifier) persistentAddressExpectations(
	implementations *continuationOPCMImplementations,
) []continuationExpectation[common.Address] {
	contracts := v.expected.OpChainContracts
	expectations := []continuationExpectation[common.Address]{
		{"CHECK-SCFG-160 SystemConfig L1CrossDomainMessenger", "predicted OpChainContracts", contracts.SystemConfigProxy, continuationL1XDMMethod, nil, contracts.L1CrossDomainMessengerProxy},
		{"CHECK-SCFG-170 SystemConfig L1ERC721Bridge", "predicted OpChainContracts", contracts.SystemConfigProxy, continuationL1ERC721BridgeMethod, nil, contracts.L1Erc721BridgeProxy},
		{"CHECK-SCFG-180 SystemConfig L1StandardBridge", "predicted OpChainContracts", contracts.SystemConfigProxy, continuationL1StandardBridgeMethod, nil, contracts.L1StandardBridgeProxy},
		{"CHECK-SCFG-200 SystemConfig OptimismPortal", "predicted OpChainContracts", contracts.SystemConfigProxy, continuationOptimismPortalMethod, nil, contracts.OptimismPortalProxy},
		{"CHECK-SCFG-210 SystemConfig mintable factory", "predicted OpChainContracts", contracts.SystemConfigProxy, continuationMintableFactoryMethod, nil, contracts.OptimismMintableErc20FactoryProxy},
		{"CHECK-OP2-25 OptimismPortal AnchorStateRegistry", "predicted OpChainContracts", contracts.OptimismPortalProxy, continuationAnchorRegistryMethod, nil, contracts.AnchorStateRegistryProxy},
		{"CHECK-OP2-90 OptimismPortal ProxyAdmin owner", "frozen DeployOPChainInput.OpChainProxyAdminOwner", contracts.OptimismPortalProxy, continuationProxyAdminOwnerMethod, nil, v.dci.OpChainProxyAdminOwner},
		{"AM-10 AddressManager owner", "predicted OpChainContracts.OpChainProxyAdminImpl", contracts.AddressManagerImpl, continuationOwnerMethod, nil, contracts.OpChainProxyAdminImpl},
		{"OPCPA-30 ProxyAdmin AddressManager", "predicted OpChainContracts.AddressManagerImpl", contracts.OpChainProxyAdminImpl, continuationAddressManagerMethod, nil, contracts.AddressManagerImpl},
	}
	if implementations == nil {
		return expectations
	}
	for _, proxy := range []struct {
		check          string
		address        common.Address
		implementation common.Address
	}{
		{"OPCPA-20 L1CrossDomainMessenger implementation", contracts.L1CrossDomainMessengerProxy, implementations.L1CrossDomainMessengerImpl},
		{"OPCPA-40 L1StandardBridge implementation", contracts.L1StandardBridgeProxy, implementations.L1StandardBridgeImpl},
		{"OPCPA-50 L1ERC721Bridge implementation", contracts.L1Erc721BridgeProxy, implementations.L1ERC721BridgeImpl},
		{"OPCPA-60 OptimismPortal implementation", contracts.OptimismPortalProxy, implementations.OptimismPortalImpl},
		{"OPCPA-70 SystemConfig implementation", contracts.SystemConfigProxy, implementations.SystemConfigImpl},
		{"OPCPA-80 mintable factory implementation", contracts.OptimismMintableErc20FactoryProxy, implementations.OptimismMintableERC20FactoryImpl},
		{"OPCPA-90 DisputeGameFactory implementation", contracts.DisputeGameFactoryProxy, implementations.DisputeGameFactoryImpl},
		{"OPCPA-100 DelayedWETH implementation", contracts.DelayedWethPermissionedGameProxy, implementations.DelayedWETHImpl},
		{"OPCPA-110 AnchorStateRegistry implementation", contracts.AnchorStateRegistryProxy, implementations.AnchorStateRegistryImpl},
		{"OPCPA-120 ETHLockbox implementation", contracts.EthLockboxProxy, implementations.ETHLockboxImpl},
	} {
		expectations = append(expectations, continuationExpectation[common.Address]{
			proxy.check,
			"pinned OPCM implementations",
			contracts.OpChainProxyAdminImpl,
			continuationProxyImplementationMethod,
			[]any{proxy.address},
			proxy.implementation,
		})
	}
	return expectations
}

func (v *continuationVerifier) verifySuperchainConfig() {
	attachments := []continuationExpectation[common.Address]{
		{"SystemConfig SuperchainConfig attachment", "frozen DeployOPChainInput.SuperchainConfig", v.expected.SystemConfigProxy, continuationSuperchainConfigMethod, nil, v.dci.SuperchainConfig},
		{"OptimismPortal SuperchainConfig attachment", "frozen DeployOPChainInput.SuperchainConfig", v.expected.OptimismPortalProxy, continuationSuperchainConfigMethod, nil, v.dci.SuperchainConfig},
		{"AnchorStateRegistry SuperchainConfig attachment", "frozen DeployOPChainInput.SuperchainConfig", v.expected.AnchorStateRegistryProxy, continuationSuperchainConfigMethod, nil, v.dci.SuperchainConfig},
		{"L1CrossDomainMessenger SuperchainConfig attachment", "frozen DeployOPChainInput.SuperchainConfig", v.expected.L1CrossDomainMessengerProxy, continuationSuperchainConfigMethod, nil, v.dci.SuperchainConfig},
		{"L1ERC721Bridge SuperchainConfig attachment", "frozen DeployOPChainInput.SuperchainConfig", v.expected.L1Erc721BridgeProxy, continuationSuperchainConfigMethod, nil, v.dci.SuperchainConfig},
		{"L1StandardBridge SuperchainConfig attachment", "frozen DeployOPChainInput.SuperchainConfig", v.expected.L1StandardBridgeProxy, continuationSuperchainConfigMethod, nil, v.dci.SuperchainConfig},
		{"ETHLockbox SuperchainConfig attachment", "frozen DeployOPChainInput.SuperchainConfig", v.expected.EthLockboxProxy, continuationSuperchainConfigMethod, nil, v.dci.SuperchainConfig},
		{"DelayedWETH permissioned SuperchainConfig attachment", "frozen DeployOPChainInput.SuperchainConfig", v.expected.DelayedWethPermissionedGameProxy, continuationConfigMethod, nil, v.dci.SuperchainConfig},
		{"DelayedWETH permissionless SuperchainConfig attachment", "frozen DeployOPChainInput.SuperchainConfig", v.expected.DelayedWethPermissionlessGameProxy, continuationConfigMethod, nil, v.dci.SuperchainConfig},
	}
	verifyContinuationExpectations(v, attachments)

	guardian, err := readContinuation[common.Address](
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
	verifyContinuationExpectations(v, []continuationExpectation[common.Address]{
		{"SystemConfig guardian", "guardian read live from frozen DeployOPChainInput.SuperchainConfig", v.expected.SystemConfigProxy, continuationGuardianMethod, nil, guardian},
		{"OptimismPortal guardian", "guardian read live from frozen DeployOPChainInput.SuperchainConfig", v.expected.OptimismPortalProxy, continuationGuardianMethod, nil, guardian},
	})
}

func (v *continuationVerifier) verifyStandardValidator() {
	validator, err := readContinuation[common.Address](
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

type continuationExpectation[T comparable] struct {
	check    string
	source   string
	contract common.Address
	method   *w3.Func
	args     []any
	expected T
}

func verifyContinuationExpectations[T comparable](
	v *continuationVerifier,
	expectations []continuationExpectation[T],
) {
	for _, expectation := range expectations {
		if expectation.contract == (common.Address{}) {
			continue
		}
		observed, err := readContinuation[T](
			v.ctx,
			v.backend,
			expectation.contract,
			expectation.method,
			expectation.args...,
		)
		if err != nil {
			v.addReadError(expectation.check, expectation.source, expectation.expected, err)
		} else if observed != expectation.expected {
			v.addMismatch(expectation.check, expectation.source, expectation.expected, observed)
		}
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

	// Shared implementations are not deployment markers because they may already
	// contain code before this chain's contracts are deployed.
	deploymentMarker bool
}

func continuationContractAddresses(contracts addresses.OpChainContracts) []namedAddress {
	return []namedAddress{
		{"OpChainProxyAdminImpl", contracts.OpChainProxyAdminImpl, true},
		{"OptimismPortalProxy", contracts.OptimismPortalProxy, true},
		{"AddressManagerImpl", contracts.AddressManagerImpl, true},
		{"L1Erc721BridgeProxy", contracts.L1Erc721BridgeProxy, true},
		{"SystemConfigProxy", contracts.SystemConfigProxy, true},
		{"OptimismMintableErc20FactoryProxy", contracts.OptimismMintableErc20FactoryProxy, true},
		{"L1StandardBridgeProxy", contracts.L1StandardBridgeProxy, true},
		{"L1CrossDomainMessengerProxy", contracts.L1CrossDomainMessengerProxy, true},
		{"EthLockboxProxy", contracts.EthLockboxProxy, true},
		{"DisputeGameFactoryProxy", contracts.DisputeGameFactoryProxy, true},
		{"AnchorStateRegistryProxy", contracts.AnchorStateRegistryProxy, true},
		{"FaultDisputeGameImpl", contracts.FaultDisputeGameImpl, false},
		{"FaultDisputeGameCannonKonaImpl", contracts.FaultDisputeGameCannonKonaImpl, false},
		{"PermissionedDisputeGameImpl", contracts.PermissionedDisputeGameImpl, false},
		{"DelayedWethPermissionedGameProxy", contracts.DelayedWethPermissionedGameProxy, true},
		{"DelayedWethPermissionlessGameProxy", contracts.DelayedWethPermissionlessGameProxy, true},
		{"AltDAChallengeProxy", contracts.AltDAChallengeProxy, true},
		{"AltDAChallengeImpl", contracts.AltDAChallengeImpl, false},
		{"L2OutputOracleProxy", contracts.L2OutputOracleProxy, true},
	}
}

var (
	continuationRespectedGameTypeMethod   = w3.MustNewFunc("respectedGameType()", "uint32")
	continuationGameImplMethod            = w3.MustNewFunc("gameImpls(uint32)", "address")
	continuationGameArgsMethod            = w3.MustNewFunc("gameArgs(uint32)", "bytes")
	continuationImplementationsMethod     = w3.MustNewFunc("implementations()", `(address superchainConfigImpl,address l1ERC721BridgeImpl,address optimismPortalImpl,address ethLockboxImpl,address systemConfigImpl,address optimismMintableERC20FactoryImpl,address l1CrossDomainMessengerImpl,address l1StandardBridgeImpl,address disputeGameFactoryImpl,address anchorStateRegistryImpl,address delayedWETHImpl,address mipsImpl,address faultDisputeGameImpl,address permissionedDisputeGameImpl,address superFaultDisputeGameImpl,address superPermissionedDisputeGameImpl,address zkDisputeGameImpl,address storageSetterImpl,address sp1PlonkAdapterImpl)`)
	continuationOwnerMethod               = w3.MustNewFunc("owner()", "address")
	continuationAddressManagerMethod      = w3.MustNewFunc("addressManager()", "address")
	continuationProxyImplementationMethod = w3.MustNewFunc("getProxyImplementation(address)", "address")
	continuationBasefeeScalarMethod       = w3.MustNewFunc("basefeeScalar()", "uint32")
	continuationBlobBasefeeScalarMethod   = w3.MustNewFunc("blobbasefeeScalar()", "uint32")
	continuationScalarMethod              = w3.MustNewFunc("scalar()", "uint256")
	continuationBatcherHashMethod         = w3.MustNewFunc("batcherHash()", "bytes32")
	continuationUnsafeBlockSignerMethod   = w3.MustNewFunc("unsafeBlockSigner()", "address")
	continuationGasLimitMethod            = w3.MustNewFunc("gasLimit()", "uint64")
	continuationL2ChainIDMethod           = w3.MustNewFunc("l2ChainId()", "uint256")
	continuationOperatorFeeScalarMethod   = w3.MustNewFunc("operatorFeeScalar()", "uint32")
	continuationOperatorFeeConstantMethod = w3.MustNewFunc("operatorFeeConstant()", "uint64")
	continuationIsCustomGasTokenMethod    = w3.MustNewFunc("isCustomGasToken()", "bool")
	continuationSuperchainConfigMethod    = w3.MustNewFunc("superchainConfig()", "address")
	continuationL1XDMMethod               = w3.MustNewFunc("l1CrossDomainMessenger()", "address")
	continuationL1ERC721BridgeMethod      = w3.MustNewFunc("l1ERC721Bridge()", "address")
	continuationL1StandardBridgeMethod    = w3.MustNewFunc("l1StandardBridge()", "address")
	continuationOptimismPortalMethod      = w3.MustNewFunc("optimismPortal()", "address")
	continuationMintableFactoryMethod     = w3.MustNewFunc("optimismMintableERC20Factory()", "address")
	continuationAnchorRegistryMethod      = w3.MustNewFunc("anchorStateRegistry()", "address")
	continuationEthLockboxMethod          = w3.MustNewFunc("ethLockbox()", "address")
	continuationProxyAdminOwnerMethod     = w3.MustNewFunc("proxyAdminOwner()", "address")
	continuationPausedMethod              = w3.MustNewFunc("paused()", "bool")
	continuationConfigMethod              = w3.MustNewFunc("config()", "address")
	continuationGuardianMethod            = w3.MustNewFunc("guardian()", "address")
	continuationStandardValidatorMethod   = w3.MustNewFunc("opcmStandardValidator()", "address")
	continuationDevFeatureBitmapMethod    = w3.MustNewFunc("devFeatureBitmap()", "bytes32")
	continuationStartingAnchorRootMethod  = w3.MustNewFunc("getStartingAnchorRoot()", "(bytes32 root,uint256 l2SequenceNumber)")
)

func readContinuation[T any](
	ctx context.Context,
	backend opcm.CallContractBackend,
	contract common.Address,
	method *w3.Func,
	args ...any,
) (T, error) {
	var value T
	calldata, err := method.EncodeArgs(args...)
	if err != nil {
		return value, fmt.Errorf("failed to encode %s call: %w", method.Signature, err)
	}
	result, err := backend.CallContract(ctx, ethereum.CallMsg{To: &contract, Data: calldata}, nil)
	if err != nil {
		return value, fmt.Errorf("%s call to %s failed: %w", method.Signature, contract, err)
	}
	if err := method.DecodeReturns(result, &value); err != nil {
		return value, fmt.Errorf("failed to decode %s result from %s: %w", method.Signature, contract, err)
	}
	return value, nil
}

func readContinuationOPCMImplementations(
	ctx context.Context,
	backend opcm.CallContractBackend,
	opcmAddress common.Address,
) (continuationOPCMImplementations, error) {
	return readContinuation[continuationOPCMImplementations](
		ctx,
		backend,
		opcmAddress,
		continuationImplementationsMethod,
	)
}

// These lengths mirror LibGameArgs in packages/contracts-bedrock/src/dispute/lib/LibGameArgs.sol.
const (
	continuationPermissionedGameArgsLength      = 164 // LibGameArgs.PERMISSIONED_ARGS_LENGTH
	continuationSuperPermissionedGameArgsLength = 40  // LibGameArgs.SUPER_PERMISSIONED_ARGS_LENGTH
)

// These offsets mirror packages/contracts-bedrock/src/dispute/lib/LibGameArgs.sol.
func decodeContinuationGameArgs(
	gameArgs []byte,
	layout continuationGameArgsLayout,
) (continuationGameArgs, error) {
	if layout == continuationSuperPermissionedGameArgs {
		if len(gameArgs) != continuationSuperPermissionedGameArgsLength {
			return continuationGameArgs{}, fmt.Errorf(
				"configured super permissioned game arguments have length %d, expected %d",
				len(gameArgs),
				continuationSuperPermissionedGameArgsLength,
			)
		}
		return continuationGameArgs{
			anchorStateRegistry: common.BytesToAddress(gameArgs[:20]),
			proposer:            common.BytesToAddress(gameArgs[20:40]),
		}, nil
	}

	if layout != continuationPermissionedGameArgs {
		return continuationGameArgs{}, fmt.Errorf("unknown continuation game argument layout %d", layout)
	}
	if len(gameArgs) != continuationPermissionedGameArgsLength {
		return continuationGameArgs{}, fmt.Errorf(
			"configured game arguments have length %d, expected %d",
			len(gameArgs),
			continuationPermissionedGameArgsLength,
		)
	}

	decoded := continuationGameArgs{
		absolutePrestate:    common.BytesToHash(gameArgs[:32]),
		vm:                  common.BytesToAddress(gameArgs[32:52]),
		anchorStateRegistry: common.BytesToAddress(gameArgs[52:72]),
		delayedWETH:         common.BytesToAddress(gameArgs[72:92]),
		l2ChainID:           new(big.Int).SetBytes(gameArgs[92:124]),
	}
	decoded.proposer = common.BytesToAddress(gameArgs[124:144])
	decoded.challenger = common.BytesToAddress(gameArgs[144:164])
	return decoded, nil
}
