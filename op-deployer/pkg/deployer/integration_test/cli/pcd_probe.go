package cli

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	txintentbindings "github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

const (
	pcdPermissionlessGameArgsLength = 124
	pcdSuperPermissionedArgsLength  = 40
)

type pcdNamedAddress struct {
	name             string
	address          common.Address
	deploymentMarker bool
}

type pcdL1Probe struct {
	client   *ethclient.Client
	deployer common.Address
}

type pcdL1State struct {
	blockNumber  *big.Int
	blockHash    common.Hash
	latestNonce  uint64
	pendingNonce uint64
	code         map[pcdNamedAddress][]byte
}

type pcdLiveChainExpectation struct {
	chainID                       common.Hash
	portal                        common.Address
	disputeGameFactory            common.Address
	anchorStateRegistry           common.Address
	respectedGameType             embedded.GameType
	fallbackGameType              embedded.GameType
	respectedGameImplementation   common.Address
	fallbackGameImplementation    common.Address
	respectedGameAbsolutePrestate common.Hash
	startingProposalRoot          common.Hash
	startingProposalSequence      uint64
	proposalArtifactPaths         []string
}

func (p pcdL1Probe) read(t *testing.T, contractAddresses []pcdNamedAddress) pcdL1State {
	t.Helper()
	header, err := p.client.HeaderByNumber(t.Context(), nil)
	require.NoError(t, err)
	blockNumber := new(big.Int).Set(header.Number)
	latestNonce, err := p.client.NonceAt(t.Context(), p.deployer, blockNumber)
	require.NoError(t, err)
	pendingNonce, err := p.client.PendingNonceAt(t.Context(), p.deployer)
	require.NoError(t, err)

	code := make(map[pcdNamedAddress][]byte, len(contractAddresses))
	for _, contract := range contractAddresses {
		contractCode, err := p.client.CodeAt(t.Context(), contract.address, blockNumber)
		require.NoErrorf(t, err, "read %s at %s at L1 block %s", contract.name, contract.address, blockNumber)
		code[contract] = contractCode
	}

	return pcdL1State{
		blockNumber:  blockNumber,
		blockHash:    header.Hash(),
		latestNonce:  latestNonce,
		pendingNonce: pendingNonce,
		code:         code,
	}
}

func requireNoPCDDeploymentMutation(t *testing.T, baseline, prepared pcdL1State) {
	t.Helper()
	require.Equalf(
		t,
		baseline.latestNonce,
		prepared.latestNonce,
		"prepare changed the deployer latest nonce at pinned L1 block %s (%s)",
		prepared.blockNumber,
		prepared.blockHash,
	)
	require.Equalf(
		t,
		baseline.pendingNonce,
		prepared.pendingNonce,
		"prepare changed the deployer pending nonce when the pinned L1 block was %s (%s)",
		prepared.blockNumber,
		prepared.blockHash,
	)
	for contract, code := range prepared.code {
		require.Emptyf(
			t,
			code,
			"prepare created code for %s at %s at pinned L1 block %s (%s)",
			contract.name,
			contract.address,
			prepared.blockNumber,
			prepared.blockHash,
		)
	}
}

func (p pcdL1Probe) requireCompletedDeployment(
	t *testing.T,
	baseline pcdL1State,
	expectedNonceDelta uint64,
	contractAddresses []pcdNamedAddress,
	chains []pcdLiveChainExpectation,
) pcdL1State {
	t.Helper()
	completed := p.read(t, contractAddresses)
	require.Equalf(
		t,
		completed.latestNonce,
		completed.pendingNonce,
		"deployment has a pending deployer transaction at pinned L1 block %s (%s)",
		completed.blockNumber,
		completed.blockHash,
	)
	require.Equalf(
		t,
		baseline.latestNonce+expectedNonceDelta,
		completed.latestNonce,
		"bootstrap-to-completion deployer nonce delta differs at pinned L1 block %s (%s)",
		completed.blockNumber,
		completed.blockHash,
	)
	for contract, code := range completed.code {
		require.NotEmptyf(
			t,
			code,
			"deployment has no code for %s at %s at pinned L1 block %s (%s)",
			contract.name,
			contract.address,
			completed.blockNumber,
			completed.blockHash,
		)
	}
	for _, chain := range chains {
		p.requireLiveGameConfiguration(t, completed.blockNumber, chain)
		p.requireStartingProposal(t, completed.blockNumber, chain)
	}
	return completed
}

