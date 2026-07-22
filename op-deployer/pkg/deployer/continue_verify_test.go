package deployer

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	opeth "github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
	"github.com/stretchr/testify/require"
)

type continuationVerificationBackend struct {
	responses       map[string][]byte
	code            map[common.Address][]byte
	calls           []ethereum.CallMsg
	validator       common.Address
	validatorResult string
}

func newContinuationVerificationBackend() *continuationVerificationBackend {
	return &continuationVerificationBackend{
		responses: make(map[string][]byte),
		code:      make(map[common.Address][]byte),
	}
}

func (b *continuationVerificationBackend) CallContract(
	_ context.Context,
	call ethereum.CallMsg,
	_ *big.Int,
) ([]byte, error) {
	b.calls = append(b.calls, call)
	if call.To == nil {
		return nil, fmt.Errorf("missing call recipient")
	}
	if *call.To == b.validator {
		outputs := abi.Arguments{{Type: opcm.MustType("string")}}
		return outputs.Pack(b.validatorResult)
	}
	result, ok := b.responses[continuationCallKey(*call.To, call.Data)]
	if !ok {
		return nil, fmt.Errorf("unexpected call to %s with calldata %s", *call.To, hex.EncodeToString(call.Data))
	}
	return bytes.Clone(result), nil
}

func (b *continuationVerificationBackend) CodeAt(
	_ context.Context,
	contract common.Address,
	_ *big.Int,
) ([]byte, error) {
	return bytes.Clone(b.code[contract]), nil
}

func (b *continuationVerificationBackend) set(
	t *testing.T,
	contract common.Address,
	method *w3.Func,
	args []any,
	values ...any,
) {
	t.Helper()
	calldata, err := method.EncodeArgs(args...)
	require.NoError(t, err)
	output, err := method.Returns.Pack(values...)
	require.NoError(t, err)
	b.responses[continuationCallKey(contract, calldata)] = output
}

func (b *continuationVerificationBackend) callsTo(contract common.Address) int {
	count := 0
	for _, call := range b.calls {
		if call.To != nil && *call.To == contract {
			count++
		}
	}
	return count
}

func continuationCallKey(contract common.Address, calldata []byte) string {
	return contract.Hex() + ":" + hex.EncodeToString(calldata)
}

type continuationVerificationFixture struct {
	backend   *continuationVerificationBackend
	observed  addresses.OpChainContracts
	expected  *state.ChainState
	dci       opcm.DeployOPChainInput
	guardian  common.Address
	vm        common.Address
	impls     continuationOPCMImplementations
	superRoot bool
}

func newContinuationVerificationFixture(
	t *testing.T,
	gameType embedded.GameType,
) *continuationVerificationFixture {
	superRoot := gameType == embedded.GameTypeSuperCannonKona
	return newContinuationVerificationFixtureWithMode(t, gameType, superRoot)
}

