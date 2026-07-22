package deployer

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
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
	method abi.Method,
	args []any,
	values ...any,
) {
	t.Helper()
	input, err := method.Inputs.Pack(args...)
	require.NoError(t, err)
	output, err := method.Outputs.Pack(values...)
	require.NoError(t, err)
	calldata := append(bytes.Clone(method.ID), input...)
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
	backend  *continuationVerificationBackend
	observed addresses.OpChainContracts
	expected *state.ChainState
	dci      opcm.DeployOPChainInput
	guardian common.Address
}

func newContinuationVerificationFixture(
	t *testing.T,
	gameType embedded.GameType,
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
		InitialGameType:    ptr.New(uint32(gameType)),
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
		BasefeeScalar:                0,
		BlobBaseFeeScalar:            0,
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
		OperatorFeeScalar:            0,
		OperatorFeeConstant:          0,
		UseCustomGasToken:            false,
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
		backend:  newContinuationVerificationBackend(),
		observed: contracts,
		expected: expected,
		dci:      dci,
		guardian: common.Address{0xb3},
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

	anchorRoot := f.dci.StartingAnchorRoot.Root
	anchorSequence := f.dci.StartingAnchorRoot.L2SequenceNumber
	f.backend.set(
		t,
		f.expected.AnchorStateRegistryProxy,
		continuationStartingAnchorMethod,
		nil,
		anchorRoot,
		anchorSequence,
	)
	f.backend.set(
		t,
		f.expected.OptimismPortalProxy,
		continuationRespectedGameTypeMethod,
		nil,
		uint32(gameType),
	)

	switch gameType {
	case embedded.GameTypePermissionedCannon:
		f.seedGameImplementation(
			t,
			uint32(gameType),
			f.expected.PermissionedDisputeGameImpl,
			permissionedContinuationGameArgs(
				f.dci.DisputeAbsolutePrestate,
				f.dci.Proposer,
				f.dci.Challenger,
			),
		)
	case embedded.GameTypeCannonKona:
		f.seedGameImplementation(
			t,
			uint32(gameType),
			f.expected.FaultDisputeGameImpl,
			permissionlessContinuationGameArgs(f.expected.Prestate),
		)
		f.seedGameImplementation(
			t,
			uint32(embedded.GameTypePermissionedCannon),
			f.expected.PermissionedDisputeGameImpl,
			permissionedContinuationGameArgs(
				opcm.PermissionedCannonFallbackPrestatePlaceholder,
				f.dci.Proposer,
				f.dci.Challenger,
			),
		)
	case embedded.GameTypeSuperCannonKona:
		f.seedGameImplementation(
			t,
			uint32(gameType),
			f.expected.FaultDisputeGameImpl,
			permissionlessContinuationGameArgs(f.expected.Prestate),
		)
		f.seedGameImplementation(
			t,
			uint32(embedded.GameTypeSuperPermissioned),
			f.expected.PermissionedDisputeGameImpl,
			superPermissionedContinuationGameArgs(f.dci.Proposer),
		)
	default:
		require.FailNow(t, "unsupported game type", gameType)
	}

	f.backend.set(t, f.expected.SystemConfigProxy, continuationOwnerMethod, nil, f.dci.SystemConfigOwner)
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

	attachments := []struct {
		address common.Address
		method  abi.Method
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
		f.backend.set(
			t,
			f.dci.Opcm,
			continuationStandardValidatorMethod,
			nil,
			f.backend.validator,
		)
	}
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

func permissionlessContinuationGameArgs(prestate common.Hash) []byte {
	args := append([]byte{}, prestate.Bytes()...)
	args = append(args, common.Address{0xc1}.Bytes()...)
	args = append(args, common.Address{0xc2}.Bytes()...)
	args = append(args, common.Address{0xc3}.Bytes()...)
	return append(args, common.LeftPadBytes(big.NewInt(901).Bytes(), common.HashLength)...)
}

func permissionedContinuationGameArgs(
	prestate common.Hash,
	proposer common.Address,
	challenger common.Address,
) []byte {
	args := permissionlessContinuationGameArgs(prestate)
	args = append(args, proposer.Bytes()...)
	return append(args, challenger.Bytes()...)
}

func superPermissionedContinuationGameArgs(proposer common.Address) []byte {
	args := append([]byte{}, common.Address{0xc2}.Bytes()...)
	return append(args, proposer.Bytes()...)
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
		require.Zero(t, fixture.backend.callsTo(fixture.dci.Opcm))
		require.Zero(t, fixture.backend.callsTo(fixture.backend.validator))
	})

	t.Run("SUPER_CANNON_KONA has a no-prestate fallback", func(t *testing.T) {
		fixture := newContinuationVerificationFixture(t, embedded.GameTypeSuperCannonKona)
		require.NoError(t, fixture.verify(t))
		require.Zero(t, fixture.dci.CannonAbsolutePrestate)
	})
}