func (p pcdL1Probe) requireStartingProposal(
	t *testing.T,
	blockNumber *big.Int,
	expected pcdLiveChainExpectation,
) {
	t.Helper()
	registry := txintentbindings.NewBindings[txintentbindings.AnchorStateRegistry](
		txintentbindings.WithTo(expected.anchorStateRegistry),
	)
	observed, err := readPCDCallAtBlock(t.Context(), p.client, blockNumber, registry.GetAnchorRoot())
	require.NoErrorf(
		t,
		err,
		"read starting proposal for chain %s from registry %s at pinned L1 block %s; oracle artifacts: %v",
		expected.chainID.Hex(),
		expected.anchorStateRegistry,
		blockNumber,
		expected.proposalArtifactPaths,
	)
	require.Equalf(
		t,
		expected.startingProposalRoot,
		observed.Root,
		"starting proposal root differs for chain %s at registry %s at pinned L1 block %s: expected %s, observed %s; oracle artifacts: %v",
		expected.chainID.Hex(),
		expected.anchorStateRegistry,
		blockNumber,
		expected.startingProposalRoot,
		observed.Root,
		expected.proposalArtifactPaths,
	)
	require.Zero(
		t,
		new(big.Int).SetUint64(expected.startingProposalSequence).Cmp(observed.L2SequenceNumber),
		"starting proposal sequence differs for chain %s at registry %s at pinned L1 block %s: expected %d, observed %s; oracle artifacts: %v",
		expected.chainID.Hex(),
		expected.anchorStateRegistry,
		blockNumber,
		expected.startingProposalSequence,
		observed.L2SequenceNumber,
		expected.proposalArtifactPaths,
	)
}