func newContinuationVerificationFixtureWithMode(
	t *testing.T,
	gameType embedded.GameType,
	superRoot bool,
) *continuationVerificationFixture {
	t.Helper()
	contracts := continuationVerificationAddresses(gameType)
	selectedPrestate := common.HexToHash("0x1234")
	startingAnchor := &state.StartingAnchorProposal{
		Root:             common.HexToHash("0x5678"),
		L2SequenceNumber: 7,
	}
	expected := &state.ChainState{
		ID:                 common.HexToHash("0x44"),
		OpChainContracts:   contracts,
		Prestate:           selectedPrestate,
		StartingAnchorRoot: startingAnchor,
	}
	dci := opcm.DeployOPChainInput{
		OpChainProxyAdminOwner:       common.Address{0xa1},
		SystemConfigOwner:            common.Address{0xa2},
		Batcher:                      common.Address{0xa3},
		UnsafeBlockSigner:            common.Address{0xa4},
		Proposer:                     common.Address{0xa5},
		Challenger:                   common.Address{0xa6},
		BasefeeScalar:                1_368,
		BlobBaseFeeScalar:            810_949,
		L2ChainId:                    big.NewInt(901),
		Opcm:                         common.Address{0xb1},
		SaltMixer:                    "",
		GasLimit:                     30_000_000,
		DisputeGameType:              uint32(gameType),
		DisputeAbsolutePrestate:      selectedPrestate,
		CannonAbsolutePrestate:       common.Hash{},
		DisputeMaxGameDepth:          nil,
		DisputeSplitDepth:            nil,
		DisputeClockExtension:        0,
		DisputeMaxClockDuration:      0,
		AllowCustomDisputeParameters: false,
		OperatorFeeScalar:            17,
		OperatorFeeConstant:          23,
		UseCustomGasToken:            true,
		StartingAnchorRoot: opcm.Proposal{
			Root:             startingAnchor.Root,
			L2SequenceNumber: big.NewInt(7),
		},
		SuperchainConfig: common.Address{0xb2},
	}
	if gameType == embedded.GameTypePermissionedCannon {
		expected.Prestate = common.Hash{}
		expected.StartingAnchorRoot = nil
		dci.DisputeAbsolutePrestate = common.HexToHash("0x4321")
		dci.StartingAnchorRoot = opcm.Proposal{
			Root:             opcm.DefaultStartingAnchorRoot.Root,
			L2SequenceNumber: new(big.Int),
		}
	}
	if gameType == embedded.GameTypeCannonKona {
		dci.CannonAbsolutePrestate = opcm.PermissionedCannonFallbackPrestatePlaceholder
	}

	fixture := &continuationVerificationFixture{
		backend:   newContinuationVerificationBackend(),
		observed:  contracts,
		expected:  expected,
		dci:       dci,
		guardian:  common.Address{0xb3},
		vm:        common.Address{0xc1},
		superRoot: superRoot,
	}
	fixture.impls = continuationOPCMImplementations{
		L1ERC721BridgeImpl:               common.Address{0xc2},
		OptimismPortalImpl:               common.Address{0xc3},
		ETHLockboxImpl:                   common.Address{0xc4},
		SystemConfigImpl:                 common.Address{0xc5},
		OptimismMintableERC20FactoryImpl: common.Address{0xc6},
		L1CrossDomainMessengerImpl:       common.Address{0xc7},
		L1StandardBridgeImpl:             common.Address{0xc8},
		DisputeGameFactoryImpl:           common.Address{0xc9},
		AnchorStateRegistryImpl:          common.Address{0xca},
		DelayedWETHImpl:                  common.Address{0xcb},
		MipsImpl:                         fixture.vm,
	}
	fixture.seed(t, gameType)
	return fixture
}

func continuationVerificationAddresses(gameType embedded.GameType) addresses.OpChainContracts {
	contracts := addresses.OpChainContracts{
		OpChainCoreContracts: addresses.OpChainCoreContracts{
			OpChainProxyAdminImpl:             common.Address{0x11},
			OptimismPortalProxy:               common.Address{0x12},
			AddressManagerImpl:                common.Address{0x13},
			L1Erc721BridgeProxy:               common.Address{0x14},
			SystemConfigProxy:                 common.Address{0x15},
			OptimismMintableErc20FactoryProxy: common.Address{0x16},
			L1StandardBridgeProxy:             common.Address{0x17},
			L1CrossDomainMessengerProxy:       common.Address{0x18},
			EthLockboxProxy:                   common.Address{0x19},
		},
		OpChainFaultProofsContracts: addresses.OpChainFaultProofsContracts{
			DisputeGameFactoryProxy:            common.Address{0x21},
			AnchorStateRegistryProxy:           common.Address{0x22},
			PermissionedDisputeGameImpl:        common.Address{0x24},
			DelayedWethPermissionedGameProxy:   common.Address{0x25},
			DelayedWethPermissionlessGameProxy: common.Address{0x25},
		},
	}
	if gameType != embedded.GameTypePermissionedCannon {
		contracts.FaultDisputeGameImpl = common.Address{0x23}
	}
	return contracts
}

