package dsl

import (
	"fmt"
	"os"
	"sync/atomic"

	hdwallet "github.com/ethereum-optimism/go-ethereum-hdwallet"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

const (
	SaltEnvVar = "DEVSTACK_KEYS_SALT"
)

// HDWallet is a collection of deterministic accounts,
// generated from an underlying devkeys keyring,
// using the standard cross-chain user identities.
type HDWallet struct {
	commonImpl
	keys          devkeys.Keys
	nextUserIndex atomic.Uint64
	hdWalletName  string
}

func NewRandomHDWallet(t devtest.T, startIndex uint64) *HDWallet {
	mnemonic, err := hdwallet.NewMnemonic(256)
	require.NoError(t, err, "failed to generate mnemonic")
	return NewHDWallet(t, mnemonic, startIndex)
}

func NewHDWallet(t devtest.T, mnemonic string, startIndex uint64) *HDWallet {
	hd, err := devkeys.NewSaltedDevKeys(mnemonic, os.Getenv(SaltEnvVar))
	t.Require().NoError(err, "must have valid mnemonic")
	w := &HDWallet{
		commonImpl: commonFromT(t),
		keys:       hd,
		// We don't want to leak the mnemonic so easily with a string method,
		// but we do want to uniquely identify it to distinguish it from other wallets,
		// so we just hash it.
		hdWalletName: fmt.Sprintf("HDWallet(%s)", crypto.Keccak256Hash([]byte(mnemonic))),
	}
	w.nextUserIndex.Store(startIndex)
	return w
}

func (w *HDWallet) String() string {
	return w.hdWalletName
}

// NewKey creates a new chain-agnostic account identity
func (w *HDWallet) NewKey() *Key {
	newNextIndex := w.nextUserIndex.Add(1)
	thisIndex := newNextIndex - 1
	k := devkeys.UserKey(thisIndex)
	priv, err := w.keys.Secret(k)
	w.t.Require().NoError(err, "must generate user secret")
	key := NewKey(w.t, priv)
	// Log with address and HD path,
	// so we can easily reproduce the private key outside the test
	// (assuming access to the mnemonic).
	w.t.Logger().Debug("Creating user Key",
		"addr", key.Address(), "path", k.HDPath())
	return key
}

// NewEOA creates a new Key and wraps it with an EL node into a new EOA
func (w *HDWallet) NewEOA(el ELNode) *EOA {
	return w.NewKey().User(el)
}