func TestVerifyContinuationDeploymentFailures(t *testing.T) {
	tests := []struct {
		name    string
		wantErr string
		mutate  func(*testing.T, *continuationVerificationFixture)
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
			name:    "anchor root",
			wantErr: "starting anchor root",
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				f.backend.set(
					t,
					f.expected.AnchorStateRegistryProxy,
					continuationStartingAnchorMethod,
					nil,
					common.Hash{0xff},
					f.dci.StartingAnchorRoot.L2SequenceNumber,
				)
			},
		},
		{
			name:    "anchor sequence",
			wantErr: "starting anchor sequence",
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				f.backend.set(
					t,
					f.expected.AnchorStateRegistryProxy,
					continuationStartingAnchorMethod,
					nil,
					f.dci.StartingAnchorRoot.Root,
					big.NewInt(8),
				)
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
			name:    "selected prestate",
			wantErr: "selected prestate",
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				f.backend.set(
					t,
					f.expected.DisputeGameFactoryProxy,
					continuationGameArgsMethod,
					[]any{uint32(embedded.GameTypeCannonKona)},
					permissionlessContinuationGameArgs(common.Hash{0xff}),
				)
			},
		},
		{
			name:    "fallback prestate",
			wantErr: "fallback prestate",
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				f.backend.set(
					t,
					f.expected.DisputeGameFactoryProxy,
					continuationGameArgsMethod,
					[]any{uint32(embedded.GameTypePermissionedCannon)},
					permissionedContinuationGameArgs(common.Hash{0xff}, f.dci.Proposer, f.dci.Challenger),
				)
			},
		},
		{
			name:    "missing fallback prestate",
			wantErr: "fallback prestate",
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				f.backend.set(
					t,
					f.expected.DisputeGameFactoryProxy,
					continuationGameArgsMethod,
					[]any{uint32(embedded.GameTypePermissionedCannon)},
					[]byte{},
				)
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
			name:    "permissioned proposer",
			wantErr: "permissioned game proposer",
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				f.backend.set(
					t,
					f.expected.DisputeGameFactoryProxy,
					continuationGameArgsMethod,
					[]any{uint32(embedded.GameTypePermissionedCannon)},
					permissionedContinuationGameArgs(
						opcm.PermissionedCannonFallbackPrestatePlaceholder,
						common.Address{0xff},
						f.dci.Challenger,
					),
				)
			},
		},
		{
			name:    "permissioned challenger",
			wantErr: "permissioned game challenger",
			mutate: func(t *testing.T, f *continuationVerificationFixture) {
				f.backend.set(
					t,
					f.expected.DisputeGameFactoryProxy,
					continuationGameArgsMethod,
					[]any{uint32(embedded.GameTypePermissionedCannon)},
					permissionedContinuationGameArgs(
						opcm.PermissionedCannonFallbackPrestatePlaceholder,
						f.dci.Proposer,
						common.Address{0xff},
					),
				)
			},
		},
		{
			name:    "ProxyAdmin owner",
			wantErr: "OpChain ProxyAdmin owner",
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
			fixture := newContinuationVerificationFixture(t, embedded.GameTypeCannonKona)
			test.mutate(t, fixture)
			require.ErrorContains(t, fixture.verify(t), test.wantErr)
		})
	}
}