func (f *continuationVerificationFixture) seed(t *testing.T, gameType embedded.GameType) {
	t.Helper()
	for _, contract := range continuationContractAddresses(f.expected.OpChainContracts) {
		if contract.address != (common.Address{}) {
			f.backend.code[contract.address] = []byte{0x60}
		}
	}

	f.backend.set(
		t,
		f.dci.Opcm,
		continuationDevFeatureBitmapMethod,
		nil,
		f.devFeatureBitmap(),
	)
	f.backend.set(
		t,
		f.dci.Opcm,
		continuationImplementationsMethod,
		nil,
		f.impls,
	)

	respectedGameType := gameType
	if gameType == embedded.GameTypePermissionedCannon && f.superRoot {
		respectedGameType = embedded.GameTypeSuperPermissioned
	}
	f.backend.set(
		t,
		f.expected.OptimismPortalProxy,
		continuationRespectedGameTypeMethod,
		nil,
		uint32(respectedGameType),
	)

	switch gameType {
	case embedded.GameTypePermissionedCannon:
		if f.superRoot {
			f.seedGameImplementation(
				t,
				uint32(embedded.GameTypeSuperPermissioned),
				f.expected.PermissionedDisputeGameImpl,
				superPermissionedContinuationGameArgs(
					f.expected.AnchorStateRegistryProxy,
					f.dci.Proposer,
				),
			)
			break
		}
		f.seedGameImplementation(
			t,
			uint32(gameType),
			f.expected.PermissionedDisputeGameImpl,
			permissionedContinuationGameArgs(
				f.dci.DisputeAbsolutePrestate,
				f.vm,
				f.expected.AnchorStateRegistryProxy,
				f.expected.DelayedWethPermissionedGameProxy,
				f.dci.L2ChainId,
				f.dci.Proposer,
				f.dci.Challenger,
			),
		)
	case embedded.GameTypeCannonKona:
		f.seedGameImplementation(
			t,
			uint32(gameType),
			f.expected.FaultDisputeGameImpl,
			permissionlessContinuationGameArgs(
				f.expected.Prestate,
				f.vm,
				f.expected.AnchorStateRegistryProxy,
				f.expected.DelayedWethPermissionlessGameProxy,
				f.dci.L2ChainId,
			),
		)
		f.seedGameImplementation(
			t,
			uint32(embedded.GameTypePermissionedCannon),
			f.expected.PermissionedDisputeGameImpl,
			permissionedContinuationGameArgs(
				opcm.PermissionedCannonFallbackPrestatePlaceholder,
				f.vm,
				f.expected.AnchorStateRegistryProxy,
				f.expected.DelayedWethPermissionedGameProxy,
				f.dci.L2ChainId,
				f.dci.Proposer,
				f.dci.Challenger,
			),
		)
	case embedded.GameTypeSuperCannonKona:
		f.seedGameImplementation(
			t,
			uint32(gameType),
			f.expected.FaultDisputeGameImpl,
			permissionlessContinuationGameArgs(
				f.expected.Prestate,
				f.vm,
				f.expected.AnchorStateRegistryProxy,
				f.expected.DelayedWethPermissionlessGameProxy,
				new(big.Int),
			),
		)
		f.seedGameImplementation(
			t,
			uint32(embedded.GameTypeSuperPermissioned),
			f.expected.PermissionedDisputeGameImpl,
			superPermissionedContinuationGameArgs(f.expected.AnchorStateRegistryProxy, f.dci.Proposer),
		)
	default:
		require.FailNow(t, "unsupported game type", gameType)
	}

	f.backend.set(t, f.expected.SystemConfigProxy, continuationOwnerMethod, nil, f.dci.SystemConfigOwner)
	f.backend.set(t, f.expected.SystemConfigProxy, continuationBasefeeScalarMethod, nil, f.dci.BasefeeScalar)
	f.backend.set(t, f.expected.SystemConfigProxy, continuationBlobBasefeeScalarMethod, nil, f.dci.BlobBaseFeeScalar)
	f.backend.set(t, f.expected.SystemConfigProxy, continuationOperatorFeeScalarMethod, nil, f.dci.OperatorFeeScalar)
	f.backend.set(t, f.expected.SystemConfigProxy, continuationOperatorFeeConstantMethod, nil, f.dci.OperatorFeeConstant)
	f.backend.set(t, f.expected.SystemConfigProxy, continuationIsCustomGasTokenMethod, nil, f.dci.UseCustomGasToken)
	f.backend.set(
		t,
		f.expected.SystemConfigProxy,
		continuationBatcherHashMethod,
		nil,
		common.BytesToHash(f.dci.Batcher.Bytes()),
	)
	f.backend.set(
		t,
		f.expected.SystemConfigProxy,
		continuationUnsafeBlockSignerMethod,
		nil,
		f.dci.UnsafeBlockSigner,
	)
	f.backend.set(t, f.expected.SystemConfigProxy, continuationGasLimitMethod, nil, f.dci.GasLimit)
	f.backend.set(t, f.expected.SystemConfigProxy, continuationL2ChainIDMethod, nil, f.dci.L2ChainId)
	f.backend.set(
		t,
		f.expected.OpChainProxyAdminImpl,
		continuationOwnerMethod,
		nil,
		f.dci.OpChainProxyAdminOwner,
	)
	verifier := &continuationVerifier{expected: f.expected, dci: f.dci}
	for _, expectation := range verifier.persistentAddressExpectations(&f.impls) {
		f.backend.set(t, expectation.contract, expectation.method, expectation.args, expectation.expected)
	}
	scalar := opeth.EncodeScalar(opeth.EcotoneScalars{
		BlobBaseFeeScalar: f.dci.BlobBaseFeeScalar,
		BaseFeeScalar:     f.dci.BasefeeScalar,
	})
	f.backend.set(t, f.expected.SystemConfigProxy, continuationScalarMethod, nil, new(big.Int).SetBytes(scalar[:]))
	f.backend.set(t, f.expected.SystemConfigProxy, continuationPausedMethod, nil, false)
	f.backend.set(t, f.expected.OptimismPortalProxy, continuationPausedMethod, nil, false)
	f.backend.set(t, f.expected.OptimismPortalProxy, continuationEthLockboxMethod, nil, f.expected.EthLockboxProxy)

	attachments := []struct {
		address common.Address
		method  *w3.Func
	}{
		{f.expected.SystemConfigProxy, continuationSuperchainConfigMethod},
		{f.expected.OptimismPortalProxy, continuationSuperchainConfigMethod},
		{f.expected.AnchorStateRegistryProxy, continuationSuperchainConfigMethod},
		{f.expected.L1CrossDomainMessengerProxy, continuationSuperchainConfigMethod},
		{f.expected.L1Erc721BridgeProxy, continuationSuperchainConfigMethod},
		{f.expected.L1StandardBridgeProxy, continuationSuperchainConfigMethod},
		{f.expected.EthLockboxProxy, continuationSuperchainConfigMethod},
		{f.expected.DelayedWethPermissionedGameProxy, continuationConfigMethod},
		{f.expected.DelayedWethPermissionlessGameProxy, continuationConfigMethod},
	}
	for _, attachment := range attachments {
		f.backend.set(t, attachment.address, attachment.method, nil, f.dci.SuperchainConfig)
	}
	f.backend.set(t, f.dci.SuperchainConfig, continuationGuardianMethod, nil, f.guardian)
	f.backend.set(t, f.expected.SystemConfigProxy, continuationGuardianMethod, nil, f.guardian)
	f.backend.set(t, f.expected.OptimismPortalProxy, continuationGuardianMethod, nil, f.guardian)

	if gameType != embedded.GameTypePermissionedCannon {
		f.backend.validator = common.Address{0xb4}
		f.backend.validatorResult = "OVERRIDES-L1PAOMULTISIG,OVERRIDES-CHALLENGER"
		f.backend.set(
			t,
			f.dci.Opcm,
			continuationStandardValidatorMethod,
			nil,
			f.backend.validator,
		)
	}
}

