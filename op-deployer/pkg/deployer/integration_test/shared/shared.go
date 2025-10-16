package shared

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// AddrFor generates an address for a given role
func AddrFor(t *testing.T, dk *devkeys.MnemonicDevKeys, key devkeys.Key) common.Address {
	addr, err := dk.Address(key)
	require.NoError(t, err)
	return addr
}

func NewChainIntent(t *testing.T, dk *devkeys.MnemonicDevKeys, l1ChainID *big.Int, l2ChainID *uint256.Int, gasLimit uint64) *state.ChainIntent {
	return &state.ChainIntent{
		ID:                         l2ChainID.Bytes32(),
		BaseFeeVaultRecipient:      AddrFor(t, dk, devkeys.BaseFeeVaultRecipientRole.Key(l1ChainID)),
		L1FeeVaultRecipient:        AddrFor(t, dk, devkeys.L1FeeVaultRecipientRole.Key(l1ChainID)),
		SequencerFeeVaultRecipient: AddrFor(t, dk, devkeys.SequencerFeeVaultRecipientRole.Key(l1ChainID)),
		Eip1559DenominatorCanyon:   standard.Eip1559DenominatorCanyon,
		Eip1559Denominator:         standard.Eip1559Denominator,
		Eip1559Elasticity:          standard.Eip1559Elasticity,
		GasLimit:                   gasLimit,
		Roles: state.ChainRoles{
			L1ProxyAdminOwner: AddrFor(t, dk, devkeys.L2ProxyAdminOwnerRole.Key(l1ChainID)),
			L2ProxyAdminOwner: AddrFor(t, dk, devkeys.L2ProxyAdminOwnerRole.Key(l1ChainID)),
			SystemConfigOwner: AddrFor(t, dk, devkeys.SystemConfigOwner.Key(l1ChainID)),
			UnsafeBlockSigner: AddrFor(t, dk, devkeys.SequencerP2PRole.Key(l1ChainID)),
			Batcher:           AddrFor(t, dk, devkeys.BatcherRole.Key(l1ChainID)),
			Proposer:          AddrFor(t, dk, devkeys.ProposerRole.Key(l1ChainID)),
			Challenger:        AddrFor(t, dk, devkeys.ChallengerRole.Key(l1ChainID)),
		},
	}
}

func NewIntent(
	t *testing.T,
	l1ChainID *big.Int,
	dk *devkeys.MnemonicDevKeys,
	l2ChainID *uint256.Int,
	l1Loc *artifacts.Locator,
	l2Loc *artifacts.Locator,
	gasLimit uint64,
) (*state.Intent, *state.State) {
	intent := &state.Intent{
		ConfigType: state.IntentTypeCustom,
		L1ChainID:  l1ChainID.Uint64(),
		SuperchainRoles: &addresses.SuperchainRoles{
			SuperchainProxyAdminOwner: AddrFor(t, dk, devkeys.L1ProxyAdminOwnerRole.Key(l1ChainID)),
			ProtocolVersionsOwner:     AddrFor(t, dk, devkeys.SuperchainDeployerKey.Key(l1ChainID)),
			SuperchainGuardian:        AddrFor(t, dk, devkeys.SuperchainConfigGuardianKey.Key(l1ChainID)),
			Challenger:                AddrFor(t, dk, devkeys.ChallengerRole.Key(l1ChainID)),
		},
		FundDevAccounts:    false,
		L1ContractsLocator: l1Loc,
		L2ContractsLocator: l2Loc,
		Chains: []*state.ChainIntent{
			NewChainIntent(t, dk, l1ChainID, l2ChainID, gasLimit),
		},
	}
	st := &state.State{
		Version: 1,
	}
	return intent, st
}

// DefaultPrivkey returns the default private key for testing
func DefaultPrivkey(t *testing.T) (string, *ecdsa.PrivateKey, *devkeys.MnemonicDevKeys) {
	pkHex := "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	pk, err := crypto.HexToECDSA(pkHex)
	require.NoError(t, err)

	dk, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)

	return pkHex, pk, dk
}