func (p pcdL1Probe) requireLiveGameConfiguration(
	t *testing.T,
	blockNumber *big.Int,
	expected pcdLiveChainExpectation,
) {
	t.Helper()
	portal := txintentbindings.NewBindings[txintentbindings.OptimismPortal2](
		txintentbindings.WithTo(expected.portal),
	)
	observedGameType, err := readPCDCallAtBlock(t.Context(), p.client, blockNumber, portal.RespectedGameType())
	require.NoErrorf(
		t,
		err,
		"read respected game type for chain %s from portal %s at pinned L1 block %s",
		expected.chainID.Hex(),
		expected.portal,
		blockNumber,
	)
	require.Equalf(
		t,
		uint32(expected.respectedGameType),
		observedGameType,
		"respected game type differs for chain %s at portal %s at pinned L1 block %s",
		expected.chainID.Hex(),
		expected.portal,
		blockNumber,
	)

	factory := txintentbindings.NewDisputeGameFactory(
		txintentbindings.WithTo(expected.disputeGameFactory),
	)
	respectedImplementation, err := readPCDCallAtBlock(
		t.Context(),
		p.client,
		blockNumber,
		factory.GameImpls(uint32(expected.respectedGameType)),
	)
	require.NoErrorf(
		t,
		err,
		"read game type %d implementation for chain %s from factory %s at pinned L1 block %s",
		expected.respectedGameType,
		expected.chainID.Hex(),
		expected.disputeGameFactory,
		blockNumber,
	)
	require.Equalf(
		t,
		expected.respectedGameImplementation,
		respectedImplementation,
		"game type %d implementation differs for chain %s at factory %s at pinned L1 block %s",
		expected.respectedGameType,
		expected.chainID.Hex(),
		expected.disputeGameFactory,
		blockNumber,
	)
	require.NotEqualf(
		t,
		common.Address{},
		respectedImplementation,
		"game type %d implementation is absent for chain %s at factory %s at pinned L1 block %s",
		expected.respectedGameType,
		expected.chainID.Hex(),
		expected.disputeGameFactory,
		blockNumber,
	)

	respectedArgs, err := readPCDCallAtBlock(
		t.Context(),
		p.client,
		blockNumber,
		factory.GameArgs(uint32(expected.respectedGameType)),
	)
	require.NoErrorf(
		t,
		err,
		"read game type %d arguments for chain %s from factory %s at pinned L1 block %s",
		expected.respectedGameType,
		expected.chainID.Hex(),
		expected.disputeGameFactory,
		blockNumber,
	)
	require.Lenf(
		t,
		respectedArgs,
		pcdPermissionlessGameArgsLength,
		"game type %d arguments have the wrong length for chain %s at factory %s at pinned L1 block %s",
		expected.respectedGameType,
		expected.chainID.Hex(),
		expected.disputeGameFactory,
		blockNumber,
	)
	observedPrestate := common.BytesToHash(respectedArgs[:common.HashLength])
	require.Equalf(
		t,
		expected.respectedGameAbsolutePrestate,
		observedPrestate,
		"game type %d absolute prestate differs for chain %s at factory %s at pinned L1 block %s",
		expected.respectedGameType,
		expected.chainID.Hex(),
		expected.disputeGameFactory,
		blockNumber,
	)

	fallbackImplementation, err := readPCDCallAtBlock(
		t.Context(),
		p.client,
		blockNumber,
		factory.GameImpls(uint32(expected.fallbackGameType)),
	)
	require.NoErrorf(
		t,
		err,
		"read game type %d implementation for chain %s from factory %s at pinned L1 block %s",
		expected.fallbackGameType,
		expected.chainID.Hex(),
		expected.disputeGameFactory,
		blockNumber,
	)
	require.Equalf(
		t,
		expected.fallbackGameImplementation,
		fallbackImplementation,
		"game type %d implementation differs for chain %s at factory %s at pinned L1 block %s",
		expected.fallbackGameType,
		expected.chainID.Hex(),
		expected.disputeGameFactory,
		blockNumber,
	)
	require.NotEqualf(
		t,
		common.Address{},
		fallbackImplementation,
		"game type %d implementation is absent for chain %s at factory %s at pinned L1 block %s",
		expected.fallbackGameType,
		expected.chainID.Hex(),
		expected.disputeGameFactory,
		blockNumber,
	)

	fallbackArgs, err := readPCDCallAtBlock(
		t.Context(),
		p.client,
		blockNumber,
		factory.GameArgs(uint32(expected.fallbackGameType)),
	)
	require.NoErrorf(
		t,
		err,
		"read game type %d arguments for chain %s from factory %s at pinned L1 block %s",
		expected.fallbackGameType,
		expected.chainID.Hex(),
		expected.disputeGameFactory,
		blockNumber,
	)
	// SuperPermissionedDisputeGame has no absolutePrestate getter. Its upgrade config
	// encodes only a proposer, and its factory arguments add only the AnchorStateRegistry.
	// See dispute/SuperPermissionedDisputeGame.sol, upgrade/embedded/upgrade.go, and
	// dispute/lib/LibGameArgs.sol in packages/contracts-bedrock/src.
	require.Lenf(
		t,
		fallbackArgs,
		pcdSuperPermissionedArgsLength,
		"game type %d arguments contain an unexpected prestate slot for chain %s at factory %s at pinned L1 block %s",
		expected.fallbackGameType,
		expected.chainID.Hex(),
		expected.disputeGameFactory,
		blockNumber,
	)
}