func (f *continuationVerificationFixture) devFeatureBitmap() common.Hash {
	if f.superRoot {
		return devfeatures.SuperRootGamesMigrationFlag
	}
	return common.Hash{}
}

func (f *continuationVerificationFixture) seedGameImplementation(
	t *testing.T,
	gameType uint32,
	implementation common.Address,
	gameArgs []byte,
) {
	t.Helper()
	f.backend.set(
		t,
		f.expected.DisputeGameFactoryProxy,
		continuationGameImplMethod,
		[]any{gameType},
		implementation,
	)
	f.backend.set(
		t,
		f.expected.DisputeGameFactoryProxy,
		continuationGameArgsMethod,
		[]any{gameType},
		gameArgs,
	)
}

func permissionlessContinuationGameArgs(
	prestate common.Hash,
	vm common.Address,
	anchorStateRegistry common.Address,
	delayedWETH common.Address,
	l2ChainID *big.Int,
) []byte {
	args := append([]byte{}, prestate.Bytes()...)
	args = append(args, vm.Bytes()...)
	args = append(args, anchorStateRegistry.Bytes()...)
	args = append(args, delayedWETH.Bytes()...)
	return append(args, common.LeftPadBytes(l2ChainID.Bytes(), common.HashLength)...)
}

func permissionedContinuationGameArgs(
	prestate common.Hash,
	vm common.Address,
	anchorStateRegistry common.Address,
	delayedWETH common.Address,
	l2ChainID *big.Int,
	proposer common.Address,
	challenger common.Address,
) []byte {
	args := permissionlessContinuationGameArgs(prestate, vm, anchorStateRegistry, delayedWETH, l2ChainID)
	args = append(args, proposer.Bytes()...)
	return append(args, challenger.Bytes()...)
}

