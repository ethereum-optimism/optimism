package cli

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

type pcdNamedAddress struct {
	name    string
	address common.Address
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

// This list matches the deployment markers from continuationContractAddresses in
// op-deployer/pkg/deployer/continue_verify.go. It omits shared implementations because
// OPCM bootstrap deploys them before prepare.
func pcdPreparedContractAddresses(prepared *state.PreparedDeployment) []pcdNamedAddress {
	var result []pcdNamedAddress
	for _, chain := range prepared.Chains {
		prefix := chain.ID.Hex() + "/"
		contracts := []pcdNamedAddress{
			{prefix + "OpChainProxyAdminImpl", chain.OpChainProxyAdminImpl},
			{prefix + "OptimismPortalProxy", chain.OptimismPortalProxy},
			{prefix + "AddressManagerImpl", chain.AddressManagerImpl},
			{prefix + "L1Erc721BridgeProxy", chain.L1Erc721BridgeProxy},
			{prefix + "SystemConfigProxy", chain.SystemConfigProxy},
			{prefix + "OptimismMintableErc20FactoryProxy", chain.OptimismMintableErc20FactoryProxy},
			{prefix + "L1StandardBridgeProxy", chain.L1StandardBridgeProxy},
			{prefix + "L1CrossDomainMessengerProxy", chain.L1CrossDomainMessengerProxy},
			{prefix + "EthLockboxProxy", chain.EthLockboxProxy},
			{prefix + "DisputeGameFactoryProxy", chain.DisputeGameFactoryProxy},
			{prefix + "AnchorStateRegistryProxy", chain.AnchorStateRegistryProxy},
			{prefix + "DelayedWethPermissionedGameProxy", chain.DelayedWethPermissionedGameProxy},
			{prefix + "DelayedWethPermissionlessGameProxy", chain.DelayedWethPermissionlessGameProxy},
			{prefix + "AltDAChallengeProxy", chain.AltDAChallengeProxy},
			{prefix + "L2OutputOracleProxy", chain.L2OutputOracleProxy},
		}
		for _, contract := range contracts {
			if contract.address != (common.Address{}) {
				result = append(result, contract)
			}
		}
	}
	return result
}