func readPCDCallAtBlock[T any](
	ctx context.Context,
	client *ethclient.Client,
	blockNumber *big.Int,
	call txintentbindings.TypedCall[T],
) (T, error) {
	var zero T
	if blockNumber == nil {
		return zero, errors.New("read PCD call requires a pinned block number")
	}
	target, err := call.To()
	if err != nil {
		return zero, fmt.Errorf("resolve call target: %w", err)
	}
	data, err := call.EncodeInput()
	if err != nil {
		return zero, fmt.Errorf("encode call input: %w", err)
	}
	result, err := client.CallContract(ctx, ethereum.CallMsg{To: target, Data: data}, blockNumber)
	if err != nil {
		return zero, fmt.Errorf("call %s at block %s: %w", *target, blockNumber, err)
	}
	decoded, err := call.DecodeOutput(result)
	if err != nil {
		return zero, fmt.Errorf("decode call result from %s at block %s: %w", *target, blockNumber, err)
	}
	return decoded, nil
}

// This list matches continuationContractAddresses in op-deployer/pkg/deployer/continue_verify.go.
// Shared implementations are not deployment markers because OPCM bootstrap deploys them before prepare.
func pcdPredictedContractAddresses(prepared *state.PreparedDeployment) []pcdNamedAddress {
	var result []pcdNamedAddress
	for _, chain := range prepared.Chains {
		prefix := chain.ID.Hex() + "/"
		contracts := []pcdNamedAddress{
			{prefix + "OpChainProxyAdminImpl", chain.OpChainProxyAdminImpl, true},
			{prefix + "OptimismPortalProxy", chain.OptimismPortalProxy, true},
			{prefix + "AddressManagerImpl", chain.AddressManagerImpl, true},
			{prefix + "L1Erc721BridgeProxy", chain.L1Erc721BridgeProxy, true},
			{prefix + "SystemConfigProxy", chain.SystemConfigProxy, true},
			{prefix + "OptimismMintableErc20FactoryProxy", chain.OptimismMintableErc20FactoryProxy, true},
			{prefix + "L1StandardBridgeProxy", chain.L1StandardBridgeProxy, true},
			{prefix + "L1CrossDomainMessengerProxy", chain.L1CrossDomainMessengerProxy, true},
			{prefix + "EthLockboxProxy", chain.EthLockboxProxy, true},
			{prefix + "DisputeGameFactoryProxy", chain.DisputeGameFactoryProxy, true},
			{prefix + "AnchorStateRegistryProxy", chain.AnchorStateRegistryProxy, true},
			{prefix + "FaultDisputeGameImpl", chain.FaultDisputeGameImpl, false},
			{prefix + "FaultDisputeGameCannonKonaImpl", chain.FaultDisputeGameCannonKonaImpl, false},
			{prefix + "PermissionedDisputeGameImpl", chain.PermissionedDisputeGameImpl, false},
			{prefix + "SuperFaultDisputeGameImpl", chain.SuperFaultDisputeGameImpl, false},
			{prefix + "SuperPermissionedDisputeGameImpl", chain.SuperPermissionedDisputeGameImpl, false},
			{prefix + "DelayedWethPermissionedGameProxy", chain.DelayedWethPermissionedGameProxy, true},
			{prefix + "DelayedWethPermissionlessGameProxy", chain.DelayedWethPermissionlessGameProxy, true},
			{prefix + "L2OutputOracleProxy", chain.L2OutputOracleProxy, true},
		}
		for _, contract := range contracts {
			if contract.address != (common.Address{}) {
				result = append(result, contract)
			}
		}
	}
	return result
}

func pcdDeploymentMarkerAddresses(predicted []pcdNamedAddress) []pcdNamedAddress {
	result := make([]pcdNamedAddress, 0, len(predicted))
	for _, contract := range predicted {
		if contract.deploymentMarker {
			result = append(result, contract)
		}
	}
	return result
}