func superPermissionedContinuationGameArgs(anchorStateRegistry common.Address, proposer common.Address) []byte {
	args := append([]byte{}, anchorStateRegistry.Bytes()...)
	return append(args, proposer.Bytes()...)
}

func (f *continuationVerificationFixture) permissionedArgs(
	prestate common.Hash,
	proposer common.Address,
	challenger common.Address,
) []byte {
	return permissionedContinuationGameArgs(
		prestate,
		f.vm,
		f.expected.AnchorStateRegistryProxy,
		f.expected.DelayedWethPermissionedGameProxy,
		f.dci.L2ChainId,
		proposer,
		challenger,
	)
}

func (f *continuationVerificationFixture) setGameArgs(
	t *testing.T,
	gameType embedded.GameType,
	gameArgs []byte,
) {
	t.Helper()
	f.backend.set(
		t,
		f.expected.DisputeGameFactoryProxy,
		continuationGameArgsMethod,
		[]any{uint32(gameType)},
		gameArgs,
	)
}

func (f *continuationVerificationFixture) verify(t *testing.T) error {
	t.Helper()
	return verifyContinuationDeployment(
		t.Context(),
		f.backend,
		f.observed,
		f.expected,
		f.dci,
	)
}

func TestVerifyContinuationDeployment(t *testing.T) {
	t.Run("CANNON_KONA", func(t *testing.T) {
		fixture := newContinuationVerificationFixture(t, embedded.GameTypeCannonKona)
		require.NoError(t, fixture.verify(t))
		require.Equal(t, 1, fixture.backend.callsTo(fixture.backend.validator))
	})

	t.Run("permissioned only skips StandardValidator", func(t *testing.T) {
		fixture := newContinuationVerificationFixture(t, embedded.GameTypePermissionedCannon)
		require.NoError(t, fixture.verify(t))
		require.Equal(t, 2, fixture.backend.callsTo(fixture.dci.Opcm))
		require.Zero(t, fixture.backend.callsTo(fixture.backend.validator))
	})

	t.Run("permissioned selector uses SUPER_PERMISSIONED with a super-root OPCM", func(t *testing.T) {
		fixture := newContinuationVerificationFixtureWithMode(t, embedded.GameTypePermissionedCannon, true)
		require.NoError(t, fixture.verify(t))
		require.Equal(t, 2, fixture.backend.callsTo(fixture.dci.Opcm))
		require.Zero(t, fixture.backend.callsTo(fixture.backend.validator))
	})

	t.Run("SUPER_CANNON_KONA has a no-prestate fallback", func(t *testing.T) {
		fixture := newContinuationVerificationFixture(t, embedded.GameTypeSuperCannonKona)
		require.NoError(t, fixture.verify(t))
		require.Zero(t, fixture.dci.CannonAbsolutePrestate)
	})
}

func TestVerifyContinuationDeploymentRejectsOPCMGameModeMismatch(t *testing.T) {
	t.Run("CANNON_KONA with super-root OPCM", func(t *testing.T) {
		fixture := newContinuationVerificationFixture(t, embedded.GameTypeCannonKona)
		fixture.backend.set(
			t,
			fixture.dci.Opcm,
			continuationDevFeatureBitmapMethod,
			nil,
			devfeatures.SuperRootGamesMigrationFlag,
		)
		require.ErrorContains(t, fixture.verify(t), "requires an OPCM without SUPER_ROOT_GAMES_MIGRATION")
	})

	t.Run("SUPER_CANNON_KONA without super-root OPCM", func(t *testing.T) {
		fixture := newContinuationVerificationFixture(t, embedded.GameTypeSuperCannonKona)
		fixture.backend.set(t, fixture.dci.Opcm, continuationDevFeatureBitmapMethod, nil, common.Hash{})
		require.ErrorContains(t, fixture.verify(t), "requires an OPCM with SUPER_ROOT_GAMES_MIGRATION")
	})
}

