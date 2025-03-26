package systemext

import (
	"crypto/ecdsa"

	"github.com/ethereum-optimism/optimism/devnet-sdk/system2"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum/go-ethereum/common"
)

type keyring struct {
	keys  *devkeys.MnemonicDevKeys
	setup *system2.Setup
}

var _ system2.L2Keys = (*keyring)(nil)

func (k *keyring) Secret(key devkeys.Key) *ecdsa.PrivateKey {
	pk, err := k.keys.Secret(key)
	k.setup.Require.NoError(err)
	return pk
}

func (k *keyring) Address(key devkeys.Key) common.Address {
	addr, err := k.keys.Address(key)
	k.setup.Require.NoError(err)
	return addr
}
