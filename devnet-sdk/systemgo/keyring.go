package systemgo

import (
	"crypto/ecdsa"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/devnet-sdk/system2"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
)

type keyring struct {
	keys    devkeys.Keys
	require *require.Assertions
}

var _ system2.L2Keys = (*keyring)(nil)

func (k *keyring) Secret(key devkeys.Key) *ecdsa.PrivateKey {
	pk, err := k.keys.Secret(key)
	k.require.NoError(err)
	return pk
}

func (k *keyring) Address(key devkeys.Key) common.Address {
	addr, err := k.keys.Address(key)
	k.require.NoError(err)
	return addr
}