func TestVerifyContinuationDeploymentSuperPermissionedArguments(t *testing.T) {
	tests := []struct {
		name    string
		wantErr string
		args    func(*continuationVerificationFixture) []byte
	}{
		{
			name:    "AnchorStateRegistry",
			wantErr: "selected game AnchorStateRegistry",
			args: func(f *continuationVerificationFixture) []byte {
				return superPermissionedContinuationGameArgs(common.Address{0xff}, f.dci.Proposer)
			},
		},
		{
			name:    "proposer",
			wantErr: "selected game proposer",
			args: func(f *continuationVerificationFixture) []byte {
				return superPermissionedContinuationGameArgs(f.expected.AnchorStateRegistryProxy, common.Address{0xff})
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newContinuationVerificationFixtureWithMode(t, embedded.GameTypePermissionedCannon, true)
			fixture.setGameArgs(t, embedded.GameTypeSuperPermissioned, test.args(fixture))
			require.ErrorContains(t, fixture.verify(t), test.wantErr)
		})
	}
}

func TestVerifyContinuationDeploymentChecksPermissionedSystemConfigExactly(t *testing.T) {
	fixture := newContinuationVerificationFixture(t, embedded.GameTypePermissionedCannon)
	fixture.backend.set(
		t,
		fixture.expected.SystemConfigProxy,
		continuationIsCustomGasTokenMethod,
		nil,
		!fixture.dci.UseCustomGasToken,
	)
	err := fixture.verify(t)
	require.ErrorContains(t, err, "SystemConfig custom gas token mode")
	require.Zero(t, fixture.backend.callsTo(fixture.backend.validator))
}

func TestVerifyContinuationDeploymentPermissionedAddressParity(t *testing.T) {
	template := newContinuationVerificationFixture(t, embedded.GameTypePermissionedCannon)
	verifier := &continuationVerifier{expected: template.expected, dci: template.dci}
	for _, expectation := range verifier.persistentAddressExpectations(&template.impls) {
		t.Run(expectation.check, func(t *testing.T) {
			fixture := newContinuationVerificationFixture(t, embedded.GameTypePermissionedCannon)
			fixture.backend.set(t, expectation.contract, expectation.method, expectation.args, common.Address{0xff})
			require.ErrorContains(t, fixture.verify(t), expectation.check)
		})
	}
}

func TestDecodeContinuationGameArgsRejectsInvalidLengths(t *testing.T) {
	tests := []struct {
		layout continuationGameArgsLayout
		length int
	}{
		{continuationPermissionedGameArgs, 163},
		{continuationPermissionedGameArgs, 165},
		{continuationSuperPermissionedGameArgs, 39},
		{continuationSuperPermissionedGameArgs, 41},
	}
	for _, test := range tests {
		_, err := decodeContinuationGameArgs(make([]byte, test.length), test.layout)
		require.Error(t, err)
	}
}

func TestVerifyContinuationDeploymentFailures(t *testing.T) {
	tests := []struct {
		name         string
		wantErr      string
		permissioned bool
		mutate       func(*testing.T, *continuationVerificationFixture)
	}{
		{
			name:    "address set",
			wantErr: "simulated contract addresses differ",
			mutate: func(_ *testing.T, f *continuationVerificationFixture) {
				f.observed.SystemConfigProxy = common.Address{0xff}
			},
		},
		{
			name:    "missing code",
			wantErr: "AddressManagerImpl code",
			mutate: func(_ *testing.T, f *continuationVerificationFixture) {
				delete(f.backend.code, f.expected.AddressManagerImpl)
			},
		},
		{
			name:    "respected game type",
			wantErr: "respected game type",
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				f.backend.set(
					t,
					f.expected.OptimismPortalProxy,
					continuationRespectedGameTypeMethod,
					nil,
					uint32(embedded.GameTypePermissionedCannon),
				)
			},
		},
		{
			name:    "initial game implementation",
			wantErr: "initial game implementation",
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				f.backend.set(
					t,
					f.expected.DisputeGameFactoryProxy,
					continuationGameImplMethod,
					[]any{uint32(embedded.GameTypeCannonKona)},
					common.Address{0xff},
				)
			},
		},
		{
			name:    "fallback game implementation",
			wantErr: "fallback game implementation",
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				f.backend.set(
					t,
					f.expected.DisputeGameFactoryProxy,
					continuationGameImplMethod,
					[]any{uint32(embedded.GameTypePermissionedCannon)},
					common.Address{0xff},
				)
			},
		},
		{
			name:         "selected prestate",
			wantErr:      "selected game prestate",
			permissioned: true,
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				f.setGameArgs(t, embedded.GameTypePermissionedCannon, f.permissionedArgs(common.Hash{0xff}, f.dci.Proposer, f.dci.Challenger))
			},
		},
		{
			name:         "selected VM",
			wantErr:      "selected game VM",
			permissioned: true,
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				args := f.permissionedArgs(f.dci.DisputeAbsolutePrestate, f.dci.Proposer, f.dci.Challenger)
				copy(args[32:52], common.Address{0xff}.Bytes())
				f.setGameArgs(t, embedded.GameTypePermissionedCannon, args)
			},
		},
		{
			name:         "selected AnchorStateRegistry",
			wantErr:      "selected game AnchorStateRegistry",
			permissioned: true,
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				args := f.permissionedArgs(f.dci.DisputeAbsolutePrestate, f.dci.Proposer, f.dci.Challenger)
				copy(args[52:72], common.Address{0xff}.Bytes())
				f.setGameArgs(t, embedded.GameTypePermissionedCannon, args)
			},
		},
		{
			name:         "selected DelayedWETH",
			wantErr:      "selected game DelayedWETH",
			permissioned: true,
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				args := f.permissionedArgs(f.dci.DisputeAbsolutePrestate, f.dci.Proposer, f.dci.Challenger)
				copy(args[72:92], common.Address{0xff}.Bytes())
				f.setGameArgs(t, embedded.GameTypePermissionedCannon, args)
			},
		},
		{
			name:         "selected L2 chain ID",
			wantErr:      "selected game L2 chain ID",
			permissioned: true,
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				args := f.permissionedArgs(f.dci.DisputeAbsolutePrestate, f.dci.Proposer, f.dci.Challenger)
				copy(args[92:124], common.LeftPadBytes(big.NewInt(902).Bytes(), common.HashLength))
				f.setGameArgs(t, embedded.GameTypePermissionedCannon, args)
			},
		},
		{
			name:    "SystemConfig owner",
			wantErr: "SystemConfig owner",
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				f.backend.set(t, f.expected.SystemConfigProxy, continuationOwnerMethod, nil, common.Address{0xff})
			},
		},
		{
			name:    "SystemConfig base fee scalar",
			wantErr: "SystemConfig base fee scalar",
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				f.backend.set(t, f.expected.SystemConfigProxy, continuationBasefeeScalarMethod, nil, f.dci.BasefeeScalar+1)
			},
		},
		{
			name:    "SystemConfig blob base fee scalar",
			wantErr: "SystemConfig blob base fee scalar",
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				f.backend.set(t, f.expected.SystemConfigProxy, continuationBlobBasefeeScalarMethod, nil, f.dci.BlobBaseFeeScalar+1)
			},
		},
		{
			name:    "SystemConfig operator fee scalar",
			wantErr: "SystemConfig operator fee scalar",
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				f.backend.set(t, f.expected.SystemConfigProxy, continuationOperatorFeeScalarMethod, nil, f.dci.OperatorFeeScalar+1)
			},
		},
		{
			name:    "SystemConfig operator fee constant",
			wantErr: "SystemConfig operator fee constant",
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				f.backend.set(t, f.expected.SystemConfigProxy, continuationOperatorFeeConstantMethod, nil, f.dci.OperatorFeeConstant+1)
			},
		},
		{
			name:    "SystemConfig custom gas token mode",
			wantErr: "SystemConfig custom gas token mode",
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				f.backend.set(t, f.expected.SystemConfigProxy, continuationIsCustomGasTokenMethod, nil, !f.dci.UseCustomGasToken)
			},
		},
		{
			name:    "SystemConfig batcher",
			wantErr: "SystemConfig batcher",
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				f.backend.set(t, f.expected.SystemConfigProxy, continuationBatcherHashMethod, nil, common.Hash{0xff})
			},
		},
		{
			name:    "SystemConfig unsafe signer",
			wantErr: "SystemConfig unsafe block signer",
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				f.backend.set(
					t,
					f.expected.SystemConfigProxy,
					continuationUnsafeBlockSignerMethod,
					nil,
					common.Address{0xff},
				)
			},
		},
		{
			name:    "SystemConfig gas limit",
			wantErr: "SystemConfig gas limit",
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				f.backend.set(t, f.expected.SystemConfigProxy, continuationGasLimitMethod, nil, f.dci.GasLimit+1)
			},
		},
		{
			name:    "SystemConfig L2 chain ID",
			wantErr: "SystemConfig L2 chain ID",
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				f.backend.set(t, f.expected.SystemConfigProxy, continuationL2ChainIDMethod, nil, big.NewInt(902))
			},
		},
		{
			name:         "SystemConfig combined scalar",
			wantErr:      "CHECK-SCFG-70 SystemConfig scalar",
			permissioned: true,
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				scalar := opeth.EncodeScalar(opeth.EcotoneScalars{
					BlobBaseFeeScalar: f.dci.BlobBaseFeeScalar,
					BaseFeeScalar:     f.dci.BasefeeScalar,
				})
				observed := new(big.Int).SetBytes(scalar[:])
				f.backend.set(t, f.expected.SystemConfigProxy, continuationScalarMethod, nil, observed.Add(observed, big.NewInt(1)))
			},
		},
		{
			name:         "OptimismPortal paused state",
			wantErr:      "CHECK-OP2-60 OptimismPortal paused state",
			permissioned: true,
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				f.backend.set(t, f.expected.OptimismPortalProxy, continuationPausedMethod, nil, true)
			},
		},
		{
			name:         "OptimismPortal ETHLockbox",
			wantErr:      "CHECK-OP2-80 OptimismPortal ETHLockbox",
			permissioned: true,
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				f.backend.set(t, f.expected.OptimismPortalProxy, continuationEthLockboxMethod, nil, common.Address{0xff})
			},
		},
		{
			name:         "permissioned proposer",
			wantErr:      "selected game proposer",
			permissioned: true,
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				f.setGameArgs(t, embedded.GameTypePermissionedCannon, f.permissionedArgs(f.dci.DisputeAbsolutePrestate, common.Address{0xff}, f.dci.Challenger))
			},
		},
		{
			name:         "permissioned challenger",
			wantErr:      "selected game challenger",
			permissioned: true,
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				f.setGameArgs(t, embedded.GameTypePermissionedCannon, f.permissionedArgs(f.dci.DisputeAbsolutePrestate, f.dci.Proposer, common.Address{0xff}))
			},
		},
		{
			name:         "ProxyAdmin owner",
			wantErr:      "OpChain ProxyAdmin owner",
			permissioned: true,
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				f.backend.set(t, f.expected.OpChainProxyAdminImpl, continuationOwnerMethod, nil, common.Address{0xff})
			},
		},
		{
			name:    "SuperchainConfig attachment",
			wantErr: "L1StandardBridge SuperchainConfig attachment",
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				f.backend.set(
					t,
					f.expected.L1StandardBridgeProxy,
					continuationSuperchainConfigMethod,
					nil,
					common.Address{0xff},
				)
			},
		},
		{
			name:    "guardian consistency",
			wantErr: "SystemConfig guardian",
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				f.backend.set(t, f.expected.SystemConfigProxy, continuationGuardianMethod, nil, common.Address{0xff})
			},
		},
		{
			name:    "StandardValidator non-empty result",
			wantErr: "StandardValidator result",
			mutate: func(_ *testing.T, f *continuationVerificationFixture) {
				f.backend.validatorResult = "TEST-FAIL"
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gameType := embedded.GameTypeCannonKona
			if test.permissioned {
				gameType = embedded.GameTypePermissionedCannon
			}
			fixture := newContinuationVerificationFixture(t, gameType)
			test.mutate(t, fixture)
			require.ErrorContains(t, fixture.verify(t), test.wantErr)
		})
	}
}
